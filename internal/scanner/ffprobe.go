// internal/scanner/ffprobe.go
package scanner

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
)

// ffprobeOutput mapea la estructura JSON que devuelve ffprobe
type ffprobeOutput struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
	Streams []struct {
		CodecName string `json:"codec_name"`
		CodecType string `json:"codec_type"`
	} `json:"streams"`
}

// GetMediaInfo ejecuta ffprobe y devuelve los metadatos esenciales
func GetMediaInfo(ctx context.Context, filePath string) (duration float64, videoCodec string, err error) {
	// -show_format: extrae duración y tamaño. -show_streams: extrae códecs
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return 0, "", err
	}

	var data ffprobeOutput
	if err := json.Unmarshal(out.Bytes(), &data); err != nil {
		return 0, "", err
	}

	// Parsear duración (viene como string, ej: "125.433000")
	duration, _ = strconv.ParseFloat(data.Format.Duration, 64)

	// Buscar el códec de video
	for _, stream := range data.Streams {
		if stream.CodecType == "video" {
			videoCodec = stream.CodecName
			break
		}
	}

	return duration, videoCodec, nil
}