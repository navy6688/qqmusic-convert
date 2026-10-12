package convert

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"qqmusic-convert/internal/audio"
)

type KeyProvider interface {
	Key(context.Context, audio.Footer) (string, error)
}

func sourceHash(ctx context.Context, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	buffer := make([]byte, 256*1024)
	for {
		if err = ctx.Err(); err != nil {
			return "", err
		}
		count, readErr := file.Read(buffer)
		if count > 0 {
			if _, err = hash.Write(buffer[:count]); err != nil {
				return "", err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func decryptedSource(ctx context.Context, job Job, temporary string, keys KeyProvider, progress func(string, float64)) (string, string, error) {
	encrypted := job.Extension == ".mflac" || job.Extension == ".mgg" || job.Extension == ".mmp4"
	if !encrypted {
		return job.Source, job.Extension[1:], nil
	}
	input, err := os.Open(job.Source)
	if err != nil {
		return "", "", err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return "", "", err
	}
	if info.Size() < 16 {
		return "", "", fmt.Errorf("加密文件长度不足：%d", info.Size())
	}
	trailer := make([]byte, 16)
	if _, err = input.ReadAt(trailer, info.Size()-16); err != nil {
		return "", "", err
	}
	if string(trailer[8:]) != "musicex\x00" {
		header := make([]byte, 16)
		count, err := input.ReadAt(header, 0)
		if err != nil && !errors.Is(err, io.EOF) {
			return "", "", err
		}
		container := audio.DetectContainer(header[:count])
		if container != "" {
			return job.Source, container, nil
		}
		return "", "", errors.New("文件不是支持的 musicex V1；旧 QMC 加密格式不受支持")
	}
	footer, err := audio.ReadFooter(input, info.Size())
	if err != nil {
		return "", "", err
	}
	if footer.MetadataLayers > 1 {
		progress(fmt.Sprintf("解析 %d 层尾部，使用原始音频资源", footer.MetadataLayers), 0.03)
	}
	progress("获取这首歌的密钥", 0.04)
	encoded, err := keys.Key(ctx, footer)
	if err != nil {
		return "", "", err
	}
	secret, err := audio.DeriveKey(encoded)
	if err != nil {
		return "", "", err
	}
	cipher, err := audio.NewCipher(secret)
	if err != nil {
		return "", "", err
	}
	reader := io.NewSectionReader(input, 0, footer.AudioLength)
	buffer := make([]byte, 256*1024)
	count, err := io.ReadFull(reader, buffer)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", "", err
	}
	cipher.Transform(buffer[:count], 0)
	container := audio.DetectContainer(buffer[:count])
	if container == "" {
		return "", "", errors.New("解密内容不是可识别音频；密钥不匹配或文件已损坏")
	}
	path := filepath.Join(temporary, "audio."+container)
	output, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", "", err
	}
	defer output.Close()
	if _, err = output.Write(buffer[:count]); err != nil {
		return "", "", err
	}
	offset := int64(count)
	for offset < footer.AudioLength {
		if err = ctx.Err(); err != nil {
			return "", "", err
		}
		count, err = reader.Read(buffer)
		if err != nil {
			return "", "", fmt.Errorf("读取加密音频偏移 %d 失败: %w", offset, err)
		}
		cipher.Transform(buffer[:count], offset)
		if _, err = output.Write(buffer[:count]); err != nil {
			return "", "", err
		}
		offset += int64(count)
		progress("解密音频", 0.06+0.32*float64(offset)/float64(footer.AudioLength))
	}
	if err = output.Close(); err != nil {
		return "", "", err
	}
	return path, container, nil
}

func copyExclusive(ctx context.Context, candidate, destination string) error {
	input, err := os.Open(candidate)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		output.Close()
		if !committed {
			os.Remove(destination)
		}
	}()
	buffer := make([]byte, 256*1024)
	for {
		if err = ctx.Err(); err != nil {
			return err
		}
		count, readErr := input.Read(buffer)
		if count > 0 {
			if _, err = output.Write(buffer[:count]); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	if err = output.Close(); err != nil {
		return err
	}
	committed = true
	return nil
}
