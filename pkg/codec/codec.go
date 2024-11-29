// pkg/codec/codec.go
package codec

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type CodecInfo struct {
	Name        string   `json:"name"`        // 코덱 이름 (예: h264, hevc 등)
	DisplayName string   `json:"displayName"` // 표시용 이름 (예: "H.264 (CPU)")
	Hardware    string   `json:"hardware"`    // 하드웨어 가속 종류 (cpu, nvidia, intel, apple)
	Formats     []string `json:"formats"`     // 지원하는 포맷 (mp4, webm 등)
}

// 지원되는 포맷 정의
var SupportedFormats = map[string][]string{
	"mp4":  {"h264", "h264_nvenc", "h264_qsv", "hevc", "hevc_nvenc", "hevc_qsv", "hevc_videotoolbox"},
	"webm": {"vp8", "vp9"},
}

// GetAvailable returns the list of available codecs on the system
func GetAvailable() ([]CodecInfo, error) {
	var codecs []CodecInfo

	// 기본 CPU 코덱 추가
	codecs = append(codecs, []CodecInfo{
		{
			Name:        "h264",
			DisplayName: "H.264 (CPU)",
			Hardware:    "cpu",
			Formats:     []string{"mp4"},
		},
		{
			Name:        "hevc",
			DisplayName: "HEVC (CPU)",
			Hardware:    "cpu",
			Formats:     []string{"mp4"},
		},
	}...)

	// FFmpeg 코덱 목록 확인
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg", "-encoders")
	output, err := cmd.Output()
	if err != nil {
		return codecs, fmt.Errorf("failed to get ffmpeg encoder list (using default codecs only): %v", err)
	}
	encoderList := string(output)

	// OS별 하드웨어 가속 코덱 확인
	switch runtime.GOOS {
	case "darwin":
		checkMacCodecs(&codecs, encoderList)
	case "windows", "linux":
		checkWindowsLinuxCodecs(&codecs, encoderList)
	}

	// VP8/VP9 코덱 확인
	checkVPCodecs(&codecs, encoderList)

	return codecs, nil
}

func checkMacCodecs(codecs *[]CodecInfo, encoderList string) {
	if strings.Contains(encoderList, "hevc_videotoolbox") {
		*codecs = append(*codecs, CodecInfo{
			Name:        "hevc_videotoolbox",
			DisplayName: "HEVC (Apple Silicon/Intel)",
			Hardware:    "apple",
			Formats:     []string{"mp4"},
		})
	}
	if strings.Contains(encoderList, "h264_videotoolbox") {
		*codecs = append(*codecs, CodecInfo{
			Name:        "h264_videotoolbox",
			DisplayName: "H.264 (Apple Silicon/Intel)",
			Hardware:    "apple",
			Formats:     []string{"mp4"},
		})
	}
}

func checkWindowsLinuxCodecs(codecs *[]CodecInfo, encoderList string) {
	if hasNvidiaGPU() {
		if strings.Contains(encoderList, "hevc_nvenc") {
			*codecs = append(*codecs, CodecInfo{
				Name:        "hevc_nvenc",
				DisplayName: "HEVC (NVIDIA GPU)",
				Hardware:    "nvidia",
				Formats:     []string{"mp4"},
			})
		}
		if strings.Contains(encoderList, "h264_nvenc") {
			*codecs = append(*codecs, CodecInfo{
				Name:        "h264_nvenc",
				DisplayName: "H.264 (NVIDIA GPU)",
				Hardware:    "nvidia",
				Formats:     []string{"mp4"},
			})
		}
	}

	if hasIntelGPU() {
		if strings.Contains(encoderList, "hevc_qsv") {
			*codecs = append(*codecs, CodecInfo{
				Name:        "hevc_qsv",
				DisplayName: "HEVC (Intel QuickSync)",
				Hardware:    "intel",
				Formats:     []string{"mp4"},
			})
		}
		if strings.Contains(encoderList, "h264_qsv") {
			*codecs = append(*codecs, CodecInfo{
				Name:        "h264_qsv",
				DisplayName: "H.264 (Intel QuickSync)",
				Hardware:    "intel",
				Formats:     []string{"mp4"},
			})
		}
	}
}

func checkVPCodecs(codecs *[]CodecInfo, encoderList string) {
	if strings.Contains(encoderList, "libvpx") {
		*codecs = append(*codecs, CodecInfo{
			Name:        "vp8",
			DisplayName: "VP8",
			Hardware:    "cpu",
			Formats:     []string{"webm"},
		})
	}
	if strings.Contains(encoderList, "libvpx-vp9") {
		*codecs = append(*codecs, CodecInfo{
			Name:        "vp9",
			DisplayName: "VP9",
			Hardware:    "cpu",
			Formats:     []string{"webm"},
		})
	}
}
