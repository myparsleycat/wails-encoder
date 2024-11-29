// pkg/codec/hardware.go
package codec

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// NVIDIA GPU 존재 여부 확인
func hasNvidiaGPU() bool {
	// 명령어 실행 시 타임아웃 설정
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	switch runtime.GOOS {
	case "windows":
		cmd := exec.CommandContext(ctx, "nvidia-smi")
		return cmd.Run() == nil
	case "linux":
		cmd := exec.CommandContext(ctx, "lspci")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(strings.ToLower(string(output)), "nvidia")
	}
	return false
}

// Intel GPU 존재 여부 확인
func hasIntelGPU() bool {
	// 명령어 실행 시 타임아웃 설정
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	switch runtime.GOOS {
	case "windows":
		// Windows의 경우 wmic로 Intel Graphics 확인
		cmd := exec.CommandContext(ctx, "wmic", "path", "win32_VideoController", "get", "name")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(strings.ToLower(string(output)), "intel") &&
			strings.Contains(strings.ToLower(string(output)), "graphics")
	case "linux":
		cmd := exec.CommandContext(ctx, "lspci")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(strings.ToLower(string(output)), "intel") &&
			strings.Contains(strings.ToLower(string(output)), "graphics")
	}
	return false
}
