package audio

import (
	"bytes"
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

func footerFixture(mid, resource string) []byte {
	footer := make([]byte, 192)
	write := func(offset int, value string) {
		for i, unit := range utf16.Encode([]rune(value)) {
			binary.LittleEndian.PutUint16(footer[offset+2*i:], unit)
		}
	}
	write(12, mid)
	write(72, resource)
	binary.LittleEndian.PutUint32(footer[176:], 192)
	binary.LittleEndian.PutUint32(footer[180:], 1)
	copy(footer[184:], "musicex\x00")
	return footer
}

func TestReadFooterChainChoosesInnermostResource(t *testing.T) {
	original := footerFixture("001234567890ab", "F0M00000000000001.mflac")
	alias := footerFixture("009876543210ab", "F0M00000000000002.mflac")
	for _, count := range []int{1, 2, 3} {
		t.Run(string(rune('0'+count)), func(t *testing.T) {
			ciphertext := bytes.Repeat([]byte{0x5a}, 1000)
			file := append(ciphertext, original...)
			for range count - 1 {
				file = append(file, alias...)
			}
			parsed, err := ReadFooter(bytes.NewReader(file), int64(len(file)))
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Resource != "F0M00000000000001.mflac" || parsed.AudioLength != 1000 || parsed.MetadataLayers != count {
				t.Fatalf("incorrect original resource/boundary %+v", parsed)
			}
		})
	}
}

func TestReadFooterChainValidatesInnerLayer(t *testing.T) {
	original := footerFixture("001234567890ab", "F0M00000000000001.mflac")
	binary.LittleEndian.PutUint32(original[180:], 2)
	file := append(bytes.Repeat([]byte{0x5a}, 1000), original...)
	file = append(file, footerFixture("009876543210ab", "F0M00000000000002.mflac")...)
	if _, err := ReadFooter(bytes.NewReader(file), int64(len(file))); err == nil {
		t.Fatal("invalid inner footer was silently ignored")
	}
}

func TestReadFooterChainRejectsExcessiveSize(t *testing.T) {
	footer := footerFixture("001234567890ab", "F0M00000000000001.mflac")
	binary.LittleEndian.PutUint32(footer[176:], 2*1024*1024)
	file := append(bytes.Repeat([]byte{0x5a}, 1000), footer...)
	if _, err := ReadFooter(bytes.NewReader(file), int64(len(file))); err == nil {
		t.Fatal("accepted oversized footer")
	}
}
