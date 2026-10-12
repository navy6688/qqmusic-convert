//go:build windows

package desktop

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"unsafe"
)

type bitmapHeader struct {
	Size                   uint32
	Width, Height          int32
	Planes, Bits           uint16
	Compression, ImageSize uint32
	XPixels, YPixels       int32
	Colors, Important      uint32
}
type bitmapInfo struct {
	Header bitmapHeader
	Color  uint32
}

func (app *window) snapshot(path string) error {
	var bounds rectangle
	getWindowRect.Call(app.handle, uintptr(unsafe.Pointer(&bounds)))
	width, height := int(bounds.Right-bounds.Left), int(bounds.Bottom-bounds.Top)
	if width <= 0 || height <= 0 {
		return fmt.Errorf("无效窗口截图尺寸 %d x %d", width, height)
	}
	dc, _, err := gdi32.NewProc("CreateCompatibleDC").Call(0)
	if dc == 0 {
		return fmt.Errorf("创建界面截图 DC 失败: %v", err)
	}
	defer gdi32.NewProc("DeleteDC").Call(dc)
	info := bitmapInfo{Header: bitmapHeader{Size: 40, Width: int32(width), Height: -int32(height), Planes: 1, Bits: 32}}
	var pixels unsafe.Pointer
	bitmap, _, err := gdi32.NewProc("CreateDIBSection").Call(dc, uintptr(unsafe.Pointer(&info)), 0, uintptr(unsafe.Pointer(&pixels)), 0, 0)
	if bitmap == 0 {
		return fmt.Errorf("创建界面截图位图失败: %v", err)
	}
	defer deleteObject.Call(bitmap)
	previous, _, _ := gdi32.NewProc("SelectObject").Call(dc, bitmap)
	defer gdi32.NewProc("SelectObject").Call(dc, previous)
	sendMessage.Call(app.handle, 0x0317, dc, 0x0000003e)
	encoded := unsafe.Slice((*byte)(pixels), width*height*4)
	snapshot := image.NewRGBA(image.Rect(0, 0, width, height))
	for row := range height {
		for column := range width {
			offset := (row*width + column) * 4
			snapshot.SetRGBA(column, row, color.RGBA{R: encoded[offset+2], G: encoded[offset+1], B: encoded[offset], A: 255})
		}
	}
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err = png.Encode(file, snapshot); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	position, _, _ := sendMessage.Call(app.progress, 0x0408, 0, 0)
	maximized, _, _ := user32.NewProc("IsZoomed").Call(app.handle)
	styleIndex := int32(-16)
	style, _, _ := user32.NewProc("GetWindowLongW").Call(app.handle, uintptr(styleIndex))
	state := map[string]any{"maximized": maximized != 0, "maximize_button_enabled": style&0x00010000 != 0, "resizable": style&0x00040000 != 0, "logo_loaded": app.brandIcon != 0, "progress_percent": float64(position) / 10, "source": text(app.source), "destination": text(app.target), "stage": text(app.stage), "summary": text(app.summary), "files": len(app.jobs), "width": width, "height": height}
	report, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path+".json", report, 0644)
}
