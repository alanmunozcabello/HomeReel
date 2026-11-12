// internal/streaming/vod.go
package streaming

import (
	"net/http"
	"os"
)

// ServeVOD utiliza http.ServeContent para manejar "206 Partial Content" nativamente
func ServeVOD(w http.ResponseWriter, r *http.Request, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Archivo no disponible", http.StatusNotFound)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Error de disco", http.StatusInternalServerError)
		return
	}

	// http.ServeContent soporta los headers Range que Video.js necesita para saltar (seek)
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), file)
}