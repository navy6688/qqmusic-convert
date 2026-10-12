//go:build windows

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"qqmusic-convert/internal/convert"
	"qqmusic-convert/internal/desktop"
)

func main() {
	source := flag.String("convert", "", "无界面转换指定源文件夹（自动验证用）")
	output := flag.String("output", "", "无界面模式的目标文件夹")
	snapshot := flag.String("ui-smoke", "", "渲染本程序界面 PNG 并退出（界面自检）")
	runCheck := flag.Bool("ui-run-check", false, "在界面自检中运行转换后截图")
	maximizeCheck := flag.Bool("ui-maximize", false, "界面自检执行原生最大化")
	flag.Parse()
	if *source != "" {
		if *output == "" {
			*output = filepath.Join(*source, "MP3")
		}
		jobs, err := convert.Plan(*source, *output)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		summary := convert.Run(context.Background(), jobs, func(convert.Event) {})
		if err = json.NewEncoder(os.Stdout).Encode(summary); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if summary.Failed > 0 || summary.Error != "" {
			os.Exit(1)
		}
		return
	}
	if err := desktop.Run(*snapshot, *runCheck, *maximizeCheck); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if *snapshot == "" {
			desktop.ShowError(err.Error())
		}
		os.Exit(1)
	}
}
