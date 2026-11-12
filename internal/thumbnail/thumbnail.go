package thumbnail

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// Generator creates a cached JPEG poster. One ffmpeg process at a time keeps
// thumbnail generation unobtrusive on a Raspberry Pi 3.
type Generator struct {
	dir       string
	semaphore chan struct{}
	mu        sync.Mutex
}

func New(dir string) *Generator { return &Generator{dir: dir, semaphore: make(chan struct{}, 1)} }

func (g *Generator) Path(id int) string { return filepath.Join(g.dir, fmt.Sprintf("%d.jpg", id)) }

func (g *Generator) Ensure(ctx context.Context, id int, videoPath string) (string, error) {
	target := g.Path(id)
	if _, err := os.Stat(target); err == nil {
		return target, nil
	}
	g.semaphore <- struct{}{}
	defer func() { <-g.semaphore }()
	// Another HTTP request may have generated it while this one waited.
	if _, err := os.Stat(target); err == nil {
		return target, nil
	}
	if err := os.MkdirAll(g.dir, 0755); err != nil {
		return "", err
	}
	temporary := target + ".tmp.jpg"
	defer os.Remove(temporary)
	cmd := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-ss", "10", "-i", videoPath, "-frames:v", "1", "-vf", "scale=320:-2", "-q:v", "5", "-y", temporary)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg: %w: %s", err, out)
	}
	if err := os.Rename(temporary, target); err != nil {
		return "", err
	}
	return target, nil
}
