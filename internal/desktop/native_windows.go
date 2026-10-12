//go:build windows

package desktop

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32             = windows.NewLazySystemDLL("user32.dll")
	gdi32              = windows.NewLazySystemDLL("gdi32.dll")
	comctl32           = windows.NewLazySystemDLL("comctl32.dll")
	shell32            = windows.NewLazySystemDLL("shell32.dll")
	ole32              = windows.NewLazySystemDLL("ole32.dll")
	kernel32           = windows.NewLazySystemDLL("kernel32.dll")
	registerClass      = user32.NewProc("RegisterClassExW")
	createWindow       = user32.NewProc("CreateWindowExW")
	defaultProcedure   = user32.NewProc("DefWindowProcW")
	getMessage         = user32.NewProc("GetMessageW")
	translateMessage   = user32.NewProc("TranslateMessage")
	dispatchMessage    = user32.NewProc("DispatchMessageW")
	sendMessage        = user32.NewProc("SendMessageW")
	setWindowText      = user32.NewProc("SetWindowTextW")
	windowTextLength   = user32.NewProc("GetWindowTextLengthW")
	getWindowText      = user32.NewProc("GetWindowTextW")
	moveWindow         = user32.NewProc("MoveWindow")
	getClientRect      = user32.NewProc("GetClientRect")
	getWindowRect      = user32.NewProc("GetWindowRect")
	showWindow         = user32.NewProc("ShowWindow")
	updateWindow       = user32.NewProc("UpdateWindow")
	enableWindow       = user32.NewProc("EnableWindow")
	destroyWindow      = user32.NewProc("DestroyWindow")
	postQuit           = user32.NewProc("PostQuitMessage")
	messageBox         = user32.NewProc("MessageBoxW")
	loadCursor         = user32.NewProc("LoadCursorW")
	setTimer           = user32.NewProc("SetTimer")
	getMetrics         = user32.NewProc("GetSystemMetrics")
	getDPI             = user32.NewProc("GetDpiForWindow")
	setDPIAware        = user32.NewProc("SetProcessDpiAwarenessContext")
	createFont         = gdi32.NewProc("CreateFontW")
	createBrush        = gdi32.NewProc("CreateSolidBrush")
	deleteObject       = gdi32.NewProc("DeleteObject")
	setTextColor       = gdi32.NewProc("SetTextColor")
	setBackgroundMode  = gdi32.NewProc("SetBkMode")
	setBackgroundColor = gdi32.NewProc("SetBkColor")
	initializeControls = comctl32.NewProc("InitCommonControlsEx")
	browseFolder       = shell32.NewProc("SHBrowseForFolderW")
	folderPath         = shell32.NewProc("SHGetPathFromIDListW")
	shellExecute       = shell32.NewProc("ShellExecuteW")
	initializeCOM      = ole32.NewProc("CoInitializeEx")
	uninitializeCOM    = ole32.NewProc("CoUninitialize")
	freeCOM            = ole32.NewProc("CoTaskMemFree")
	moduleHandle       = kernel32.NewProc("GetModuleHandleW")
)

const (
	wmDestroy     = 0x0002
	wmSize        = 0x0005
	wmClose       = 0x0010
	wmCommand     = 0x0111
	wmTimer       = 0x0113
	wmStaticColor = 0x0138
	wmEditColor   = 0x0133
	wmSetFont     = 0x0030
	wsChild       = 0x40000000
	wsVisible     = 0x10000000
	wsTabStop     = 0x00010000
	lvmFirst      = 0x1000
)

type point struct{ X, Y int32 }
type rectangle struct{ Left, Top, Right, Bottom int32 }
type message struct {
	Window         uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Position       point
	Private        uint32
}
type windowClass struct {
	Size, Style                        uint32
	Procedure                          uintptr
	ClassExtra, WindowExtra            int32
	Instance, Icon, Cursor, Background uintptr
	MenuName, ClassName                *uint16
	SmallIcon                          uintptr
}
type commonControls struct{ Size, Classes uint32 }

type listColumn struct {
	Mask                                                                  uint32
	Format, Width                                                         int32
	Text                                                                  *uint16
	TextLength, SubItem, Image, Order, MinWidth, DefaultWidth, IdealWidth int32
}

type listItem struct {
	Mask                        uint32
	Index, SubItem              int32
	State, StateMask            uint32
	Text                        *uint16
	TextLength, Image           int32
	Parameter                   uintptr
	Indent, GroupID             int32
	Columns                     uint32
	ColumnPointer, ColumnFormat uintptr
	Group                       int32
}

type browseInfo struct {
	Owner, Root         uintptr
	DisplayName, Title  *uint16
	Flags               uint32
	Callback, Parameter uintptr
	Image               int32
}

func wide(value string) *uint16 {
	encoded, err := windows.UTF16PtrFromString(value)
	if err != nil {
		panic("Windows UI 字符串包含空字符")
	}
	return encoded
}

func setText(window uintptr, value string) {
	setWindowText.Call(window, uintptr(unsafe.Pointer(wide(value))))
}
func text(window uintptr) string {
	length, _, _ := windowTextLength.Call(window)
	encoded := make([]uint16, length+1)
	getWindowText.Call(window, uintptr(unsafe.Pointer(&encoded[0])), uintptr(len(encoded)))
	return windows.UTF16ToString(encoded)
}

func control(parent, instance uintptr, class, label string, style uintptr, id int, font uintptr) (uintptr, error) {
	handle, _, err := createWindow.Call(0, uintptr(unsafe.Pointer(wide(class))), uintptr(unsafe.Pointer(wide(label))), wsChild|wsVisible|style, 0, 0, 0, 0, parent, uintptr(id), instance, 0)
	if handle == 0 {
		return 0, fmt.Errorf("创建界面控件 %s 失败: %v", class, err)
	}
	sendMessage.Call(handle, wmSetFont, font, 1)
	return handle, nil
}

func place(handle uintptr, x, y, width, height int) {
	moveWindow.Call(handle, uintptr(x), uintptr(y), uintptr(max(1, width)), uintptr(max(1, height)), 1)
}
func enabled(handle uintptr, active bool) {
	value := uintptr(0)
	if active {
		value = 1
	}
	enableWindow.Call(handle, value)
}
func alert(owner uintptr, title, body string) {
	messageBox.Call(owner, uintptr(unsafe.Pointer(wide(body))), uintptr(unsafe.Pointer(wide(title))), 0x40)
}

func listCell(list uintptr, index, column int, value string) {
	item := listItem{Mask: 1, Index: int32(index), SubItem: int32(column), Text: wide(value)}
	if column == 0 {
		sendMessage.Call(list, lvmFirst+77, 0, uintptr(unsafe.Pointer(&item)))
	} else {
		sendMessage.Call(list, lvmFirst+116, uintptr(index), uintptr(unsafe.Pointer(&item)))
	}
}
