//go:build windows

package media

import (
	"context"
	"os/exec"
	"syscall"
)

func Command(ctx context.Context, path string, arguments ...string) *exec.Cmd {
	command := exec.CommandContext(ctx, path, arguments...)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return command
}
