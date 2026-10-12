package convert

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"qqmusic-convert/internal/media"
)

type Probe struct {
	Codec      string  `json:"codec"`
	SampleRate int     `json:"sample_rate"`
	Channels   int     `json:"channels"`
	Duration   float64 `json:"duration"`
}

type Integrity struct {
	Method   string `json:"method"`
	Samples  uint64 `json:"samples,omitempty"`
	PCMMD5   string `json:"pcm_md5,omitempty"`
	Warnings bool   `json:"decoder_warning,omitempty"`
}

type limitedLog struct{ bytes.Buffer }

func (log *limitedLog) Write(chunk []byte) (int, error) {
	count := len(chunk)
	if log.Len() < 4096 {
		if _, err := log.Buffer.Write(chunk[:min(count, 4096-log.Len())]); err != nil {
			return 0, err
		}
	}
	return count, nil
}

func inspectAudio(ctx context.Context, path string, tools media.Tools) (Probe, error) {
	command := media.Command(ctx, tools.FFprobe, "-v", "error", "-select_streams", "a:0", "-show_entries",
		"format=duration:stream=codec_name,sample_rate,channels", "-of", "json", path)
	var stdout bytes.Buffer
	var stderr limitedLog
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return Probe{}, ctx.Err()
		}
		return Probe{}, fmt.Errorf("音频识别失败: %s", strings.TrimSpace(stderr.String()))
	}
	var reply struct {
		Streams []struct {
			Codec    string `json:"codec_name"`
			Rate     string `json:"sample_rate"`
			Channels int    `json:"channels"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &reply); err != nil || len(reply.Streams) != 1 {
		return Probe{}, fmt.Errorf("无法识别有效音轨 %s", path)
	}
	duration, err := strconv.ParseFloat(reply.Format.Duration, 64)
	if err != nil || duration <= 0 || math.IsInf(duration, 0) || math.IsNaN(duration) {
		return Probe{}, fmt.Errorf("无效音频时长 %q", reply.Format.Duration)
	}
	rate, err := strconv.Atoi(reply.Streams[0].Rate)
	if err != nil || rate <= 0 || reply.Streams[0].Channels <= 0 {
		return Probe{}, fmt.Errorf("无效音轨采样率 %q 或声道数 %d", reply.Streams[0].Rate, reply.Streams[0].Channels)
	}
	return Probe{Codec: reply.Streams[0].Codec, SampleRate: rate, Channels: reply.Streams[0].Channels, Duration: duration}, nil
}

func timedAudioCommand(ctx context.Context, tools media.Tools, arguments []string, duration float64, progress func(float64)) error {
	arguments = append([]string{"-hide_banner", "-v", "error", "-nostats", "-progress", "pipe:1"}, arguments...)
	command := media.Command(ctx, tools.FFmpeg, arguments...)
	var stderr limitedLog
	command.Stderr = &stderr
	pipe, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	if err = command.Start(); err != nil {
		return fmt.Errorf("无法启动内置 FFmpeg: %w", err)
	}
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "out_time_us=") && duration > 0 {
			microseconds, parseErr := strconv.ParseFloat(strings.TrimPrefix(line, "out_time_us="), 64)
			if parseErr == nil && progress != nil {
				progress(min(1, max(0, microseconds/1e6/duration)))
			}
		}
	}
	scanErr := scanner.Err()
	waitErr := command.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if scanErr != nil {
		return fmt.Errorf("读取转码进度失败: %w", scanErr)
	}
	if waitErr != nil {
		return fmt.Errorf("FFmpeg 处理失败: %s", strings.TrimSpace(stderr.String()))
	}
	return nil
}

type flacInfo struct {
	Rate     int
	Channels int
	Bits     int
	Samples  uint64
	MD5      string
}

func readFLACInfo(path string) (flacInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return flacInfo{}, err
	}
	defer file.Close()
	header := make([]byte, 42)
	if _, err = io.ReadFull(file, header); err != nil {
		return flacInfo{}, err
	}
	if string(header[:4]) != "fLaC" || header[4]&127 != 0 || header[5] != 0 || header[6] != 0 || header[7] != 34 {
		return flacInfo{}, fmt.Errorf("FLAC STREAMINFO 数据不合法")
	}
	packed := binary.BigEndian.Uint64(header[18:26])
	return flacInfo{Rate: int(packed >> 44), Channels: int((packed>>41)&7) + 1, Bits: int((packed>>36)&31) + 1,
		Samples: packed & ((1 << 36) - 1), MD5: hex.EncodeToString(header[26:42])}, nil
}

type pcmDigest struct {
	hash     hash.Hash
	count    uint64
	expected uint64
	progress func(float64)
}

func (digest *pcmDigest) Write(chunk []byte) (int, error) {
	count, err := digest.hash.Write(chunk)
	digest.count += uint64(count)
	if digest.expected > 0 && digest.progress != nil {
		digest.progress(min(1, float64(digest.count)/float64(digest.expected)))
	}
	return count, err
}

func verifySource(ctx context.Context, path, container string, probe Probe, tools media.Tools, progress func(float64)) (Integrity, error) {
	if container != "flac" {
		err := timedAudioCommand(ctx, tools, []string{"-xerror", "-i", path, "-map", "0:a:0", "-f", "null", "-"}, probe.Duration, progress)
		return Integrity{Method: "strict_full_decode"}, err
	}
	info, err := readFLACInfo(path)
	if err != nil {
		return Integrity{}, err
	}
	if info.Samples == 0 || info.MD5 == strings.Repeat("0", 32) {
		err := timedAudioCommand(ctx, tools, []string{"-xerror", "-i", path, "-map", "0:a:0", "-f", "null", "-"}, probe.Duration, progress)
		return Integrity{Method: "strict_full_decode"}, err
	}
	formats := map[int][2]string{8: {"pcm_s8", "s8"}, 16: {"pcm_s16le", "s16le"}, 24: {"pcm_s24le", "s24le"}, 32: {"pcm_s32le", "s32le"}}
	format, ok := formats[info.Bits]
	if !ok {
		return Integrity{}, fmt.Errorf("FLAC 完整性校验不支持 %d 位音频", info.Bits)
	}
	expected := info.Samples * uint64(info.Channels) * uint64((info.Bits+7)/8)
	digest := &pcmDigest{hash: md5.New(), expected: expected, progress: progress}
	command := media.Command(ctx, tools.FFmpeg, "-hide_banner", "-v", "error", "-i", path, "-map", "0:a:0", "-c:a", format[0], "-f", format[1], "pipe:1")
	var stderr limitedLog
	command.Stdout = digest
	command.Stderr = &stderr
	if err = command.Run(); err != nil {
		if ctx.Err() != nil {
			return Integrity{}, ctx.Err()
		}
		return Integrity{}, fmt.Errorf("FLAC 完整解码失败: %s", strings.TrimSpace(stderr.String()))
	}
	actual := hex.EncodeToString(digest.hash.Sum(nil))
	if digest.count != expected || actual != info.MD5 {
		return Integrity{}, fmt.Errorf("FLAC 完整性校验失败：期望 %d PCM 字节，实际 %d；期望 MD5 %s，实际 %s", expected, digest.count, info.MD5, actual)
	}
	return Integrity{Method: "pcm_md5_and_sample_count", Samples: info.Samples, PCMMD5: actual, Warnings: stderr.Len() > 0}, nil
}
