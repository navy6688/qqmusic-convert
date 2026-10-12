//go:build windows

package convert

import (
	"context"
	"fmt"

	"golang.org/x/sys/windows"
)

// publish moves an already-verified file on the same volume without replacement.
func publish(ctx context.Context, candidate, destination string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	source, err := windows.UTF16PtrFromString(candidate)
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	if err = windows.MoveFile(source, target); err != nil {
		return fmt.Errorf("发布 MP3 %s 失败: %w", destination, err)
	}
	return nil
}
