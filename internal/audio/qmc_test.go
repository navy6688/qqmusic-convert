package audio

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"unicode/utf16"
)

func TestMapReferenceVector(t *testing.T) {
	cipher, err := NewCipher([]byte("ABCDEFGHIJKLMNOP"))
	if err != nil {
		t.Fatal(err)
	}
	actual := make([]byte, 16)
	cipher.Transform(actual, 0)
	expected := []byte{0x3f, 0x8a, 0xc1, 0x49, 0x3f, 0x49, 0xc1, 0x8a, 0x3f, 0x8a, 0xc1, 0x49, 0x3f, 0x49, 0xc1, 0x8a}
	if !bytes.Equal(actual, expected) {
		t.Fatalf("map vector = %x, expected %x", actual, expected)
	}
}

func TestChunkOffsetsAndSymmetry(t *testing.T) {
	for _, size := range []int{8, 32, 512} {
		t.Run(fmt.Sprintf("key_%d_bytes", size), func(t *testing.T) {
			key := make([]byte, size)
			for i := range key {
				key[i] = byte(i*37 + 11)
			}
			cipher, err := NewCipher(key)
			if err != nil {
				t.Fatal(err)
			}
			original := make([]byte, 90000)
			for i := range original {
				original[i] = byte(i)
			}
			whole := bytes.Clone(original)
			cipher.Transform(whole, 0)
			chunks := bytes.Clone(original)
			for offset := 0; offset < len(chunks); {
				end := min(offset+777, len(chunks))
				cipher.Transform(chunks[offset:end], int64(offset))
				offset = end
			}
			if !bytes.Equal(whole, chunks) {
				t.Fatal("absolute offset chunks differ from one-shot transform")
			}
			cipher.Transform(chunks, 0)
			if !bytes.Equal(chunks, original) {
				t.Fatal("cipher is not symmetric")
			}
		})
	}
}

func TestRawKeyIsNotMistakenForTEA(t *testing.T) {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i*37 + 11)
	}
	actual, err := DeriveKey(base64.StdEncoding.EncodeToString(raw))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, actual) {
		t.Fatal("raw key was incorrectly unwrapped")
	}
}

func TestInvalidKeys(t *testing.T) {
	for _, encoded := range []string{"", "%%%", base64.StdEncoding.EncodeToString([]byte("short"))} {
		if _, err := DeriveKey(encoded); err == nil {
			t.Fatalf("accepted invalid key of length %d", len(encoded))
		}
	}
}

func TestMusicexFooterValidation(t *testing.T) {
	footer := make([]byte, 192)
	write := func(offset int, value string) {
		for i, unit := range utf16.Encode([]rune(value)) {
			binary.LittleEndian.PutUint16(footer[offset+2*i:], unit)
		}
	}
	write(12, "001234567890ab")
	write(72, "F0M00000000000001.mflac")
	binary.LittleEndian.PutUint32(footer[176:], 192)
	binary.LittleEndian.PutUint32(footer[180:], 1)
	copy(footer[184:], "musicex\x00")
	info, err := ParseFooter(footer, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if info.AudioLength != 808 || info.Resource != "F0M00000000000001.mflac" {
		t.Fatalf("unexpected footer %+v", info)
	}
	binary.LittleEndian.PutUint32(footer[180:], 2)
	if _, err := ParseFooter(footer, 1000); err == nil {
		t.Fatal("accepted version 2")
	}
	binary.LittleEndian.PutUint32(footer[180:], 1)
	binary.LittleEndian.PutUint32(footer[176:], 1001)
	if _, err := ParseFooter(footer, 1000); err == nil {
		t.Fatal("accepted oversized footer")
	}
}

func teaEncryptFixture(plaintext, secret []byte) []byte {
	padding := (8 - (len(plaintext)+10)%8) % 8
	padded := make([]byte, 1+padding+2+len(plaintext)+7)
	padded[0] = byte(0xa0 | padding)
	for i := 1; i < 1+padding+2; i++ {
		padded[i] = byte(i*11 + 3)
	}
	copy(padded[1+padding+2:], plaintext)
	var key [4]uint32
	for i := range 4 {
		key[i] = binary.BigEndian.Uint32(secret[4*i:])
	}
	previousCipher, previousIntermediate := make([]byte, 8), make([]byte, 8)
	ciphertext := make([]byte, len(padded))
	for offset := 0; offset < len(padded); offset += 8 {
		intermediate := make([]byte, 8)
		for i := range 8 {
			intermediate[i] = padded[offset+i] ^ previousCipher[i]
		}
		left, right := binary.BigEndian.Uint32(intermediate), binary.BigEndian.Uint32(intermediate[4:])
		sum := uint32(0)
		for range 16 {
			sum += 0x9e3779b9
			left += ((right << 4) + key[0]) ^ (right + sum) ^ ((right >> 5) + key[1])
			right += ((left << 4) + key[2]) ^ (left + sum) ^ ((left >> 5) + key[3])
		}
		encrypted := make([]byte, 8)
		binary.BigEndian.PutUint32(encrypted, left)
		binary.BigEndian.PutUint32(encrypted[4:], right)
		for i := range 8 {
			ciphertext[offset+i] = encrypted[i] ^ previousIntermediate[i]
		}
		copy(previousCipher, ciphertext[offset:offset+8])
		copy(previousIntermediate, intermediate)
	}
	return ciphertext
}

func TestEncV1AndEncV2KeyDerivation(t *testing.T) {
	head := []byte("12345678")
	body := []byte("synthetic payload for key")
	key := make([]byte, 16)
	for i := range 8 {
		key[2*i] = simpleKey[i]
		key[2*i+1] = head[i]
	}
	encV1 := append(append([]byte(nil), head...), teaEncryptFixture(body, key)...)
	expected := append(append([]byte(nil), head...), body...)
	v1 := base64.StdEncoding.EncodeToString(encV1)
	v2Inner := teaEncryptFixture([]byte(v1), v2Key2)
	v2Outer := teaEncryptFixture(v2Inner, v2Key1)
	v2 := base64.StdEncoding.EncodeToString(append([]byte("QQMusic EncV2,Key:"), v2Outer...))
	for _, encoded := range []string{v1, v2} {
		actual, err := DeriveKey(encoded)
		if err != nil || !bytes.Equal(actual, expected) {
			t.Fatalf("wrapped synthetic key derivation failed: length %d, error %v", len(actual), err)
		}
	}
}

func BenchmarkMapChunk(b *testing.B) {
	cipher, err := NewCipher([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		b.Fatal(err)
	}
	chunk := make([]byte, 256*1024)
	b.SetBytes(int64(len(chunk)))
	b.ResetTimer()
	for range b.N {
		cipher.Transform(chunk, 0)
	}
}

func TestRC4MatchesPinnedUpstreamVectors(t *testing.T) {
	encoded, err := os.ReadFile("testdata/qmc2_rc4_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []struct {
		Offset   int64  `json:"offset"`
		Expected string `json:"expected"`
	}
	if err = json.Unmarshal(encoded, &vectors); err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 512)
	for i := range key {
		key[i] = byte(i*37 + 11)
	}
	cipher, err := NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	for _, vector := range vectors {
		expected, err := hex.DecodeString(vector.Expected)
		if err != nil {
			t.Fatal(err)
		}
		actual := make([]byte, len(expected))
		cipher.Transform(actual, vector.Offset)
		if !bytes.Equal(actual, expected) {
			t.Fatalf("upstream RC4 mismatch at offset %d", vector.Offset)
		}
	}
}
