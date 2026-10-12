//go:build windows

package desktop

import (
	"fmt"
	"unsafe"
)

const brandIconResourceID = 2 // rsrc: manifest is resource 1, the icon group is 2.

func brandIcon(instance uintptr, size int) (uintptr, error) {
	icon, _, err := user32.NewProc("LoadImageW").Call(instance, brandIconResourceID, 1, uintptr(size), uintptr(size), 0)
	if icon == 0 {
		return 0, fmt.Errorf("加载应用 Logo 失败；请通过 build.ps1 构建带图标资源的 EXE: %v", err)
	}
	return icon, nil
}

func (app *window) createLogo() error {
	icon, err := brandIcon(app.instance, int(52*app.scale))
	if err != nil {
		return err
	}
	app.brandIcon = icon
	handle, err := control(app.handle, app.instance, "STATIC", "", 0x0003|0x0040, 0, app.font)
	if err != nil {
		return err
	}
	app.logo = handle
	sendMessage.Call(handle, 0x0170, icon, 0)
	return nil
}

func (app *window) maximizeCheck() error {
	styleIndex := int32(-16)
	style, _, _ := user32.NewProc("GetWindowLongW").Call(app.handle, uintptr(styleIndex))
	if style&0x00010000 == 0 || style&0x00040000 == 0 {
		return fmt.Errorf("窗口没有启用最大化或调整尺寸")
	}
	// Exercise the same native action as the enabled title-bar maximize button.
	sendMessage.Call(app.handle, 0x0112, 0xf030, 0)
	maximized, _, _ := user32.NewProc("IsZoomed").Call(app.handle)
	if maximized == 0 {
		return fmt.Errorf("窗口最大化自检失败")
	}
	app.layout()
	return nil
}

// Pointer storage is owned by Win32; this function never reinterprets lParam.
func (app *window) minimumSize(width, height int) bool {
	minimumWidth, minimumHeight := int(850*app.scale), int(650*app.scale)
	if width >= minimumWidth && height >= minimumHeight {
		return false
	}
	var outer rectangle
	getWindowRect.Call(app.handle, uintptr(unsafe.Pointer(&outer)))
	borderWidth := int(outer.Right-outer.Left) - width
	borderHeight := int(outer.Bottom-outer.Top) - height
	user32.NewProc("SetWindowPos").Call(app.handle, 0, 0, 0,
		uintptr(max(width, minimumWidth)+borderWidth), uintptr(max(height, minimumHeight)+borderHeight), 0x0016)
	return true
}
