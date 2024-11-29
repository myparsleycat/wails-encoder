// pkg/encoder/options.go
package encoder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"encoder/pkg/codec"
)

type QualityMode string

const (
	QualityModeCRF     QualityMode = "crf"
	QualityModeBitrate QualityMode = "bitrate"
)

type EncodingOptions struct {
	VideoFormat  string      `json:"videoformat"`
	VideoCodec   string      `json:"videocodec"`
	QualityMode  QualityMode `json:"qualitymode"`
	QualityValue int         `json:"qualityvalue"`
	Use2Pass     bool        `json:"use2pass"`

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
			default_: 23,
		},
	},
	"hevc": {
		defaultMode: QualityModeCRF,
		qualityRange: struct{ min, max, default_ int }{
			min:      0,
			max:      51,
			default_: 28,
		},
	},
	"vp9": {
		defaultMode: QualityModeCRF,
		qualityRange: struct{ min, max, default_ int }{
			min:      0,
			max:      63,
			default_: 31,
		},
	},
}

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
	// Format validation
	supportedCodecs, exists := codec.SupportedFormats[opts.VideoFormat]
	if !exists {
		return fmt.Errorf("unsupported video format: %s", opts.VideoFormat)
	}

	// Codec validation
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

	// Quality settings validation
	baseCodec := strings.Split(opts.VideoCodec, "_")[0]
	codecSet, exists := codecSettings[baseCodec]
	if exists {
		if opts.QualityValue == 0 {
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
	}

	if opts.Use2Pass && opts.QualityMode != QualityModeBitrate {
		return fmt.Errorf("2-pass encoding is only available with bitrate mode")
	}

	return nil
}

func (opts *EncodingOptions) BuildFFmpegArgs(inputPath string) ([]string, error) {
	args := []string{"-i", inputPath}

	// Video codec
	args = append(args, "-c:v", opts.VideoCodec)

	// Quality settings
	switch opts.QualityMode {
	case QualityModeCRF:
		args = append(args, "-crf", fmt.Sprintf("%d", opts.QualityValue))
	case QualityModeBitrate:
		args = append(args, "-b:v", fmt.Sprintf("%dk", opts.QualityValue))
	}

	// Resize settings
	if opts.IsResize && opts.Width > 0 && opts.Height > 0 {
		args = append(args, "-vf", fmt.Sprintf("scale=%d:%d", opts.Width, opts.Height))
	}

	// Audio settings
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
	// First pass arguments
	pass1Args := []string{
		"-i", inputPath,
		"-c:v", opts.VideoCodec,
		"-b:v", fmt.Sprintf("%dk", opts.QualityValue),
		"-pass", "1",
		"-passlogfile", passLogFile,
		"-an",
		"-f", "null",
	}
	if opts.IsResize && opts.Width > 0 && opts.Height > 0 {
		pass1Args = append(pass1Args, "-vf", fmt.Sprintf("scale=%d:%d", opts.Width, opts.Height))
	}
	pass1Args = append(pass1Args, os.DevNull)

	// Second pass arguments
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

	// Audio settings for second pass
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
