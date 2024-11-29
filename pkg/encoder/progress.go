// pkg/encoder/progress.go
package encoder

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type EncodingProgress struct {
	Filename string  `json:"filename"`
	Frame    int     `json:"frame"`
	FPS      int     `json:"fps"`
	Time     string  `json:"time"`
	Size     int     `json:"size"`
	Bitrate  float64 `json:"bitrate"`
	Speed    float64 `json:"speed"`
	Progress float64 `json:"progress"`
	Status   string  `json:"status"`
}

var (
	frameRegex   = regexp.MustCompile(`frame=\s*(\d+)`)
	fpsRegex     = regexp.MustCompile(`fps=\s*(\d+)`)
	timeRegex    = regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2}\.\d{2})`)
	sizeRegex    = regexp.MustCompile(`size=\s*(\d+)kB`)
	bitrateRegex = regexp.MustCompile(`bitrate=\s*(\d+\.\d+)kbits/s`)
	speedRegex   = regexp.MustCompile(`speed=\s*(\d+\.\d+)x`)
)

type ProgressReader struct {
	callback     func(EncodingProgress)
	filename     string
	scanner      *bufio.Scanner
	lastProgress EncodingProgress
}

func NewProgressReader(callback func(EncodingProgress), filename string) *ProgressReader {
	return &ProgressReader{
		callback: callback,
		filename: filename,
	}
}

func (pr *ProgressReader) ReadProgress(reader io.Reader) {
	pr.scanner = bufio.NewScanner(reader)
	pr.scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
			return i + 1, data[0:i], nil
		}

		if atEOF {
			return len(data), data, nil
		}

		return 0, nil, nil
	})

	for pr.scanner.Scan() {
		line := pr.scanner.Text()
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		progress := EncodingProgress{
			Filename: pr.filename,
			Status:   "processing",
		}

		// Parse progress information
		if matches := frameRegex.FindStringSubmatch(line); len(matches) > 1 {
			progress.Frame, _ = strconv.Atoi(matches[1])
		}

		if matches := fpsRegex.FindStringSubmatch(line); len(matches) > 1 {
			progress.FPS, _ = strconv.Atoi(matches[1])
		}

		if matches := timeRegex.FindStringSubmatch(line); len(matches) > 1 {
			progress.Time = matches[0][5:] // Remove "time=" prefix
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

		// Only emit progress event if we have meaningful progress data
		if progress.Time != "" || progress.Frame > 0 {
			pr.callback(progress)
			pr.lastProgress = progress
		}
	}
}
