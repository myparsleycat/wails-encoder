// pkg/encoder/encoder.go
package encoder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Encoder struct {
	ctx context.Context
}

func NewEncoder(ctx context.Context) *Encoder {
	return &Encoder{
		ctx: ctx,
	}
}

// StartEncoding starts the encoding process for multiple files
func (e *Encoder) StartEncoding(paths []string, options EncodingOptions, progressCallback func(EncodingProgress)) error {
	if err := options.Validate(); err != nil {
		return fmt.Errorf("invalid encoding options: %w", err)
	}

	// FFmpeg 존재 여부 확인
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("FFmpeg is not installed: %w", err)
	}

	for _, inputPath := range paths {
		if err := e.encodeFile(inputPath, options, progressCallback); err != nil {
			return err
		}
	}

	return nil
}

// encodeFile handles the encoding of a single file
func (e *Encoder) encodeFile(inputPath string, options EncodingOptions, progressCallback func(EncodingProgress)) error {
	// 입력 파일 존재 여부 확인
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("input file not found (%s): %w", inputPath, err)
	}

	filename := filepath.Base(inputPath)
	outputPath := options.getOutputPath(inputPath)

	// 출력 디렉토리 생성
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory (%s): %w", outputDir, err)
	}

	// 출력 파일 중복 확인
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("output file already exists: %s", outputPath)
	}

	// 초기 진행상황 알림
	progressCallback(EncodingProgress{
		Filename: filename,
		Status:   "processing",
	})

	if options.Use2Pass && options.QualityMode == QualityModeBitrate {
		if err := e.runTwoPassEncoding(inputPath, outputPath, options, progressCallback); err != nil {
			return err
		}
	} else {
		if err := e.runSinglePassEncoding(inputPath, outputPath, options, progressCallback); err != nil {
			return err
		}
	}

	// 출력 파일 확인
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("encoded file not found: %s", outputPath)
	}

	// 완료 상태 업데이트
	progressCallback(EncodingProgress{
		Filename: filename,
		Status:   "completed",
	})

	return nil
}

// runSinglePassEncoding performs single pass encoding
func (e *Encoder) runSinglePassEncoding(inputPath, outputPath string, options EncodingOptions, progressCallback func(EncodingProgress)) error {
	args, err := options.BuildFFmpegArgs(inputPath)
	if err != nil {
		return fmt.Errorf("failed to build FFmpeg arguments: %w", err)
	}
	args = append(args, outputPath)

	return e.runFFmpegCommand(args, filepath.Base(inputPath), progressCallback)
}

// runTwoPassEncoding performs two pass encoding
func (e *Encoder) runTwoPassEncoding(inputPath, outputPath string, options EncodingOptions, progressCallback func(EncodingProgress)) error {
	passLogFile := filepath.Join(os.TempDir(), fmt.Sprintf("ffmpeg2pass_%d", time.Now().UnixNano()))
	pass1Args, pass2Args := options.Build2PassArgs(inputPath, passLogFile)

	// First pass
	if err := e.runFFmpegCommand(pass1Args, filepath.Base(inputPath), progressCallback); err != nil {
		return fmt.Errorf("first pass failed: %w", err)
	}

	// Second pass
	pass2Args = append(pass2Args, outputPath)
	if err := e.runFFmpegCommand(pass2Args, filepath.Base(inputPath), progressCallback); err != nil {
		return fmt.Errorf("second pass failed: %w", err)
	}

	// Cleanup temporary files
	os.Remove(passLogFile + "-0.log")
	os.Remove(passLogFile + "-0.log.mbtree")

	return nil
}

// runFFmpegCommand executes the FFmpeg command with progress monitoring
func (e *Encoder) runFFmpegCommand(args []string, filename string, progressCallback func(EncodingProgress)) error {
	cmd := exec.Command("ffmpeg", args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	progressReader := NewProgressReader(progressCallback, filename)
	go progressReader.ReadProgress(stderr)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start encoding: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("encoding failed: %w\nError output:\n%s", err, string(exitErr.Stderr))
		}
		return fmt.Errorf("encoding failed: %w", err)
	}

	return nil
}
