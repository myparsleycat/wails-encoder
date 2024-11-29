// encoder/app.go
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	wails_runtime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp은 새 앱 애플리케이션 구조를 생성합니다.
func NewApp() *App {
	return &App{}
}

// startup is called at application startup
func (a *App) startup(ctx context.Context) {
	// Perform your setup here
	a.ctx = ctx
}

// domReady는 프론트엔드 리소스가 로드된 후에 호출됩니다.
func (a App) domReady(ctx context.Context) {
	// Add your action here
}

// 창 닫기 버튼을 클릭하거나 runtime.Quit을 호출하여 애플리케이션을 종료하려고 할 때 호출됩니다.
// 참을 반환하면 애플리케이션이 계속 실행되고, 거짓을 반환하면 정상적으로 종료됩니다.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	return false
}

// shutdown은 애플리케이션 종료 시 호출됩니다.
func (a *App) shutdown(ctx context.Context) {
	// Perform your teardown here
}

func (a *App) ShowNotification(title, message string) error {
	switch runtime.GOOS {
	// case "windows":
	// return showWindowsNotification(title, message)
	case "darwin":
		return showMacNotification(title, message)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// Windows용 알림
// func showWindowsNotification(title, message string) error {
// 	notification := toast.Notification{
// 		AppID:   "Wails App", // Windows에서 식별할 앱 ID
// 		Title:   title,
// 		Message: message,
// 	}

// 	return notification.Push()
// }

// macOS용 알림
func showMacNotification(title, message string) error {
	script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// 지원하는 비디오 확장자 목록
var supportedExtensions = map[string]bool{
	".mp4":  true,
	".avi":  true,
	".mov":  true,
	".mkv":  true,
	".wmv":  true,
	".flv":  true,
	".webm": true,
}

// 파일 경로가 비디오 파일인지 확인
func isVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return supportedExtensions[ext]
}

type VideoMetadata struct {
	Name     string  `json:"name"`
	Size     int64   `json:"size"`
	Duration float64 `json:"duration"`
	Format   string  `json:"format"`
	Codec    string  `json:"codec"`
	Path     string  `json:"path"`
}

// 디렉토리를 재귀적으로 탐색하여 비디오 파일 찾기
func (a *App) FindVideoFiles(path string) ([]string, error) {
	var videos []string

	// 파일 정보 가져오기
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("파일 정보 읽기 실패: %v", err)
	}

	// 단일 파일인 경우
	if !fileInfo.IsDir() {
		if isVideoFile(path) {
			return []string{path}, nil
		}
		return nil, nil
	}

	// 디렉토리인 경우 재귀적으로 탐색
	err = filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 숨김 파일/폴더 무시
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() && isVideoFile(currentPath) {
			videos = append(videos, currentPath)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("디렉토리 탐색 중 오류 발생: %v", err)
	}

	return videos, nil
}

func (a *App) ProcessVideo(filePath string) (*VideoMetadata, error) {
	// 파일 이름 추출
	fileName := filepath.Base(filePath)

	// ffprobe로 비디오 정보 추출
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe 실행 중 오류 발생: %v", err)
	}

	// JSON 파싱을 위한 구조체
	var probe struct {
		Streams []struct {
			CodecName string `json:"codec_name"`
		} `json:"streams"`
		Format struct {
			Filename string `json:"filename"`
			Size     string `json:"size"`
			Duration string `json:"duration"`
			Format   string `json:"format_name"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &probe); err != nil {
		return nil, fmt.Errorf("JSON 파싱 중 오류 발생: %v", err)
	}

	// 동영상 길이를 float로 변환
	duration, _ := strconv.ParseFloat(probe.Format.Duration, 64)
	// 파일 크기를 int64로 변환
	size, _ := strconv.ParseInt(probe.Format.Size, 10, 64)

	// 비디오 코덱 찾기
	videoCodec := ""
	for _, stream := range probe.Streams {
		if stream.CodecName != "" {
			videoCodec = stream.CodecName
			break
		}
	}

	return &VideoMetadata{
		Name:     fileName,
		Size:     size,
		Duration: duration,
		Format:   strings.Split(probe.Format.Format, ",")[0], // 첫 번째 포맷만 사용
		Codec:    videoCodec,
		Path:     filePath,
	}, nil
}

// ProcessVideoPaths는 여러 경로에서 비디오 파일을 찾아 처리합니다
func (a *App) ProcessVideoPaths(paths []string) error {
	// 메타데이터를 전송할 채널 생성
	for _, path := range paths {
		// 각 경로에 대해 고루틴으로 처리
		go func(path string) {
			// 각 경로에 대해 비디오 파일 찾기
			videoPaths, err := a.FindVideoFiles(path)
			if err != nil {
				wails_runtime.EventsEmit(a.ctx, "video_error", map[string]string{
					"path":  path,
					"error": fmt.Sprintf("경로 처리 중 오류 발생: %v", err),
				})
				return
			}

			// 찾은 각 비디오 파일에 대해 메타데이터 추출 및 전송
			for _, videoPath := range videoPaths {
				metadata, err := a.ProcessVideo(videoPath)
				if err != nil {
					wails_runtime.EventsEmit(a.ctx, "video_error", map[string]string{
						"path":  videoPath,
						"error": fmt.Sprintf("비디오 처리 중 오류 발생: %v", err),
					})
					continue
				}

				// 메타데이터를 이벤트로 전송
				wails_runtime.EventsEmit(a.ctx, "video_processed", metadata)
			}
		}(path)
	}

	return nil
}

// CodecInfo는 코덱의 상세 정보를 담는 구조체
type CodecInfo struct {
	Name        string   `json:"name"`        // 코덱 이름 (예: h264, hevc 등)
	DisplayName string   `json:"displayName"` // 표시용 이름 (예: "H.264 (CPU)")
	Hardware    string   `json:"hardware"`    // 하드웨어 가속 종류 (cpu, nvidia, intel, apple)
	Formats     []string `json:"formats"`     // 지원하는 포맷 (mp4, webm 등)
}

// 시스템에서 사용 가능한 코덱 목록을 반환
func (a *App) GetAvailableCodecs() ([]CodecInfo, error) {
	var codecs []CodecInfo

	// 1. 기본 CPU 코덱 추가
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

	// 2. FFmpeg 코덱 목록 확인
	// 타임아웃 설정
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg", "-encoders")
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}
	}

	output, err := cmd.Output()
	if err != nil {
		return codecs, fmt.Errorf("ffmpeg encoder 목록 조회 실패 (기본 코덱만 사용 가능): %v", err)
	}
	encoderList := string(output)

	// 3. OS별 하드웨어 가속 코덱 확인
	switch runtime.GOOS {
	case "darwin":
		// macOS: VideoToolbox 지원 확인
		if strings.Contains(encoderList, "hevc_videotoolbox") {
			codecs = append(codecs, CodecInfo{
				Name:        "hevc_videotoolbox",
				DisplayName: "HEVC (Apple Silicon/Intel)",
				Hardware:    "apple",
				Formats:     []string{"mp4"},
			})
		}
		if strings.Contains(encoderList, "h264_videotoolbox") {
			codecs = append(codecs, CodecInfo{
				Name:        "h264_videotoolbox",
				DisplayName: "H.264 (Apple Silicon/Intel)",
				Hardware:    "apple",
				Formats:     []string{"mp4"},
			})
		}

	case "windows", "linux":
		// NVIDIA GPU 확인
		if hasNvidiaGPU() {
			if strings.Contains(encoderList, "hevc_nvenc") {
				codecs = append(codecs, CodecInfo{
					Name:        "hevc_nvenc",
					DisplayName: "HEVC (NVIDIA GPU)",
					Hardware:    "nvidia",
					Formats:     []string{"mp4"},
				})
			}
			if strings.Contains(encoderList, "h264_nvenc") {
				codecs = append(codecs, CodecInfo{
					Name:        "h264_nvenc",
					DisplayName: "H.264 (NVIDIA GPU)",
					Hardware:    "nvidia",
					Formats:     []string{"mp4"},
				})
			}
		}

		// Intel QSV 확인
		if hasIntelGPU() {
			if strings.Contains(encoderList, "hevc_qsv") {
				codecs = append(codecs, CodecInfo{
					Name:        "hevc_qsv",
					DisplayName: "HEVC (Intel QuickSync)",
					Hardware:    "intel",
					Formats:     []string{"mp4"},
				})
			}
			if strings.Contains(encoderList, "h264_qsv") {
				codecs = append(codecs, CodecInfo{
					Name:        "h264_qsv",
					DisplayName: "H.264 (Intel QuickSync)",
					Hardware:    "intel",
					Formats:     []string{"mp4"},
				})
			}
		}
	}

	// VP8/VP9 코덱 추가 (WebM 용)
	if strings.Contains(encoderList, "libvpx") {
		codecs = append(codecs, CodecInfo{
			Name:        "vp8",
			DisplayName: "VP8",
			Hardware:    "cpu",
			Formats:     []string{"webm"},
		})
	}
	if strings.Contains(encoderList, "libvpx-vp9") {
		codecs = append(codecs, CodecInfo{
			Name:        "vp9",
			DisplayName: "VP9",
			Hardware:    "cpu",
			Formats:     []string{"webm"},
		})
	}

	return codecs, nil
}

// NVIDIA GPU 존재 여부 확인
func hasNvidiaGPU() bool {
	// 명령어 실행 시 타임아웃 설정
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	switch runtime.GOOS {
	case "windows":
		cmd := exec.CommandContext(ctx, "nvidia-smi")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}
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
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}
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

// 품질 설정 방식
type QualityMode string

const (
	QualityModeCRF     QualityMode = "crf"     // Constant Rate Factor
	QualityModeBitrate QualityMode = "bitrate" // Target Bitrate
)

// 지원되는 포맷 정의
var supportedFormats = map[string][]string{
	"mp4":  {"h264", "h264_nvenc", "h264_qsv", "hevc", "hevc_nvenc", "hevc_qsv", "hevc_videotoolbox"},
	"webm": {"vp8", "vp9"},
}

// 코덱별 설정 정의
var codecSettings = map[string]struct {
	defaultMode  QualityMode
	qualityRange struct {
		min, max int
		default_ int
	}
}{
	"h264": {
		defaultMode: QualityModeCRF,
		qualityRange: struct{ min, max, default_ int }{
			min:      0,
			max:      51,
			default_: 23, // 일반적으로 사용되는 기본값
		},
	},
	"hevc": {
		defaultMode: QualityModeCRF,
		qualityRange: struct{ min, max, default_ int }{
			min:      0,
			max:      51,
			default_: 28, // HEVC의 일반적인 기본값
		},
	},
	"vp9": {
		defaultMode: QualityModeCRF,
		qualityRange: struct{ min, max, default_ int }{
			min:      0,
			max:      63,
			default_: 31, // VP9의 일반적인 기본값
		},
	},
}

type EncodingOptions struct {
	VideoFormat  string      `json:"videoformat"`
	VideoCodec   string      `json:"videocodec"`
	QualityMode  QualityMode `json:"qualitymode"`
	QualityValue int         `json:"qualityvalue"` // CRF 값 또는 비트레이트(kbps)
	Use2Pass     bool        `json:"use2pass"`     // 비트레이트 모드에서만 사용

	// 크기 조정 옵션
	IsResize bool `json:"isresize"`
	Width    int  `json:"width"`
	Height   int  `json:"height"`

	// 출력 경로 옵션
	OutputPath string `json:"outputpath"`
	Prefix     string `json:"prefix"`
	Postfix    string `json:"postfix"`

	// 오디오 옵션
	AudioCodec      string `json:"audiocodec"`
	AudioBitrate    int    `json:"audiobitrate"`
	AudioSamplerate int    `json:"audiosamplerate"`
}

// 출력 파일 경로를 생성하는 함수
func (opts *EncodingOptions) getOutputPath(inputPath string) string {
	if opts.OutputPath != "" {
		return opts.OutputPath
	}

	dir := filepath.Dir(inputPath)
	filename := filepath.Base(inputPath)
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	newName := nameWithoutExt
	if opts.Prefix != "" {
		newName = opts.Prefix + newName
	}
	if opts.Postfix != "" {
		newName = newName + opts.Postfix
	}

	return filepath.Join(dir, newName+"."+opts.VideoFormat)
}

func (opts *EncodingOptions) Validate() error {
	// 1. 포맷 검증
	supportedCodecs, formatOk := supportedFormats[opts.VideoFormat]
	if !formatOk {
		return fmt.Errorf("unsupported video format: %s", opts.VideoFormat)
	}

	// 2. 코덱 검증
	codecSupported := false
	for _, codec := range supportedCodecs {
		if opts.VideoCodec == codec {
			codecSupported = true
			break
		}
	}
	if !codecSupported {
		return fmt.Errorf("unsupported codec %s for format %s", opts.VideoCodec, opts.VideoFormat)
	}

	// 3. 코덱 설정 검증
	codecSet, exists := codecSettings[opts.VideoCodec]
	if !exists {
		// 하드웨어 가속 코덱 등 특별한 경우는 기본 코덱의 설정을 사용
		baseCodec := strings.Split(opts.VideoCodec, "_")[0]
		codecSet, exists = codecSettings[baseCodec]
		if !exists {
			return fmt.Errorf("no settings found for codec: %s", opts.VideoCodec)
		}
	}

	// 4. 품질 모드 및 값 검증
	if opts.QualityValue == 0 {
		// 기본값 설정
		opts.QualityMode = codecSet.defaultMode
		opts.QualityValue = codecSet.qualityRange.default_
	}

	if opts.QualityValue < codecSet.qualityRange.min ||
		opts.QualityValue > codecSet.qualityRange.max {
		return fmt.Errorf("quality value %d out of range [%d-%d] for codec %s",
			opts.QualityValue,
			codecSet.qualityRange.min,
			codecSet.qualityRange.max,
			opts.VideoCodec)
	}

	// 5. 2-pass 설정 검증
	if opts.Use2Pass && opts.QualityMode != QualityModeBitrate {
		return fmt.Errorf("2-pass encoding is only available with bitrate mode")
	}

	return nil
}

func (opts *EncodingOptions) BuildFFmpegArgs(inputPath string) ([]string, error) {
	args := []string{"-i", inputPath}

	// 비디오 코덱 설정
	args = append(args, "-c:v", opts.VideoCodec)

	// 품질 설정
	switch opts.QualityMode {
	case QualityModeCRF:
		args = append(args, "-crf", fmt.Sprintf("%d", opts.QualityValue))
	case QualityModeBitrate:
		args = append(args, "-b:v", fmt.Sprintf("%dk", opts.QualityValue))
	}

	// 크기 조정 설정
	if opts.IsResize && opts.Width > 0 && opts.Height > 0 {
		args = append(args, "-vf", fmt.Sprintf("scale=%d:%d", opts.Width, opts.Height))
	}

	// 오디오 설정
	if opts.AudioCodec != "" {
		args = append(args, "-c:a", opts.AudioCodec)
	} else {
		args = append(args, "-c:a", "copy")
	}
	if opts.AudioBitrate > 0 {
		args = append(args, "-b:a", fmt.Sprintf("%dk", opts.AudioBitrate))
	}
	if opts.AudioSamplerate > 0 {
		args = append(args, "-ar", fmt.Sprintf("%d", opts.AudioSamplerate))
	}

	return args, nil
}

func (opts *EncodingOptions) Build2PassArgs(inputPath string, passLogFile string) ([]string, []string) {
	// 1차 패스 인자
	pass1Args := []string{
		"-i", inputPath,
		"-c:v", opts.VideoCodec,
		"-b:v", fmt.Sprintf("%dk", opts.QualityValue),
		"-pass", "1",
		"-passlogfile", passLogFile,
		"-an", // 1차 패스에서는 오디오 처리 제외
		"-f", "null",
	}
	if opts.IsResize && opts.Width > 0 && opts.Height > 0 {
		pass1Args = append(pass1Args, "-vf", fmt.Sprintf("scale=%d:%d", opts.Width, opts.Height))
	}
	pass1Args = append(pass1Args, os.DevNull)

	// 2차 패스 인자
	pass2Args := []string{
		"-i", inputPath,
		"-c:v", opts.VideoCodec,
		"-b:v", fmt.Sprintf("%dk", opts.QualityValue),
		"-pass", "2",
		"-passlogfile", passLogFile,
	}
	if opts.IsResize && opts.Width > 0 && opts.Height > 0 {
		pass2Args = append(pass2Args, "-vf", fmt.Sprintf("scale=%d:%d", opts.Width, opts.Height))
	}

	// 오디오 설정 (2차 패스에만 추가)
	if opts.AudioCodec != "" {
		pass2Args = append(pass2Args, "-c:a", opts.AudioCodec)
	} else {
		pass2Args = append(pass2Args, "-c:a", "copy")
	}
	if opts.AudioBitrate > 0 {
		pass2Args = append(pass2Args, "-b:a", fmt.Sprintf("%dk", opts.AudioBitrate))
	}
	if opts.AudioSamplerate > 0 {
		pass2Args = append(pass2Args, "-ar", fmt.Sprintf("%d", opts.AudioSamplerate))
	}

	return pass1Args, pass2Args
}

// 프로그레스 업데이트를 프론트엔드로 전송하는 함수
func (a *App) EmitProgress(progress EncodingProgress) {
	wails_runtime.EventsEmit(a.ctx, "encoding_progress", progress)
}

// 프로그레스 관련 정규식들
var frameRegex = regexp.MustCompile(`frame=\s*(\d+)`)
var fpsRegex = regexp.MustCompile(`fps=\s*(\d+)`)
var timeRegex = regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2}\.\d{2})`)
var sizeRegex = regexp.MustCompile(`size=\s*(\d+)kB`)
var bitrateRegex = regexp.MustCompile(`bitrate=\s*(\d+\.\d+)kbits/s`)
var speedRegex = regexp.MustCompile(`speed=\s*(\d+\.\d+)x`)

// 진행 상황을 위한 구조체
type EncodingProgress struct {
	Filename string  `json:"filename"`
	Frame    int     `json:"frame"`    // 현재 프레임
	FPS      int     `json:"fps"`      // 현재 FPS
	Time     string  `json:"time"`     // 현재 처리된 시간 (HH:MM:SS.MS 형식)
	Size     int     `json:"size"`     // 현재 파일 크기 (KB)
	Bitrate  float64 `json:"bitrate"`  // 비트레이트 (kbits/s)
	Speed    float64 `json:"speed"`    // 인코딩 속도
	Progress float64 `json:"progress"` // 진행률 (0-100)
	Status   string  `json:"status"`
}

// FFmpeg 출력을 읽는 reader 구조체
type ProgressReader struct {
	app          *App
	filename     string
	lastProgress EncodingProgress
	scanner      *bufio.Scanner
}

// func parseTime(str string) float64 {
// 	matches := timeRegex.FindStringSubmatch(str)
// 	if len(matches) < 4 {
// 		return 0
// 	}
// 	h, _ := strconv.Atoi(matches[1])
// 	m, _ := strconv.Atoi(matches[2])
// 	s, _ := strconv.ParseFloat(matches[3], 64)
// 	return float64(h)*3600 + float64(m)*60 + s
// }

func (a *App) StartEncodingWithOptions(paths []string, options EncodingOptions) error {
	if err := options.Validate(); err != nil {
		return fmt.Errorf("인코딩 옵션이 올바르지 않습니다: %w", err)
	}

	// FFmpeg 존재 여부 확인
	cmd := exec.Command("ffmpeg", "-h")
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("FFmpeg가 설치되어 있지 않습니다: %w", err)
	}

	for _, inputPath := range paths {
		// 입력 파일 존재 여부 확인
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			return fmt.Errorf("입력 파일을 찾을 수 없습니다 (%s): %w", inputPath, err)
		}

		filename := filepath.Base(inputPath)
		outputPath := options.getOutputPath(inputPath)

		// 출력 디렉토리 쓰기 권한 확인
		outputDir := filepath.Dir(outputPath)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("출력 디렉토리 생성 실패 (%s): %w", outputDir, err)
		}

		// 출력 파일 이미 존재하는지 확인
		if _, err := os.Stat(outputPath); err == nil {
			return fmt.Errorf("출력 파일이 이미 존재합니다: %s", outputPath)
		}

		// 초기 상태 업데이트
		a.EmitProgress(EncodingProgress{
			Filename: filename,
			Status:   "진행중",
		})

		var cmd *exec.Cmd
		// var cmdErr error

		if options.Use2Pass && options.QualityMode == QualityModeBitrate {
			// 2-pass 인코딩
			passLogFile := filepath.Join(os.TempDir(), fmt.Sprintf("ffmpeg2pass_%d", time.Now().UnixNano()))

			// 1차 패스
			pass1Args, pass2Args := options.Build2PassArgs(inputPath, passLogFile)

			// 명령어 로깅
			fmt.Printf("1차 패스 명령어: ffmpeg %s\n", strings.Join(pass1Args, " "))

			cmd = exec.Command("ffmpeg", pass1Args...)
			if runtime.GOOS == "windows" {
				cmd.SysProcAttr = &syscall.SysProcAttr{
					HideWindow:    true,
					CreationFlags: 0x08000000,
				}
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				return fmt.Errorf("1차 패스 파이프 생성 실패: %w", err)
			}

			progressReader := NewProgressReader(a, filename)
			go io.Copy(progressReader, stderr)

			if err := cmd.Start(); err != nil {
				return fmt.Errorf("1차 패스 시작 실패: %w", err)
			}

			if err := cmd.Wait(); err != nil {
				// 명령어 실행 결과 포함
				if exitErr, ok := err.(*exec.ExitError); ok {
					return fmt.Errorf("1차 패스 실패: %w\n오류 출력:\n%s", err, string(exitErr.Stderr))
				}
				return fmt.Errorf("1차 패스 실패: %w", err)
			}

			// 2차 패스
			fmt.Printf("2차 패스 명령어: ffmpeg %s\n", strings.Join(pass2Args, " "))

			cmd = exec.Command("ffmpeg", pass2Args...)
			if runtime.GOOS == "windows" {
				cmd.SysProcAttr = &syscall.SysProcAttr{
					HideWindow:    true,
					CreationFlags: 0x08000000,
				}
			}
			stderr, err = cmd.StderrPipe()
			if err != nil {
				return fmt.Errorf("2차 패스 파이프 생성 실패: %w", err)
			}

			progressReader = NewProgressReader(a, filename)
			go io.Copy(progressReader, stderr)

			if err := cmd.Start(); err != nil {
				return fmt.Errorf("2차 패스 시작 실패: %w", err)
			}

			if err := cmd.Wait(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					return fmt.Errorf("2차 패스 실패: %w\n오류 출력:\n%s", err, string(exitErr.Stderr))
				}
				return fmt.Errorf("2차 패스 실패: %w", err)
			}

			// 임시 파일 정리
			os.Remove(passLogFile + "-0.log")
			os.Remove(passLogFile + "-0.log.mbtree")
		} else {
			// 일반 인코딩
			args, err := options.BuildFFmpegArgs(inputPath)
			if err != nil {
				return fmt.Errorf("FFmpeg 인자 생성 실패: %w", err)
			}
			args = append(args, outputPath)

			// 명령어 로깅
			fmt.Printf("인코딩 명령어: ffmpeg %s\n", strings.Join(args, " "))

			cmd = exec.Command("ffmpeg", args...)
			if runtime.GOOS == "windows" {
				cmd.SysProcAttr = &syscall.SysProcAttr{
					HideWindow:    true,
					CreationFlags: 0x08000000,
				}
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				return fmt.Errorf("stderr 파이프 생성 실패: %w", err)
			}

			progressReader := NewProgressReader(a, filename)
			go io.Copy(progressReader, stderr)

			if err := cmd.Start(); err != nil {
				return fmt.Errorf("인코딩 시작 실패: %w", err)
			}

			if err := cmd.Wait(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					return fmt.Errorf("인코딩 실패: %w\n오류 출력:\n%s", err, string(exitErr.Stderr))
				}
				return fmt.Errorf("인코딩 실패: %w", err)
			}
		}

		// 출력 파일 확인
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			return fmt.Errorf("인코딩된 파일을 찾을 수 없습니다: %s", outputPath)
		}

		// 완료 상태 업데이트
		a.EmitProgress(EncodingProgress{
			Filename: filename,
			Status:   "완료",
		})
	}

	return nil
}

func NewProgressReader(app *App, filename string) *ProgressReader {
	return &ProgressReader{
		app:      app,
		filename: filename,
	}
}

func (pr *ProgressReader) Write(p []byte) (n int, err error) {
	if pr.scanner == nil {
		pr.scanner = bufio.NewScanner(bytes.NewReader(p))
		pr.scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			if atEOF && len(data) == 0 {
				return 0, nil, nil
			}

			// \r 또는 \n을 발견하면 그 위치까지를 하나의 라인으로 처리
			if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
				return i + 1, data[0:i], nil
			}

			if atEOF {
				return len(data), data, nil
			}

			return 0, nil, nil
		})
	} else {
		pr.scanner = bufio.NewScanner(bytes.NewReader(p))
	}

	for pr.scanner.Scan() {
		line := pr.scanner.Text()
		// 빈 라인 무시
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		progress := EncodingProgress{
			Filename: pr.filename,
			Status:   "진행중",
		}

		// 각 정보 파싱
		if matches := frameRegex.FindStringSubmatch(line); len(matches) > 1 {
			progress.Frame, _ = strconv.Atoi(matches[1])
		}

		if matches := fpsRegex.FindStringSubmatch(line); len(matches) > 1 {
			progress.FPS, _ = strconv.Atoi(matches[1])
		}

		if matches := timeRegex.FindStringSubmatch(line); len(matches) > 1 {
			progress.Time = matches[0][5:] // "time=" 부분 제거
		}

		if matches := sizeRegex.FindStringSubmatch(line); len(matches) > 1 {
			progress.Size, _ = strconv.Atoi(matches[1])
		}

		if matches := bitrateRegex.FindStringSubmatch(line); len(matches) > 1 {
			progress.Bitrate, _ = strconv.ParseFloat(matches[1], 64)
		}

		if matches := speedRegex.FindStringSubmatch(line); len(matches) > 1 {
			progress.Speed, _ = strconv.ParseFloat(matches[1], 64)
		}

		// ffmpeg 진행 상황 라인인 경우에만 이벤트 발송
		if progress.Time != "" || progress.Frame > 0 {
			pr.app.EmitProgress(progress)
			pr.lastProgress = progress
		}
	}

	return len(p), nil
}
