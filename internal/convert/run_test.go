package convert

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"

	"qqmusic-convert/internal/audio"
	"qqmusic-convert/internal/media"
)

type fixtureKeys struct {
	encoded string
	calls   int
	before  func()
}

func (keys *fixtureKeys) Key(ctx context.Context, footer audio.Footer) (string, error) {
	keys.calls++
	if keys.before != nil {
		keys.before()
	}
	return keys.encoded, nil
}

func fixtureTools(t *testing.T) media.Tools {
	t.Helper()
	tools, err := media.Prepare(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return tools
}

func syntheticFLAC(t *testing.T, root string, tools media.Tools) []byte {
	t.Helper()
	path := filepath.Join(root, "fixture.flac")
	command := media.Command(context.Background(), tools.FFmpeg, "-hide_banner", "-v", "error", "-n", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=0.25", "-ac", "2", "-c:a", "flac", "-sample_fmt", "s16", path)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("create audio fixture: %v; %s", err, output)
	}
	plaintext, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return plaintext
}

func encryptedFixture(t *testing.T, root, name string, plaintext, key []byte) string {
	t.Helper()
	cipher, err := audio.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	encrypted := append([]byte(nil), plaintext...)
	cipher.Transform(encrypted, 0)
	footer := make([]byte, 192)
	field := func(offset int, value string) {
		for i, unit := range utf16.Encode([]rune(value)) {
			binary.LittleEndian.PutUint16(footer[offset+2*i:], unit)
		}
	}
	field(12, "001234567890ab")
	field(72, "F0M00000000000001.mflac")
	binary.LittleEndian.PutUint32(footer[176:], 192)
	binary.LittleEndian.PutUint32(footer[180:], 1)
	copy(footer[184:], "musicex\x00")
	encrypted = append(encrypted, footer...)
	path := filepath.Join(root, name+".mflac")
	if err = os.WriteFile(path, encrypted, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEncryptedFLACConversionIntegrityAndOriginalPreservation(t *testing.T) {
	root := t.TempDir()
	tools := fixtureTools(t)
	plain := syntheticFLAC(t, root, tools)
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i*37 + 11)
	}
	source := encryptedFixture(t, root, "加密歌曲", append(plain, []byte("non audio trailer")...), key)
	before, err := sourceHash(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	keys := &fixtureKeys{encoded: base64.StdEncoding.EncodeToString(key)}
	output := filepath.Join(root, "MP3", "加密歌曲.mp3")
	result := ConvertFile(context.Background(), Job{Source: source, Destination: output, Extension: ".mflac"}, tools, keys, nil)
	if result.Status != "converted" || result.Integrity.Method != "pcm_md5_and_sample_count" {
		t.Fatalf("conversion = %+v", result)
	}
	if keys.calls != 1 {
		t.Fatal("resource key was not requested exactly once")
	}
	after, err := sourceHash(context.Background(), source)
	if err != nil || after != before {
		t.Fatal("original encrypted fixture was modified")
	}
	again := ConvertFile(context.Background(), result.Job, media.Tools{}, nil, nil)
	if again.Status != "skipped" {
		t.Fatalf("existing output was not skipped: %+v", again)
	}
	if matches, err := filepath.Glob(filepath.Join(filepath.Dir(output), ".qqmusic-convert-*")); err != nil || len(matches) != 0 {
		t.Fatal("temporary audio was not cleaned up")
	}
}

func TestBadFLACMD5DoesNotPublishMP3(t *testing.T) {
	root := t.TempDir()
	tools := fixtureTools(t)
	plain := syntheticFLAC(t, root, tools)
	plain[26] ^= 1
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i*37 + 11)
	}
	source := encryptedFixture(t, root, "坏校验", plain, key)
	output := filepath.Join(root, "bad.mp3")
	result := ConvertFile(context.Background(), Job{Source: source, Destination: output, Extension: ".mflac"}, tools, &fixtureKeys{encoded: base64.StdEncoding.EncodeToString(key)}, nil)
	if result.Status != "failed" {
		t.Fatal("invalid FLAC checksum was accepted")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("invalid audio was published")
	}
}

func TestChangedSourceDoesNotPublishMP3(t *testing.T) {
	root := t.TempDir()
	tools := fixtureTools(t)
	plain := syntheticFLAC(t, root, tools)
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i*37 + 11)
	}
	source := encryptedFixture(t, root, "源文件变化", plain, key)
	output := filepath.Join(root, "changed.mp3")
	keys := &fixtureKeys{encoded: base64.StdEncoding.EncodeToString(key), before: func() {
		file, err := os.OpenFile(source, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = file.Write([]byte("changed")); err != nil {
			t.Fatal(err)
		}
		if err = file.Close(); err != nil {
			t.Fatal(err)
		}
	}}
	result := ConvertFile(context.Background(), Job{Source: source, Destination: output, Extension: ".mflac"}, tools, keys, nil)
	if result.Status != "failed" {
		t.Fatal("source modification was not detected")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("output published after source changed")
	}
}

func TestContextCancellationAndExclusivePublication(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	output := filepath.Join(root, "output")
	if err := os.WriteFile(source, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := publish(context.Background(), source, output); !errors.Is(err, os.ErrExist) {
		t.Fatalf("publication must refuse an existing file, got %v", err)
	}
	preserved, err := os.ReadFile(output)
	if err != nil || string(preserved) != "existing" {
		t.Fatal("existing output changed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = sourceHash(ctx, source); !errors.Is(err, context.Canceled) {
		t.Fatal("hashing ignored cancellation")
	}
	target := filepath.Join(root, "canceled.mp3")
	if err = publish(ctx, source, target); !errors.Is(err, context.Canceled) {
		t.Fatal("publication ignored cancellation")
	}
	if _, err = os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("canceled publication left partial output")
	}
}

func TestPlainAudioDoesNotNeedQQMusicSession(t *testing.T) {
	root := t.TempDir()
	tools := fixtureTools(t)
	syntheticFLAC(t, root, tools)
	source := filepath.Join(root, "fixture.flac")
	result := ConvertFile(context.Background(), Job{Source: source, Destination: filepath.Join(root, "plain.mp3"), Extension: ".flac"}, tools, nil, nil)
	if result.Status != "converted" {
		t.Fatalf("plain audio failed without session: %+v", result)
	}
}

type resourceFixtureKeys struct {
	original     string
	right, wrong string
	requested    []string
}

func (keys *resourceFixtureKeys) Key(ctx context.Context, footer audio.Footer) (string, error) {
	keys.requested = append(keys.requested, footer.Resource)
	if footer.Resource == keys.original {
		return keys.right, nil
	}
	return keys.wrong, nil
}

func TestAppendedAliasFooterUsesOriginalResourceAndExcludesEveryFooter(t *testing.T) {
	root := t.TempDir()
	tools := fixtureTools(t)
	plain := syntheticFLAC(t, root, tools)
	key := make([]byte, 32)
	wrong := make([]byte, 32)
	for i := range key {
		key[i] = byte(i*37 + 11)
		wrong[i] = byte(i*19 + 7)
	}
	source := encryptedFixture(t, root, "cached-alias", plain, key)
	bytes, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	alias := append([]byte(nil), bytes[len(bytes)-192:]...)
	for i := range alias[72:140] {
		alias[72+i] = 0
	}
	for i, unit := range utf16.Encode([]rune("F0M00000000000002.mflac")) {
		binary.LittleEndian.PutUint16(alias[72+2*i:], unit)
	}
	bytes = append(bytes, alias...)
	if err = os.WriteFile(source, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	keys := &resourceFixtureKeys{original: "F0M00000000000001.mflac", right: base64.StdEncoding.EncodeToString(key), wrong: base64.StdEncoding.EncodeToString(wrong)}
	output := filepath.Join(root, "original-resource.mp3")
	result := ConvertFile(context.Background(), Job{Source: source, Destination: output, Extension: ".mflac"}, tools, keys, nil)
	if result.Status != "converted" {
		t.Fatalf("cached file with a newer alias footer must convert, got %s: %s; requested resources=%v", result.Status, result.Message, keys.requested)
	}
	if len(keys.requested) != 1 || keys.requested[0] != keys.original {
		t.Fatalf("requested wrong resource: %v", keys.requested)
	}
	if result.Integrity.Warnings {
		t.Fatal("plaintext metadata footers were included in encrypted audio")
	}
}
