// internal/scanner/scanner.go
package scanner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// File is the small unit exchanged between the filesystem scan and persistence.
type File struct {
	Path, Title        string
	Size, ModifiedAtNS int64
}

// Metadata is the result of one ffprobe invocation.
type Metadata struct {
	File
	Duration float64
	Codec    string
	Err      error
}

// ProbeFiles bounds concurrent ffprobe processes. Two workers keep a Raspberry
// Pi 3 responsive while still avoiding a long serial scan of a large library.
func ProbeFiles(ctx context.Context, files []File, workers int) []Metadata {
	if workers < 1 {
		workers = 1
	}
	if workers > len(files) {
		workers = len(files)
	}
	if workers == 0 {
		return []Metadata{}
	}
	jobs := make(chan File)
	results := make(chan Metadata, len(files))
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range jobs {
				duration, codec, err := GetMediaInfo(ctx, file.Path)
				results <- Metadata{File: file, Duration: duration, Codec: codec, Err: err}
			}
		}()
	}
	go func() {
		for _, file := range files {
			jobs <- file
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	output := make([]Metadata, 0, len(files))
	for result := range results {
		output = append(output, result)
	}
	return output
}

// Walk finds locally playable files. It performs no expensive metadata work.
func Walk(ctx context.Context, root string, visit func(File) error) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".mp4" && ext != ".mkv" && ext != ".webm" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		return visit(File{Path: path, Title: strings.TrimSuffix(filepath.Base(path), ext), Size: info.Size(), ModifiedAtNS: info.ModTime().UnixNano()})
	})
}
