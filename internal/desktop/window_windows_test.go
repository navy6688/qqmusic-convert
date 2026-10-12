//go:build windows

package desktop

import "testing"

func TestWindowStyleAllowsNativeMaximizeAndResize(t *testing.T) {
	if mainWindowStyle&0x00010000 == 0 {
		t.Fatal("maximize box is missing")
	}
	if mainWindowStyle&0x00040000 == 0 {
		t.Fatal("sizing frame is missing")
	}
}
