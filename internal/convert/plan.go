package convert

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var formats = map[string]bool{".mflac": true, ".mgg": true, ".mmp4": true, ".flac": true, ".wav": true, ".ogg": true, ".m4a": true, ".aac": true, ".opus": true, ".wma": true, ".mp3": true}

type Job struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Extension   string `json:"extension"`
}

func Plan(source, output string) ([]Job, error) {
	if strings.TrimSpace(source) == "" || strings.TrimSpace(output) == "" {
		return nil, fmt.Errorf("源目录或目标目录为空：源 %q，目标 %q", source, output)
	}
	source, err := filepath.Abs(strings.TrimSpace(source))
	if err != nil {
		return nil, err
	}
	output, err = filepath.Abs(strings.TrimSpace(output))
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(source)
	if err != nil {
		return nil, fmt.Errorf("源文件夹不可访问 %s: %w", source, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("源路径不是文件夹: %s", source)
	}
	if info, err = os.Stat(output); err == nil && !info.IsDir() {
		return nil, fmt.Errorf("目标路径不是文件夹: %s", output)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("目标文件夹不可访问 %s: %w", output, err)
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return nil, fmt.Errorf("读取源目录 %s: %w", source, err)
	}
	counts := map[string]int{}
	for _, entry := range entries {
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if !entry.IsDir() && formats[extension] {
			stem := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			counts[strings.ToLower(stem)]++
		}
	}
	var jobs []Job
	destinations := map[string]bool{}
	for _, entry := range entries {
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if entry.IsDir() || !formats[extension] {
			continue
		}
		stem := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		// Multiple formats of the same title remain separate, without overwriting.
		if counts[strings.ToLower(stem)] > 1 && extension != ".mp3" {
			stem += " (" + extension[1:] + ")"
		}
		destination := filepath.Join(output, stem+".mp3")
		identity := strings.ToLower(destination)
		if destinations[identity] {
			existing, err := os.Stat(destination)
			if err != nil || !existing.Mode().IsRegular() {
				return nil, fmt.Errorf("两个源文件会生成同一个目标 %s，请分开转换", destination)
			}
			// Re-running with output==source may list the already-generated MP3 too.
			// Both entries are safe: ConvertFile skips this existing regular file.
		}
		destinations[identity] = true
		jobs = append(jobs, Job{Source: filepath.Join(source, entry.Name()), Destination: destination, Extension: extension})
	}
	sort.Slice(jobs, func(i, j int) bool { return strings.ToLower(jobs[i].Source) < strings.ToLower(jobs[j].Source) })
	return jobs, nil
}
