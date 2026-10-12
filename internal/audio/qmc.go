// QMC2 algorithm ported from QM Unlock, MIT; see licenses/QMUnlock-MIT.txt.
package audio

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

var simpleKey = [8]byte{0x69, 0x56, 0x46, 0x38, 0x2b, 0x20, 0x15, 0x0b}
var v2Key1 = []byte("386ZJY!@#*$%^&)(")
var v2Key2 = []byte("**#!(#$%&^a1cZ,T")

func decodeBase64(encoded string) ([]byte, error) {
	encoded = strings.TrimSpace(encoded)
	if len(encoded)%4 != 0 {
		encoded += strings.Repeat("=", 4-len(encoded)%4)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.New("音频密钥不是有效的 Base64（密钥已隐藏）")
	}
	return decoded, nil
}

// DeriveKey accepts EncV2, EncV1 and already-unwrapped raw resource keys.
func DeriveKey(encoded string) ([]byte, error) {
	decoded, err := decodeBase64(encoded)
	if err != nil {
		return nil, err
	}
	prefix := []byte("QQMusic EncV2,Key:")
	if bytes.HasPrefix(decoded, prefix) {
		inner, err := teaDecode(decoded[len(prefix):], v2Key1)
		if err != nil {
			return nil, fmt.Errorf("EncV2 第一层密钥还原失败: %w", err)
		}
		inner, err = teaDecode(inner, v2Key2)
		if err != nil {
			return nil, fmt.Errorf("EncV2 第二层密钥还原失败: %w", err)
		}
		decoded, err = decodeBase64(string(inner))
		if err != nil {
			return nil, err
		}
	}
	if len(decoded) < 8 {
		return nil, fmt.Errorf("音频密钥长度不足：%d（内容已隐藏）", len(decoded))
	}
	if len(decoded) == 8 {
		return decoded, nil
	}
	teaKey := make([]byte, 16)
	for i := range 8 {
		teaKey[2*i] = simpleKey[i]
		teaKey[2*i+1] = decoded[i]
	}
	body, err := teaDecode(decoded[8:], teaKey)
	if err != nil {
		return decoded, nil
	}
	return append(append([]byte(nil), decoded[:8]...), body...), nil
}

func teaBlock(block []byte, key [4]uint32) {
	left, right := binary.BigEndian.Uint32(block), binary.BigEndian.Uint32(block[4:])
	sum := uint32(0xe3779b90)
	for range 16 {
		right -= ((left << 4) + key[2]) ^ (left + sum) ^ ((left >> 5) + key[3])
		left -= ((right << 4) + key[0]) ^ (right + sum) ^ ((right >> 5) + key[1])
		sum -= 0x9e3779b9
	}
	binary.BigEndian.PutUint32(block, left)
	binary.BigEndian.PutUint32(block[4:], right)
}

func teaDecode(ciphertext, secret []byte) ([]byte, error) {
	if len(secret) != 16 || len(ciphertext) < 16 || len(ciphertext)%8 != 0 {
		return nil, errors.New("TEA 密钥或密文长度不合法")
	}
	var key [4]uint32
	for i := range 4 {
		key[i] = binary.BigEndian.Uint32(secret[4*i:])
	}
	current := append([]byte(nil), ciphertext[:8]...)
	teaBlock(current, key)
	padding := int(current[0] & 7)
	length := len(ciphertext) - 1 - padding - 2 - 7
	if length < 0 {
		return nil, errors.New("TEA 填充长度不合法")
	}
	previous := make([]byte, 8)
	cipherBlock := append([]byte(nil), ciphertext[:8]...)
	inputPosition, position := 8, 1+padding
	nextBlock := func() error {
		if inputPosition+8 > len(ciphertext) {
			return errors.New("TEA 密文提前结束")
		}
		copy(previous, cipherBlock)
		copy(cipherBlock, ciphertext[inputPosition:inputPosition+8])
		for i := range 8 {
			current[i] ^= cipherBlock[i]
		}
		teaBlock(current, key)
		inputPosition += 8
		position = 0
		return nil
	}
	readByte := func() (byte, error) {
		if position >= 8 {
			if err := nextBlock(); err != nil {
				return 0, err
			}
		}
		value := current[position] ^ previous[position]
		position++
		return value, nil
	}
	for range 2 {
		if _, err := readByte(); err != nil {
			return nil, err
		}
	}
	plaintext := make([]byte, length)
	for i := range plaintext {
		value, err := readByte()
		if err != nil {
			return nil, err
		}
		plaintext[i] = value
	}
	for range 7 {
		value, err := readByte()
		if err != nil {
			return nil, err
		}
		if value != 0 {
			return nil, errors.New("TEA 尾部校验失败")
		}
	}
	return plaintext, nil
}

// Cipher transforms chunks using their absolute file offset.
type Cipher struct {
	key  []byte
	box  []byte
	hash uint32
}

func NewCipher(key []byte) (*Cipher, error) {
	if len(key) == 0 {
		return nil, errors.New("音频密钥不能为空")
	}
	cipher := &Cipher{key: append([]byte(nil), key...)}
	if len(key) <= 300 {
		return cipher, nil
	}
	cipher.box = make([]byte, len(key))
	for i := range cipher.box {
		cipher.box[i] = byte(i)
	}
	swap := 0
	for i := range cipher.box {
		swap = (swap + int(cipher.box[i]) + int(key[i])) % len(key)
		cipher.box[i], cipher.box[swap] = cipher.box[swap], cipher.box[i]
	}
	cipher.hash = 1
	for _, value := range key {
		if value == 0 {
			continue
		}
		next := cipher.hash * uint32(value)
		if next == 0 || next <= cipher.hash {
			break
		}
		cipher.hash = next
	}
	return cipher, nil
}

func (cipher *Cipher) Transform(chunk []byte, offset int64) {
	if len(cipher.key) <= 300 {
		cipher.mapTransform(chunk, offset)
		return
	}
	processed := 0
	if offset < 128 {
		count := min(len(chunk), 128-int(offset))
		for i := range count {
			chunk[i] ^= cipher.key[cipher.segmentSkip(offset+int64(i))]
		}
		offset += int64(count)
		processed = count
	}
	for processed < len(chunk) {
		count := min(len(chunk)-processed, 5120-int(offset%5120))
		cipher.segmentTransform(chunk[processed:processed+count], offset)
		offset += int64(count)
		processed += count
	}
}

func (cipher *Cipher) mapTransform(chunk []byte, offset int64) {
	for i := range chunk {
		position := offset + int64(i)
		if position > 0x7fff {
			position %= 0x7fff
		}
		index := (position*position + 71214) % int64(len(cipher.key))
		value := uint16(cipher.key[index])
		shift := uint((index&7)+4) & 7
		// QQ Music's mapL is same-direction shift OR, not a standard rotate.
		mask := byte(value)
		if shift != 0 {
			mask = byte(value<<shift | value>>shift)
		}
		chunk[i] ^= mask
	}
}

func (cipher *Cipher) segmentSkip(segment int64) int {
	seed := cipher.key[segment%int64(len(cipher.key))]
	if seed == 0 {
		return 0
	}
	return int(float64(cipher.hash)/float64((segment+1)*int64(seed))*100) % len(cipher.key)
}

func (cipher *Cipher) segmentTransform(chunk []byte, offset int64) {
	box := append([]byte(nil), cipher.box...)
	first, second := 0, 0
	skip := int(offset%5120) + cipher.segmentSkip(offset/5120)
	for i := -skip; i < len(chunk); i++ {
		first = (first + 1) % len(box)
		second = (second + int(box[first])) % len(box)
		box[first], box[second] = box[second], box[first]
		if i >= 0 {
			chunk[i] ^= box[(int(box[first])+int(box[second]))%len(box)]
		}
	}
}
