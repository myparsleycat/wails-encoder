// pkg/app/app.go
package app

import (
	"context"
	"fmt"
	"runtime"

	"encoder/pkg/codec"
	"encoder/pkg/encoder"
	"encoder/pkg/video"

	wails_runtime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx     context.Context
	encoder *encoder.Encoder
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// Startup is called at application startup
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.encoder = encoder.NewEncoder(ctx)
}

// DomReady is called after front-end resources have been loaded
func (a App) DomReady(ctx context.Context) {
}

// BeforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	return false
}

// Shutdown is called at application termination
func (a *App) Shutdown(ctx context.Context) {
}

// EmitProgress sends encoding progress updates to the frontend
func (a *App) EmitProgress(progress encoder.EncodingProgress) {
	wails_runtime.EventsEmit(a.ctx, "encoding_progress", progress)
}

// Public methods that will be called from frontend

func (a *App) ProcessVideoPaths(paths []string) ([]*video.VideoMetadata, error) {
	return video.ProcessPaths(paths)
}

func (a *App) GetAvailableCodecs() ([]codec.CodecInfo, error) {
	return codec.GetAvailable()
}

func (a *App) StartEncodingWithOptions(paths []string, options encoder.EncodingOptions) error {
	return a.encoder.StartEncoding(paths, options, a.EmitProgress)
}

func (a *App) ShowNotification(title, message string) error {
	switch runtime.GOOS {
	case "darwin":
		return showMacNotification(title, message)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
