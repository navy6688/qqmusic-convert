//go:build windows

package desktop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"qqmusic-convert/internal/convert"
)

const (
	idSourceBrowse = 101
	idTargetBrowse = 102
	idStart        = 103
	idStop         = 104
	idRefresh      = 105
	idOpen         = 106
	idDetails      = 107
	idHelp         = 108
)

var current *window
var procedure = syscall.NewCallback(windowProcedure)
var folderCallback = syscall.NewCallback(folderProcedure)

const mainWindowStyle = 0x00cf0000 // WS_OVERLAPPEDWINDOW: sizing, minimize and maximize.

type window struct {
	logo, icon, smallIcon, brandIcon                                                                      uintptr
	handle, instance                                                                                      uintptr
	source, target, sourceBrowse, targetBrowse, list, start, stop, refresh, open, progress, details, help uintptr
	title, subtitle, sourceLabel, targetLabel, hint, filesLabel, stage, summary                           uintptr
	font, titleFont, background, white                                                                    uintptr
	scale                                                                                                 float64
	jobs                                                                                                  []convert.Job
	rowMessages                                                                                           []string
	updates                                                                                               chan convert.Event
	done                                                                                                  chan convert.Summary
	cancel                                                                                                context.CancelFunc
	active, closing                                                                                       bool
	success, skipped, failed                                                                              int
	smokePath                                                                                             string
	smokeError                                                                                            error
}

// Run owns the Windows UI thread; workers communicate only through channels.
func Run(smokePath string, runCheck, maximizedCheck bool) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	result, _, _ := initializeCOM.Call(0, 2)
	if int32(result) < 0 {
		return fmt.Errorf("初始化文件夹选择界面失败：HRESULT %x", result)
	}
	defer uninitializeCOM.Call()
	if setDPIAware.Find() == nil {
		setDPIAware.Call(^uintptr(1))
	}
	releaseTheme, err := activateTheme()
	if err != nil {
		return err
	}
	defer releaseTheme()
	controls := commonControls{Size: uint32(unsafe.Sizeof(commonControls{})), Classes: 0x00004021}
	initializeControls.Call(uintptr(unsafe.Pointer(&controls)))
	instance, _, _ := moduleHandle.Call(0)
	background, _, _ := createBrush.Call(0x00fbf7f5)
	white, _, _ := createBrush.Call(0x00ffffff)
	app := &window{instance: instance, background: background, white: white, scale: 1, updates: make(chan convert.Event, 64), done: make(chan convert.Summary, 1)}
	current = app
	defer func() {
		current = nil
		deleteObject.Call(app.font)
		deleteObject.Call(app.titleFont)
		deleteObject.Call(background)
		deleteObject.Call(white)
	}()
	name := wide("QQMusicConvertWindow")
	cursor, _, _ := loadCursor.Call(0, 32512)
	icon, err := brandIcon(instance, 32)
	if err != nil {
		return err
	}
	app.icon = icon
	smallIcon, err := brandIcon(instance, 16)
	if err != nil {
		return err
	}
	app.smallIcon = smallIcon
	defer func() {
		for _, owned := range []uintptr{app.icon, app.smallIcon, app.brandIcon} {
			if owned != 0 {
				user32.NewProc("DestroyIcon").Call(owned)
			}
		}
	}()
	class := windowClass{Size: uint32(unsafe.Sizeof(windowClass{})), Procedure: procedure, Instance: instance, Icon: icon, SmallIcon: smallIcon, Cursor: cursor, Background: background, ClassName: name}
	registered, _, err := registerClass.Call(uintptr(unsafe.Pointer(&class)))
	if registered == 0 {
		return fmt.Errorf("注册 Windows 窗口失败: %v", err)
	}
	defaultCoordinate := int32(-2147483648)
	defaultPosition := uintptr(defaultCoordinate)
	handle, _, err := createWindow.Call(0, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(wide("QQ 音乐转换工具"))), mainWindowStyle,
		defaultPosition, defaultPosition, 1080, 760, 0, 0, instance, 0)
	if handle == 0 {
		return fmt.Errorf("创建 Windows 窗口失败: %v", err)
	}
	app.handle = handle
	if getDPI.Find() == nil {
		dpi, _, _ := getDPI.Call(handle)
		if dpi > 0 {
			app.scale = float64(dpi) / 96
		}
	}
	screenWidth, _, _ := getMetrics.Call(0)
	screenHeight, _, _ := getMetrics.Call(1)
	desiredWidth := min(int(1080*app.scale), int(screenWidth)-60)
	desiredHeight := min(int(760*app.scale), int(screenHeight)-60)
	moveWindow.Call(handle, uintptr((int(screenWidth)-desiredWidth)/2), uintptr((int(screenHeight)-desiredHeight)/2), uintptr(desiredWidth), uintptr(desiredHeight), 1)
	if err = app.createControls(); err != nil {
		destroyWindow.Call(handle)
		return err
	}
	app.layout()
	if maximizedCheck {
		if err = app.maximizeCheck(); err != nil {
			destroyWindow.Call(handle)
			return err
		}
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	source := filepath.Dir(executable)
	setText(app.source, source)
	setText(app.target, filepath.Join(source, "MP3"))
	app.scan(false)
	setTimer.Call(handle, 1, 80, 0)
	if smokePath != "" && !runCheck {
		// Self-check paints only this application's window; never touches other apps.
		showWindow.Call(handle, 0)
		updateWindow.Call(handle)
		if err = app.snapshot(smokePath); err != nil {
			destroyWindow.Call(handle)
			return err
		}
		destroyWindow.Call(handle)
		return nil
	}
	if runCheck {
		if smokePath == "" || len(app.jobs) == 0 {
			destroyWindow.Call(handle)
			return fmt.Errorf("界面运行自检需要截图路径和 EXE 目录内的音频夹具")
		}
		app.smokePath = smokePath
		app.startBatch()
		showWindow.Call(handle, 0)
	} else {
		showWindow.Call(handle, 10)
	}
	updateWindow.Call(handle)
	var event message
	for {
		received, _, err := getMessage.Call(uintptr(unsafe.Pointer(&event)), 0, 0, 0)
		if int32(received) == -1 {
			return fmt.Errorf("Windows 消息循环失败: %v", err)
		}
		if received == 0 {
			break
		}
		dialogHandled, _, _ := user32.NewProc("IsDialogMessageW").Call(handle, uintptr(unsafe.Pointer(&event)))
		if dialogHandled != 0 {
			continue
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&event)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&event)))
	}
	return app.smokeError
}

func (app *window) createControls() error {
	font, _, _ := createFont.Call(uintptr(int32(-int32(15*app.scale))), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(wide("Segoe UI"))))
	titleFont, _, _ := createFont.Call(uintptr(int32(-int32(25*app.scale))), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(wide("Segoe UI"))))
	app.font = font
	app.titleFont = titleFont
	definitions := []struct {
		destination  *uintptr
		class, label string
		style        uintptr
		id           int
	}{
		{&app.title, "STATIC", "QQ 音乐转换", 0, 0}, {&app.subtitle, "STATIC", "本地解密 · MP3 转码 · 完整音频校验", 0, 0},
		{&app.sourceLabel, "STATIC", "源文件夹", 0, 0}, {&app.targetLabel, "STATIC", "保存到", 0, 0},
		{&app.source, "EDIT", "", wsTabStop | 0x00800000 | 0x0080, 0}, {&app.target, "EDIT", "", wsTabStop | 0x00800000 | 0x0080, 0},
		{&app.sourceBrowse, "BUTTON", "选择文件夹…", wsTabStop, idSourceBrowse}, {&app.targetBrowse, "BUTTON", "选择文件夹…", wsTabStop, idTargetBrowse},
		{&app.hint, "STATIC", "加密音频只读使用 QQ 音乐当前登录状态取密钥，不保存凭据。仅扫描当前文件夹，原文件和已有 MP3 不覆盖。", 0, 0},
		{&app.filesLabel, "STATIC", "待转换文件", 0, 0},
		{&app.list, "SysListView32", "", wsTabStop | 0x00800000 | 0x0001 | 0x0004 | 0x0008, 0},
		{&app.start, "BUTTON", "开始转换", wsTabStop | 0x0001, idStart}, {&app.stop, "BUTTON", "停止", wsTabStop, idStop},
		{&app.refresh, "BUTTON", "刷新文件列表", wsTabStop, idRefresh}, {&app.open, "BUTTON", "打开保存目录", wsTabStop, idOpen},
		{&app.details, "BUTTON", "查看详情", wsTabStop, idDetails},
		{&app.help, "BUTTON", "使用说明", wsTabStop, idHelp},
		{&app.progress, "msctls_progress32", "", 1, 0}, {&app.stage, "STATIC", "请选择需要转换的音乐目录", 0, 0},
		{&app.summary, "STATIC", "支持 MFLAC / MGG / MMP4 / FLAC / WAV / M4A / OGG / AAC / OPUS / WMA / MP3", 0, 0},
	}
	for _, definition := range definitions {
		handle, err := control(app.handle, app.instance, definition.class, definition.label, definition.style, definition.id, app.font)
		if err != nil {
			return err
		}
		*definition.destination = handle
	}
	if err := app.createLogo(); err != nil {
		return err
	}
	sendMessage.Call(app.title, wmSetFont, app.titleFont, 1)
	sendMessage.Call(app.list, lvmFirst+54, 0, 0x00010020|0x00004000)
	for index, title := range []string{"音乐文件", "格式", "处理状态", "目标文件"} {
		column := listColumn{Mask: 1 | 2 | 4 | 8, Width: 200, Text: wide(title), SubItem: int32(index)}
		sendMessage.Call(app.list, lvmFirst+97, uintptr(index), uintptr(unsafe.Pointer(&column)))
	}
	windows.NewLazySystemDLL("uxtheme.dll").NewProc("SetWindowTheme").Call(app.progress, uintptr(unsafe.Pointer(wide(""))), uintptr(unsafe.Pointer(wide(""))))
	sendMessage.Call(app.progress, 0x0406, 0, 1000)
	sendMessage.Call(app.progress, 0x0409, 0, 0x00cf682e)
	enabled(app.stop, false)
	return nil
}

func (app *window) layout() {
	if app.source == 0 {
		return
	}
	var client rectangle
	getClientRect.Call(app.handle, uintptr(unsafe.Pointer(&client)))
	width, height := int(client.Right), int(client.Bottom)
	if width <= 0 || height <= 0 {
		return
	}
	if app.minimumSize(width, height) {
		return
	}
	px := func(value int) int { return int(float64(value) * app.scale) }
	margin := px(28)
	fieldX := px(112)
	buttonWidth := px(130)
	fieldWidth := width - fieldX - buttonWidth - margin - px(12)
	place(app.logo, margin, px(22), px(52), px(52))
	place(app.title, margin+px(70), px(19), width-2*margin-px(70), px(36))
	place(app.subtitle, margin+px(70), px(58), width-2*margin-px(70), px(24))
	place(app.sourceLabel, margin, px(111), px(80), px(26))
	place(app.targetLabel, margin, px(161), px(80), px(26))
	place(app.source, fieldX, px(104), fieldWidth, px(34))
	place(app.target, fieldX, px(154), fieldWidth, px(34))
	place(app.sourceBrowse, width-margin-buttonWidth, px(103), buttonWidth, px(35))
	place(app.targetBrowse, width-margin-buttonWidth, px(153), buttonWidth, px(35))
	place(app.hint, margin, px(202), width-2*margin, px(38))
	place(app.filesLabel, margin, px(246), width-2*margin, px(24))
	tableY := px(278)
	tableHeight := height - px(470)
	place(app.list, margin, tableY, width-2*margin, tableHeight)
	tableWidth := width - 2*margin - px(8)
	proportions := []float64{0.34, 0.09, 0.22, 0.35}
	for index, fraction := range proportions {
		sendMessage.Call(app.list, lvmFirst+30, uintptr(index), uintptr(int(float64(tableWidth)*fraction)))
	}
	actionY := height - px(167)
	place(app.start, margin, actionY, px(135), px(38))
	place(app.stop, margin+px(149), actionY, px(90), px(38))
	place(app.refresh, margin+px(254), actionY, px(140), px(38))
	place(app.open, width-margin-px(155), actionY, px(155), px(38))
	place(app.details, margin+px(410), actionY, px(110), px(38))
	place(app.help, margin+px(535), actionY, px(100), px(38))
	place(app.progress, margin, height-px(108), width-2*margin, px(18))
	place(app.stage, margin, height-px(74), width-2*margin, px(25))
	place(app.summary, margin, height-px(39), width-2*margin, px(24))
}

func (app *window) scan(showErrors bool) {
	if app.active {
		return
	}
	app.jobs = nil
	app.rowMessages = nil
	enabled(app.start, false)
	sendMessage.Call(app.list, lvmFirst+9, 0, 0)
	source, output := strings.TrimSpace(text(app.source)), strings.TrimSpace(text(app.target))
	if source == "" || output == "" {
		if showErrors {
			alert(app.handle, "请选择目录", "源文件夹和保存目录均不能为空。")
		}
		return
	}
	jobs, err := convert.Plan(source, output)
	if err != nil {
		if showErrors {
			alert(app.handle, "无法读取目录", err.Error())
		}
		setText(app.stage, err.Error())
		enabled(app.start, false)
		return
	}
	app.jobs = jobs
	app.rowMessages = make([]string, len(jobs))
	sendMessage.Call(app.list, lvmFirst+9, 0, 0)
	for index, job := range jobs {
		listCell(app.list, index, 0, filepath.Base(job.Source))
		listCell(app.list, index, 1, strings.ToUpper(job.Extension[1:]))
		listCell(app.list, index, 2, "等待转换")
		listCell(app.list, index, 3, filepath.Base(job.Destination))
		app.rowMessages[index] = job.Source + "\n\n保存到：" + job.Destination
	}
	setText(app.filesLabel, fmt.Sprintf("待转换文件  ·  %d 首", len(jobs)))
	setText(app.stage, fmt.Sprintf("找到 %d 个音频文件。确认目录后点击“开始转换”。", len(jobs)))
	sendMessage.Call(app.progress, 0x0402, 0, 0)
	enabled(app.start, len(jobs) > 0)
}

func (app *window) startBatch() {
	if app.active {
		return
	}
	app.scan(true)
	if len(app.jobs) == 0 {
		return
	}
	app.active = true
	app.success = 0
	app.skipped = 0
	app.failed = 0
	ctx, cancel := context.WithCancel(context.Background())
	app.cancel = cancel
	for _, handle := range []uintptr{app.source, app.target, app.sourceBrowse, app.targetBrowse, app.start, app.refresh} {
		enabled(handle, false)
	}
	enabled(app.stop, true)
	jobs := append([]convert.Job(nil), app.jobs...)
	go func() {
		summary := convert.Run(ctx, jobs, func(event convert.Event) {
			select {
			case app.updates <- event:
			case <-ctx.Done():
			}
		})
		app.done <- summary
	}()
}

func (app *window) poll() {
	for {
		select {
		case event := <-app.updates:
			app.display(event)
		default:
			goto finished
		}
	}
finished:
	select {
	case summary := <-app.done:
		app.finish(summary)
	default:
	}
}

func (app *window) display(event convert.Event) {
	if event.Index < 0 {
		setText(app.stage, event.Stage)
		return
	}
	if event.Index >= len(app.jobs) {
		return
	}
	fraction := min(1, max(0, event.Fraction))
	totalFraction := (float64(event.Index) + fraction) / float64(max(1, event.Total))
	sendMessage.Call(app.progress, 0x0402, uintptr(int(totalFraction*1000)), 0)
	listCell(app.list, event.Index, 2, event.Stage)
	sendMessage.Call(app.list, lvmFirst+19, uintptr(event.Index), 0)
	setText(app.stage, fmt.Sprintf("[%d / %d] %s  ·  %s  ·  总进度 %.0f%%", event.Index+1, event.Total, filepath.Base(app.jobs[event.Index].Source), event.Stage, totalFraction*100))
	if event.Result != nil {
		result := event.Result
		app.rowMessages[event.Index] = result.Job.Source + "\n\n" + result.Message + "\n\n目标：" + result.Job.Destination
		switch result.Status {
		case "converted":
			app.success++
		case "skipped":
			app.skipped++
		default:
			app.failed++
		}
		setText(app.summary, fmt.Sprintf("成功 %d  ·  跳过 %d  ·  失败 %d  |  选择条目后点击“查看详情”", app.success, app.skipped, app.failed))
	}
}

func (app *window) finish(summary convert.Summary) {
	app.active = false
	app.cancel = nil
	if app.closing {
		destroyWindow.Call(app.handle)
		return
	}
	for _, handle := range []uintptr{app.source, app.target, app.sourceBrowse, app.targetBrowse, app.start, app.refresh} {
		enabled(handle, true)
	}
	enabled(app.stop, false)
	state := "转换完成"
	if summary.Canceled {
		state = "已停止，已完成的文件保留"
	} else if summary.Failed > 0 {
		state = "处理结束，部分文件失败；请选择条目查看详情"
	}
	if summary.Error != "" {
		state = summary.Error
	}
	setText(app.stage, state)
	setText(app.summary, fmt.Sprintf("成功 %d  ·  跳过 %d  ·  失败 %d  ·  总计 %d  |  结果报告保存在目标目录", summary.Converted, summary.Skipped, summary.Failed, summary.Total))
	if !summary.Canceled && summary.Error == "" {
		sendMessage.Call(app.progress, 0x0402, 1000, 0)
	}
	if app.smokePath != "" {
		app.smokeError = app.snapshot(app.smokePath)
		if app.smokeError == nil && (summary.Failed > 0 || summary.Error != "") {
			app.smokeError = fmt.Errorf("界面运行自检失败：%s，失败 %d 首", summary.Error, summary.Failed)
		}
		destroyWindow.Call(app.handle)
	}
}

func (app *window) chooseSource() {
	previous := text(app.source)
	chosen := app.folder(previous)
	if chosen == "" {
		return
	}
	setText(app.source, chosen)
	if text(app.target) == filepath.Join(previous, "MP3") {
		setText(app.target, filepath.Join(chosen, "MP3"))
	}
	app.scan(true)
}

func (app *window) folder(initial string) string {
	display := make([]uint16, 32768)
	initialPath := wide(initial)
	browser := browseInfo{Owner: app.handle, DisplayName: &display[0], Title: wide("选择音乐文件夹"), Flags: 0x0001 | 0x0040 | 0x0010, Callback: folderCallback, Parameter: uintptr(unsafe.Pointer(initialPath))}
	id, _, _ := browseFolder.Call(uintptr(unsafe.Pointer(&browser)))
	runtime.KeepAlive(initialPath)
	if id == 0 {
		return ""
	}
	defer freeCOM.Call(id)
	path := make([]uint16, 32768)
	success, _, _ := folderPath.Call(id, uintptr(unsafe.Pointer(&path[0])))
	if success == 0 {
		return ""
	}
	return syscall.UTF16ToString(path)
}

func folderProcedure(handle uintptr, message uint32, parameter, initial uintptr) uintptr {
	if message == 1 {
		sendMessage.Call(handle, 0x0400+103, 1, initial)
	}
	return 0
}

func windowProcedure(handle uintptr, notificationCode uint32, first, second uintptr) uintptr {
	app := current
	if app == nil {
		result, _, _ := defaultProcedure.Call(handle, uintptr(notificationCode), first, second)
		return result
	}
	switch notificationCode {
	case wmSize:
		if first == 1 {
			return 0
		}
		app.layout()
		return 0
	case wmStaticColor:
		setBackgroundMode.Call(first, 1)
		setTextColor.Call(first, 0x00634829)
		return app.background
	case wmEditColor:
		setBackgroundColor.Call(first, 0x00ffffff)
		setTextColor.Call(first, 0x00382d22)
		return app.white
	case wmCommand:
		switch int(first & 0xffff) {
		case idSourceBrowse:
			app.chooseSource()
		case idTargetBrowse:
			if chosen := app.folder(text(app.target)); chosen != "" {
				setText(app.target, chosen)
				app.scan(true)
			}
		case idStart:
			app.startBatch()
		case idStop:
			if app.cancel != nil {
				app.cancel()
				enabled(app.stop, false)
				setText(app.stage, "正在停止，等待当前操作安全退出…")
			}
		case idRefresh:
			app.scan(true)
		case idDetails:
			selected, _, _ := sendMessage.Call(app.list, lvmFirst+12, ^uintptr(0), 2)
			if int32(selected) >= 0 && int(selected) < len(app.rowMessages) {
				alert(handle, "转换详情", app.rowMessages[selected])
			} else {
				alert(handle, "转换详情", "请先在列表中选择一个音乐文件。")
			}
		case idHelp:
			app.showHelp()
		case idOpen:
			target := text(app.target)
			if info, err := os.Stat(target); err != nil || !info.IsDir() {
				alert(handle, "保存目录", "目标文件夹尚不存在。完成转换后即可打开。")
				break
			}
			shellExecute.Call(handle, uintptr(unsafe.Pointer(wide("open"))), uintptr(unsafe.Pointer(wide(target))), 0, 0, 1)
		}
		return 0
	case wmTimer:
		app.poll()
		return 0
	case wmClose:
		if app.active {
			answer, _, _ := messageBox.Call(handle, uintptr(unsafe.Pointer(wide("转换尚未结束。停止转换并关闭？\n已完成的 MP3 会保留，原文件不会修改。"))), uintptr(unsafe.Pointer(wide("确认关闭"))), 0x24)
			if answer == 6 {
				// MessageBox runs a nested message loop: conversion may finish meanwhile.
				if app.active && app.cancel != nil {
					app.closing = true
					app.cancel()
					setText(app.stage, "正在安全停止…")
				} else {
					destroyWindow.Call(handle)
				}
			}
			return 0
		}
		destroyWindow.Call(handle)
		return 0
	case wmDestroy:
		postQuit.Call(0)
		return 0
	}
	result, _, _ := defaultProcedure.Call(handle, uintptr(notificationCode), first, second)
	return result
}

func ShowError(message string) { alert(0, "QQ 音乐转换工具", message) }
