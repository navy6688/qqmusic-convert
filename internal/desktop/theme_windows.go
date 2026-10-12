//go:build windows

package desktop

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"
)

//go:embed assets/*
var uiAssets embed.FS

type activationContext struct {
	Size, Flags                                      uint32
	Source                                           *uint16
	Architecture, Language                           uint16
	AssemblyDirectory, ResourceName, ApplicationName *uint16
	Module                                           uintptr
}

func activateTheme() (func(), error) {
	manifest, err := uiAssets.ReadFile("assets/app.manifest")
	if err != nil {
		return nil, err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	directory := filepath.Join(cache, "qqmusic-convert", "ui")
	if err = os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	source := filepath.Join(directory, "app-v1.manifest")
	if err = os.WriteFile(source, manifest, 0600); err != nil {
		return nil, err
	}
	context := activationContext{Size: uint32(unsafe.Sizeof(activationContext{})), Source: wide(source)}
	handle, _, err := kernel32.NewProc("CreateActCtxW").Call(uintptr(unsafe.Pointer(&context)))
	if handle == ^uintptr(0) {
		return nil, fmt.Errorf("创建 Windows 控件主题失败: %v", err)
	}
	var cookie uintptr
	success, _, err := kernel32.NewProc("ActivateActCtx").Call(handle, uintptr(unsafe.Pointer(&cookie)))
	if success == 0 {
		kernel32.NewProc("ReleaseActCtx").Call(handle)
		return nil, fmt.Errorf("激活 Windows 控件主题失败: %v", err)
	}
	return func() {
		kernel32.NewProc("DeactivateActCtx").Call(0, cookie)
		kernel32.NewProc("ReleaseActCtx").Call(handle)
	}, nil
}

func (app *window) showHelp() {
	notice, err := uiAssets.ReadFile("assets/licenses.txt")
	if err != nil {
		alert(app.handle, "使用说明", err.Error())
		return
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		alert(app.handle, "使用说明", err.Error())
		return
	}
	directory := filepath.Join(cache, "qqmusic-convert", "ui")
	if err = os.MkdirAll(directory, 0700); err != nil {
		alert(app.handle, "使用说明", err.Error())
		return
	}
	path := filepath.Join(directory, "第三方来源与许可.txt")
	if err = os.WriteFile(path, notice, 0600); err != nil {
		alert(app.handle, "使用说明", err.Error())
		return
	}
	alert(app.handle, "使用说明", "1. 将 EXE 放进音乐目录，双击后自动扫描该目录。\n2. 也可以选择其他源文件夹和保存目录。\n3. 加密歌曲需要 QQ 音乐正在运行并保持登录。\n4. 点击开始转换；停止不会删除已完成的 MP3。\n5. 原文件和已有输出不覆盖。\n6. 同名多种格式会附加格式后缀，避免冲突。\n\n成功结果和失败原因保存在目标文件夹的转换报告中。\n\n第三方许可与来源：\n"+path)
}
