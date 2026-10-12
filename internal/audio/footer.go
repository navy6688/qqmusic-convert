package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"regexp"
	"unicode/utf16"
)

var songMID = regexp.MustCompile(`^[A-Za-z0-9]{1,30}$`)
var resourceName = regexp.MustCompile(`^[A-Za-z0-9]+\.m(flac|gg|mp4)$`)

// Footer identifies the specific encrypted musicex resource, not just its title.
type Footer struct {
	MID            string
	Resource       string
	AudioLength    int64
	MetadataLayers int
}

func ParseFooter(trailer []byte, fileSize int64) (Footer, error) {
	if len(trailer) < 192 || string(trailer[len(trailer)-8:]) != "musicex\x00" {
		return Footer{}, fmt.Errorf("不支持的加密格式：文件没有 musicex V1 标记")
	}
	size := int64(binary.LittleEndian.Uint32(trailer[len(trailer)-16:]))
	version := binary.LittleEndian.Uint32(trailer[len(trailer)-12:])
	if size < 192 || size >= fileSize || size > int64(len(trailer)) {
		return Footer{}, fmt.Errorf("无效尾部长度 %d，文件长度 %d", size, fileSize)
	}
	if version != 1 {
		return Footer{}, fmt.Errorf("不支持的 musicex 版本 %d", version)
	}
	footer := trailer[int64(len(trailer))-size:]
	mid, resource := fieldUTF16(footer[12:72]), fieldUTF16(footer[72:140])
	if !songMID.MatchString(mid) {
		return Footer{}, fmt.Errorf("无效歌曲标识 %q", mid)
	}
	if !resourceName.MatchString(resource) {
		return Footer{}, fmt.Errorf("无效资源文件名 %q", resource)
	}
	return Footer{MID: mid, Resource: resource, AudioLength: fileSize - size, MetadataLayers: 1}, nil
}

func fieldUTF16(encoded []byte) string {
	units := make([]uint16, 0, len(encoded)/2)
	for offset := 0; offset+1 < len(encoded); offset += 2 {
		unit := binary.LittleEndian.Uint16(encoded[offset:])
		if unit == 0 {
			break
		}
		units = append(units, unit)
	}
	return string(utf16.Decode(units))
}

func DetectContainer(header []byte) string {
	switch {
	case len(header) >= 4 && string(header[:4]) == "fLaC":
		return "flac"
	case len(header) >= 4 && string(header[:4]) == "OggS":
		return "ogg"
	case len(header) >= 8 && string(header[4:8]) == "ftyp":
		return "m4a"
	case len(header) >= 3 && string(header[:3]) == "ID3":
		return "mp3"
	case len(header) >= 2 && header[0] == 255 && (header[1] == 0xf2 || header[1] == 0xf3 || header[1] == 0xfb):
		return "mp3"
	default:
		return ""
	}
}

// ReadFooter resolves contiguous musicex layers from the outside inward.
// The innermost layer belongs to the encrypted bytes. QQ Music can append a
// different alias resource after reusing cached audio; that outer key is wrong.
func ReadFooter(reader io.ReaderAt, fileSize int64) (Footer, error) {
	cursor := fileSize
	var original Footer
	layers := 0
	for cursor >= 16 {
		trailer := make([]byte, 16)
		if _, err := reader.ReadAt(trailer, cursor-16); err != nil {
			return Footer{}, fmt.Errorf("读取音频尾部失败: %w", err)
		}
		if string(trailer[8:]) != "musicex\x00" {
			break
		}
		size := int64(binary.LittleEndian.Uint32(trailer))
		if size < 192 || size >= cursor || size > 1024*1024 {
			return Footer{}, fmt.Errorf("无效加密尾部长度 %d，剩余文件长度 %d", size, cursor)
		}
		layers++
		if layers > 64 {
			return Footer{}, fmt.Errorf("musicex 尾部层数超过 64")
		}
		footerBytes := make([]byte, size)
		if _, err := reader.ReadAt(footerBytes, cursor-size); err != nil {
			return Footer{}, fmt.Errorf("读取第 %d 层尾部失败: %w", layers, err)
		}
		footer, err := ParseFooter(footerBytes, cursor)
		if err != nil {
			return Footer{}, fmt.Errorf("第 %d 层尾部无效: %w", layers, err)
		}
		original = footer
		cursor -= size
	}
	if layers == 0 {
		return Footer{}, fmt.Errorf("文件没有 musicex V1 尾部")
	}
	original.AudioLength = cursor
	original.MetadataLayers = layers
	return original, nil
}
