//go:build windows

package qqmusic

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var accountPattern = regexp.MustCompile(`(?m)^\s*Uin=\s*(\d+)\s*$`)
var tokenPattern = regexp.MustCompile(`(?:"authst"|\\"authst\\"|authst=)[:=\s\\"]*([A-Za-z0-9_=-]{21,512})`)

func currentAccount() (string, error) {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		return "", errors.New("找不到 Windows APPDATA 目录")
	}
	config := filepath.Join(appdata, "Tencent", "QQMusic", "QQMusicServiceConfig.ini")
	content, err := os.ReadFile(config)
	if err != nil {
		return "", fmt.Errorf("无法读取 QQ 音乐账号配置 %s；请先登录", config)
	}
	match := accountPattern.FindSubmatch(bytes.TrimPrefix(content, []byte{0xef, 0xbb, 0xbf}))
	if len(match) < 2 {
		return "", errors.New("QQ 音乐配置没有当前账号；请先登录客户端")
	}
	return string(match[1]), nil
}

func processIDs() ([]uint32, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("无法检查 QQ 音乐进程: %w", err)
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	var pids []uint32
	err = windows.Process32First(snapshot, &entry)
	for err == nil {
		if strings.EqualFold(windows.UTF16ToString(entry.ExeFile[:]), "QQMusic.exe") {
			pids = append(pids, entry.ProcessID)
		}
		err = windows.Process32Next(snapshot, &entry)
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return nil, fmt.Errorf("QQ 音乐进程枚举失败: %w", err)
	}
	return pids, nil
}

func tokenCandidates(chunk []byte) []string {
	var found []string
	for _, match := range tokenPattern.FindAllSubmatch(chunk, -1) {
		found = append(found, string(match[1]))
	}
	marker := []byte{'a', 0, 'u', 0, 't', 0, 'h', 0, 's', 0, 't', 0}
	for position := 0; position < len(chunk); {
		relative := bytes.Index(chunk[position:], marker)
		if relative < 0 {
			break
		}
		start := position + relative
		encoded := chunk[max(0, start-4):min(len(chunk), start+2048)]
		units := make([]uint16, len(encoded)/2)
		for i := range units {
			units[i] = binary.LittleEndian.Uint16(encoded[2*i:])
		}
		decoded := []byte(string(utf16.Decode(units)))
		for _, match := range tokenPattern.FindAllSubmatch(decoded, -1) {
			found = append(found, string(match[1]))
		}
		position = start + len(marker)
	}
	return found
}

func scanTokens(ctx context.Context, pid uint32, accept func(string) (bool, error)) error {
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, pid)
	if err != nil {
		return fmt.Errorf("无法只读访问 QQMusic.exe（PID %d）；请检查客户端与工具的权限级别", pid)
	}
	defer windows.CloseHandle(process)
	name := make([]uint16, 32768)
	size := uint32(len(name))
	if err = windows.QueryFullProcessImageName(process, 0, &name[0], &size); err != nil {
		return fmt.Errorf("无法确认 PID %d 的程序名称", pid)
	}
	if !strings.EqualFold(filepath.Base(windows.UTF16ToString(name[:size])), "QQMusic.exe") {
		return fmt.Errorf("拒绝读取非 QQMusic.exe 进程 PID %d", pid)
	}
	deadline := time.Now().Add(90 * time.Second)
	seen := make(map[string]bool)
	buffer := make([]byte, 1024*1024)
	var address uintptr
	for {
		if err = ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return errors.New("QQ 音乐登录信息读取超过 90 秒")
		}
		var region windows.MemoryBasicInformation
		if err = windows.VirtualQueryEx(process, address, &region, unsafe.Sizeof(region)); err != nil {
			break
		}
		next := region.BaseAddress + region.RegionSize
		if next <= address {
			break
		}
		protection := region.Protect & 255
		readable := protection == 2 || protection == 4 || protection == 8 || protection == 32 || protection == 64 || protection == 128
		if region.State == windows.MEM_COMMIT && region.Protect&windows.PAGE_GUARD == 0 && readable {
			var carry []byte
			for offset := uintptr(0); offset < region.RegionSize; {
				if err = ctx.Err(); err != nil {
					return err
				}
				count := min(uintptr(len(buffer)), region.RegionSize-offset)
				var read uintptr
				err = windows.ReadProcessMemory(process, region.BaseAddress+offset, &buffer[0], count, &read)
				runtime.KeepAlive(buffer)
				if err == nil && read > 0 {
					joined := append(carry, buffer[:read]...)
					for _, token := range tokenCandidates(joined) {
						if seen[token] {
							continue
						}
						seen[token] = true
						accepted, err := accept(token)
						if err != nil {
							return err
						}
						if accepted {
							return nil
						}
						if len(seen) >= 8 {
							return nil
						}
					}
					carry = append(carry[:0], joined[max(0, len(joined)-2048):]...)
				} else {
					carry = nil
				}
				offset += count
			}
		}
		address = next
	}
	return nil
}
