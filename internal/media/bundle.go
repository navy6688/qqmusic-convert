package media

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

//go:embed assets/*
var bundled embed.FS

type asset struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type Tools struct {
	FFmpeg  string
	FFprobe string
}

// Prepare verifies and expands the embedded binaries into a per-user cache.
func Prepare(ctx context.Context, progress func(float64)) (Tools, error) {
	manifest, err := bundled.ReadFile("assets/manifest.json")
	if err != nil {
		return Tools{}, fmt.Errorf("EXE 缺少内置转码组件，请通过 build.ps1 重新构建: %w", err)
	}
	var assets []asset
	if err = json.Unmarshal(manifest, &assets); err != nil || len(assets) != 2 {
		return Tools{}, fmt.Errorf("内置转码组件清单无效")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return Tools{}, fmt.Errorf("无法取得用户缓存目录: %w", err)
	}
	fingerprint := sha256.Sum256(manifest)
	directory := filepath.Join(cache, "qqmusic-convert", "media", hex.EncodeToString(fingerprint[:6]))
	if err = os.MkdirAll(directory, 0700); err != nil {
		return Tools{}, fmt.Errorf("无法准备组件目录 %s: %w", directory, err)
	}
	tools := Tools{FFmpeg: filepath.Join(directory, "ffmpeg.exe"), FFprobe: filepath.Join(directory, "ffprobe.exe")}
	for index, entry := range assets {
		if entry.Name != "ffmpeg.exe" && entry.Name != "ffprobe.exe" {
			return Tools{}, fmt.Errorf("无效组件名称 %q", entry.Name)
		}
		path := filepath.Join(directory, entry.Name)
		valid, err := validAsset(ctx, path, entry)
		if err != nil {
			return Tools{}, err
		}
		if !valid {
			if err = expandAsset(ctx, path, entry); err != nil {
				return Tools{}, err
			}
		}
		if progress != nil {
			progress(float64(index+1) / float64(len(assets)))
		}
	}
	return tools, nil
}

func validAsset(ctx context.Context, path string, entry asset) (bool, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("无法读取缓存组件 %s: %w", path, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return false, err
	}
	if info.Size() != entry.Size {
		return false, nil
	}
	digest := sha256.New()
	buffer := make([]byte, 1024*1024)
	for {
		if err = ctx.Err(); err != nil {
			return false, err
		}
		count, readErr := file.Read(buffer)
		if count > 0 {
			if _, err = digest.Write(buffer[:count]); err != nil {
				return false, err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return false, readErr
		}
	}
	return hex.EncodeToString(digest.Sum(nil)) == entry.SHA256, nil
}

func expandAsset(ctx context.Context, path string, entry asset) error {
	packed, err := bundled.Open("assets/" + entry.Name + ".gz")
	if err != nil {
		return fmt.Errorf("内置组件 %s 不存在", entry.Name)
	}
	defer packed.Close()
	reader, err := gzip.NewReader(packed)
	if err != nil {
		return fmt.Errorf("内置组件 %s 无法解压: %w", entry.Name, err)
	}
	defer reader.Close()
	temporary, err := os.CreateTemp(filepath.Dir(path), ".component-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name) // Only the exact temporary file created above is removed.
	digest := sha256.New()
	buffer := make([]byte, 1024*1024)
	var total int64
	for {
		if err = ctx.Err(); err != nil {
			temporary.Close()
			return err
		}
		count, readErr := reader.Read(buffer)
		if count > 0 {
			written, writeErr := temporary.Write(buffer[:count])
			total += int64(written)
			if writeErr != nil {
				temporary.Close()
				return writeErr
			}
			if _, err = digest.Write(buffer[:count]); err != nil {
				temporary.Close()
				return err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			temporary.Close()
			return readErr
		}
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	if total != entry.Size || hex.EncodeToString(digest.Sum(nil)) != entry.SHA256 {
		return fmt.Errorf("内置组件 %s 校验失败", entry.Name)
	}
	// This replaces only our verified cache file, never an audio input or output.
	if err = os.Rename(name, path); err != nil {
		return fmt.Errorf("安装缓存组件 %s 失败: %w", entry.Name, err)
	}
	return nil
}
