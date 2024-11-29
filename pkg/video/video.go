// pkg/video/video.go
package video

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type VideoMetadata struct {
	Name     string  `json:"name"`
	Size     int64   `json:"size"`
	Duration float64 `json:"duration"`
	Format   string  `json:"format"`
	Codec    string  `json:"codec"`
	Path     string  `json:"path"`
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

// IsVideoFile checks if the file is a video file
func IsVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return supportedExtensions[ext]
}

// FindVideoFiles recursively finds video files in the given path
func FindVideoFiles(path string) ([]string, error) {
	var videos []string

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file info: %v", err)
	}

	if !fileInfo.IsDir() {
		if IsVideoFile(path) {
			return []string{path}, nil
		}
		return nil, nil
	}

	err = filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() && IsVideoFile(currentPath) {
			videos = append(videos, currentPath)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error while walking directory: %v", err)
	}

	return videos, nil
}

// ProcessVideo processes a single video file and returns its metadata
func ProcessVideo(filePath string) (*VideoMetadata, error) {
	fileName := filepath.Base(filePath)

	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe execution failed: %v", err)
	}

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
		return nil, fmt.Errorf("JSON parsing failed: %v", err)
	}

	duration, _ := strconv.ParseFloat(probe.Format.Duration, 64)
	size, _ := strconv.ParseInt(probe.Format.Size, 10, 64)

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
		Format:   strings.Split(probe.Format.Format, ",")[0],
		Codec:    videoCodec,
		Path:     filePath,
	}, nil
}

// ProcessPaths processes multiple paths and returns video metadata for each video file
func ProcessPaths(paths []string) ([]*VideoMetadata, error) {
	var results []*VideoMetadata
	var errors []string

	for _, path := range paths {
		videoPaths, err := FindVideoFiles(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error processing path (%s): %v", path, err))
			continue
		}

		for _, videoPath := range videoPaths {
			metadata, err := ProcessVideo(videoPath)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error processing video (%s): %v", videoPath, err))
				continue
			}
			results = append(results, metadata)
		}
	}

	if len(errors) > 0 {
		return results, fmt.Errorf("errors occurred while processing files:\n%s", strings.Join(errors, "\n"))
	}

	return results, nil
}
