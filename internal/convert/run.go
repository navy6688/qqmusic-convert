package convert

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"qqmusic-convert/internal/media"
	"qqmusic-convert/internal/qqmusic"
)

type Result struct {
	Job          Job       `json:"job"`
	Status       string    `json:"status"`
	Message      string    `json:"message"`
	SourceSHA256 string    `json:"source_sha256,omitempty"`
	Integrity    Integrity `json:"source_integrity,omitempty"`
	Probe        Probe     `json:"mp3,omitempty"`
}

type Summary struct {
	Total     int      `json:"total"`
	Converted int      `json:"converted"`
	Skipped   int      `json:"skipped"`
	Failed    int      `json:"failed"`
	Canceled  bool     `json:"canceled"`
	Error     string   `json:"error,omitempty"`
	Report    string   `json:"report,omitempty"`
	Results   []Result `json:"results"`
}

type Event struct {
	Index    int
	Total    int
	Stage    string
	Fraction float64
	Result   *Result
}

func Run(ctx context.Context, jobs []Job, notify func(Event)) Summary {
	summary := Summary{Total: len(jobs)}
	if len(jobs) == 0 {
		return summary
	}
	needsTools := false
	for _, job := range jobs {
		if _, err := os.Stat(job.Destination); os.IsNotExist(err) {
			needsTools = true
			break
		}
	}
	var tools media.Tools
	if needsTools {
		var err error
		tools, err = media.Prepare(ctx, func(fraction float64) {
			notify(Event{Index: -1, Total: len(jobs), Stage: fmt.Sprintf("准备内置转码组件 %.0f%%", fraction*100)})
		})
		if err != nil {
			summary.Error = err.Error()
			summary.Canceled = ctx.Err() != nil
			return summary
		}
	}
	keys := qqmusic.NewClient()
	defer keys.Clear()
	for index, job := range jobs {
		if ctx.Err() != nil {
			summary.Canceled = true
			break
		}
		lastUpdate := time.Time{}
		lastStage := ""
		progress := func(stage string, fraction float64) {
			now := time.Now()
			if stage != lastStage || now.Sub(lastUpdate) >= 80*time.Millisecond || fraction >= 1 {
				notify(Event{Index: index, Total: len(jobs), Stage: stage, Fraction: fraction})
				lastUpdate = now
				lastStage = stage
			}
		}
		result := ConvertFile(ctx, job, tools, keys, progress)
		if errors.Is(ctx.Err(), context.Canceled) && result.Status != "converted" && result.Status != "skipped" {
			summary.Canceled = true
			break
		}
		summary.Results = append(summary.Results, result)
		switch result.Status {
		case "converted":
			summary.Converted++
		case "skipped":
			summary.Skipped++
		default:
			summary.Failed++
		}
		notify(Event{Index: index, Total: len(jobs), Stage: result.Message, Fraction: 1, Result: &result})
	}
	if len(summary.Results) > 0 {
		report, err := writeReport(jobs[0].Destination, summary)
		if err != nil {
			summary.Error = "转换结果已保留，但保存报告失败: " + err.Error()
		} else {
			summary.Report = report
		}
	}
	return summary
}

func ConvertFile(ctx context.Context, job Job, tools media.Tools, keys KeyProvider, progress func(string, float64)) Result {
	result := Result{Job: job, Status: "failed"}
	if existing, err := os.Stat(job.Destination); err == nil {
		if !existing.Mode().IsRegular() {
			result.Message = "目标已存在且不是普通文件：" + job.Destination
			return result
		}
		result.Status = "skipped"
		result.Message = "目标已存在，未覆盖"
		return result
	} else if !os.IsNotExist(err) {
		result.Message = err.Error()
		return result
	}
	if progress == nil {
		progress = func(string, float64) {}
	}
	progress("检查原文件", 0.01)
	fingerprint, err := sourceHash(ctx, job.Source)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	result.SourceSHA256 = fingerprint
	parent := filepath.Dir(job.Destination)
	if err = os.MkdirAll(parent, 0755); err != nil {
		result.Message = fmt.Sprintf("创建目标文件夹失败: %v", err)
		return result
	}
	temporary, err := os.MkdirTemp(parent, ".qqmusic-convert-")
	if err != nil {
		result.Message = err.Error()
		return result
	}
	if filepath.Dir(temporary) != filepath.Clean(parent) {
		result.Message = "临时目录越出目标文件夹"
		return result
	}
	defer os.RemoveAll(temporary) // Verified exact child of the selected output directory.
	source, container, err := decryptedSource(ctx, job, temporary, keys, progress)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	probe, err := inspectAudio(ctx, source, tools)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	integrity, err := verifySource(ctx, source, container, probe, tools, func(fraction float64) { progress("校验完整音频", 0.40+0.18*fraction) })
	if err != nil {
		result.Message = err.Error()
		return result
	}
	candidate := filepath.Join(temporary, "verified.mp3")
	if container == "mp3" && probe.Codec == "mp3" {
		err = copyExclusive(ctx, source, candidate)
	} else {
		err = timedAudioCommand(ctx, tools, []string{"-n", "-i", source, "-map", "0:a:0", "-map_metadata", "0", "-c:a", "libmp3lame", "-b:a", "320k", candidate}, probe.Duration,
			func(fraction float64) { progress("转换为 MP3", 0.58+0.34*fraction) })
	}
	if err != nil {
		result.Message = err.Error()
		return result
	}
	outputProbe, err := inspectAudio(ctx, candidate, tools)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	if outputProbe.Codec != "mp3" || (integrity.Method == "pcm_md5_and_sample_count" && math.Abs(outputProbe.Duration-probe.Duration) > 0.12) {
		result.Message = fmt.Sprintf("输出音频校验失败：格式 %s，源时长 %.3f 秒，输出 %.3f 秒", outputProbe.Codec, probe.Duration, outputProbe.Duration)
		return result
	}
	err = timedAudioCommand(ctx, tools, []string{"-xerror", "-i", candidate, "-map", "0:a:0", "-f", "null", "-"}, outputProbe.Duration,
		func(fraction float64) { progress("检查 MP3 全曲", 0.92+0.06*fraction) })
	if err != nil {
		result.Message = err.Error()
		return result
	}
	after, err := sourceHash(ctx, job.Source)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	if after != fingerprint {
		result.Message = "原文件在转换过程中发生变化，未发布 MP3"
		return result
	}
	if err = publish(ctx, candidate, job.Destination); err != nil {
		if errors.Is(err, os.ErrExist) {
			result.Status = "skipped"
			result.Message = "目标被其他程序创建，未覆盖"
			return result
		}
		result.Message = err.Error()
		return result
	}
	result.Status = "converted"
	result.Message = "转换完成，校验通过"
	result.Integrity = integrity
	result.Probe = outputProbe
	progress("转换完成", 1)
	return result
}

func writeReport(destination string, summary Summary) (string, error) {
	path := filepath.Join(filepath.Dir(destination), "转换报告_"+time.Now().Format("20060102_150405.000000000")+".json")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return "", err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(summary); err != nil {
		file.Close()
		return "", err
	}
	if err = file.Close(); err != nil {
		return "", err
	}
	return path, nil
}
