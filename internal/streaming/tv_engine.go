// internal/streaming/tv_engine.go
package streaming

import (
	"time"
)

type PlaylistItem struct {
	ID       int
	FilePath string
	Duration int64 // en segundos
}

// CalculateLiveBroadcast determina qué video está "al aire" y en qué segundo exacto
func CalculateLiveBroadcast(playlist []PlaylistItem) (currentVideoID int, seekToSeconds int64) {
	if len(playlist) == 0 {
		return 0, 0
	}

	var totalDuration int64 = 0
	for _, item := range playlist {
		if item.Duration > 0 {
			totalDuration += item.Duration
		}
	}
	if totalDuration == 0 {
		return 0, 0
	}

	// Usar el reloj del sistema como fuente de verdad.
	// El módulo del tiempo actual contra la duración total crea un loop infinito sincronizado.
	now := time.Now().Unix()
	currentLoopPosition := now % totalDuration

	var accumulated int64 = 0
	for _, item := range playlist {
		if item.Duration <= 0 {
			continue
		}
		if currentLoopPosition < accumulated+item.Duration {
			// Encontramos el video actual
			offset := currentLoopPosition - accumulated
			return item.ID, offset
		}
		accumulated += item.Duration
	}

	return 0, 0
}
