package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"homereel/internal/config"
	"homereel/internal/repository/sqlite"
	"homereel/internal/scanner"
	"homereel/internal/streaming"
	"homereel/internal/thumbnail"
)

// SetupRoutes ahora recibe el repositorio como dependencia
func SetupRoutes(repo *sqlite.MediaRepo, mediaDir, settingsPath string) *http.ServeMux {
	mux := http.NewServeMux()
	controller := &streaming.LiveController{}
	thumbnails := thumbnail.New(filepath.Join("data", "thumbnails"))
	library := &mediaRoot{dir: mediaDir, settingsPath: settingsPath}

	// Endpoint 1: Streaming VOD usando el ID de la base de datos
	mux.HandleFunc("GET /api/v1/stream/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id") // Función de Go 1.22
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "ID de video inválido", http.StatusBadRequest)
			return
		}

		// Buscar la ruta física en SQLite
		media, err := repo.GetMediaByID(r.Context(), id)
		if err != nil || media == nil {
			http.Error(w, "Archivo no encontrado en la base de datos", http.StatusNotFound)
			return
		}

		// Pasar la ruta real al motor nativo
		streaming.ServeVOD(w, r, media.FilePath)
	})

	// Posters are generated once and then served from the local cache.
	mux.HandleFunc("GET /api/v1/thumbnail/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil || id < 1 {
			http.Error(w, "ID de video invalido", http.StatusBadRequest)
			return
		}
		media, err := repo.GetMediaByID(r.Context(), id)
		if err != nil || media == nil {
			http.NotFound(w, r)
			return
		}
		poster, err := thumbnails.Ensure(r.Context(), id, media.FilePath)
		if err != nil {
			http.Error(w, "No se pudo crear miniatura", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=86400")
		http.ServeFile(w, r, poster)
	})

	mux.HandleFunc("GET /api/v1/library", func(w http.ResponseWriter, r *http.Request) {
		items, err := repo.GetAvailableMediaForPath(r.Context(), library.get())
		if err != nil {
			http.Error(w, "Error consultando catálogo", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(presentLibrary(items, library.get()))
	})

	mux.HandleFunc("GET /api/v1/settings/media", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"media_dir": library.get()})
	})
	mux.HandleFunc("PUT /api/v1/settings/media", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			MediaDir string `json:"media_dir"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "JSON invalido", 400)
			return
		}
		if err := library.set(request.MediaDir); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"media_dir": library.get()})
	})
	mux.HandleFunc("POST /api/v1/settings/media/pick", func(w http.ResponseWriter, r *http.Request) {
		dir, err := pickMediaFolder(r.Context())
		if err != nil {
			http.Error(w, "No se pudo abrir el selector: "+err.Error(), 500)
			return
		}
		if dir == "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err := library.set(dir); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"media_dir": library.get()})
	})

	// Endpoint 2: Canal de TV sincronizado con el catálogo real
	mux.HandleFunc("GET /api/v1/tv/live", func(w http.ResponseWriter, r *http.Request) {
		// Obtener todos los videos disponibles
		mediaList, err := repo.GetAvailableMediaForPath(r.Context(), library.get())
		if err != nil {
			http.Error(w, "Error consultando catálogo", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		// Manejo seguro si la base de datos está vacía
		if len(mediaList) == 0 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"now_playing_id": 0,
				"seek_seconds":   0,
				"stream_url":     "",
				"message":        "No hay videos escaneados en el catálogo",
			})
			return
		}

		// Convertir el modelo de BD al modelo de lista de reproducción del motor TV
		var playlist []streaming.PlaylistItem
		for _, m := range mediaList {
			playlist = append(playlist, streaming.PlaylistItem{
				ID:       m.ID,
				FilePath: m.FilePath,
				Duration: int64(m.Duration),
			})
		}

		videoID, offset, paused := controller.Position(playlist)

		response := map[string]interface{}{
			"now_playing_id": videoID,
			"seek_seconds":   offset,
			"stream_url":     "/api/v1/stream/" + strconv.Itoa(videoID),
			"paused":         paused,
		}

		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("POST /api/v1/tv/control", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Action      string `json:"action"`
			MediaID     int    `json:"media_id"`
			SeekSeconds int64  `json:"seek_seconds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "JSON invalido", 400)
			return
		}
		items, err := repo.GetAvailableMediaForPath(r.Context(), library.get())
		if err != nil {
			http.Error(w, "Error consultando catalogo", 500)
			return
		}
		playlist := make([]streaming.PlaylistItem, 0, len(items))
		for _, m := range items {
			playlist = append(playlist, streaming.PlaylistItem{ID: m.ID, Duration: int64(m.Duration)})
		}
		if err := controller.Control(playlist, request.Action, request.MediaID, request.SeekSeconds); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})

	// Escaneo diferencial: ffprobe sólo se ejecuta para archivos nuevos o modificados.
	mux.HandleFunc("POST /api/v1/library/scan", func(w http.ResponseWriter, r *http.Request) {
		mediaDir := library.get()
		if err := os.MkdirAll(mediaDir, 0755); err != nil {
			http.Error(w, "No se pudo preparar la biblioteca", 500)
			return
		}
		libraryID, err := repo.EnsureLibrary(r.Context(), "Media local", mediaDir)
		if err != nil {
			http.Error(w, "No se pudo registrar la biblioteca", 500)
			return
		}
		known, err := repo.GetLibraryMedia(r.Context(), libraryID)
		if err != nil {
			http.Error(w, "No se pudo leer el catálogo", 500)
			return
		}
		seen, added, skipped, failed := map[string]bool{}, 0, 0, 0
		changed := make([]scanner.File, 0)
		err = scanner.Walk(r.Context(), mediaDir, func(file scanner.File) error {
			seen[file.Path] = true
			if old, ok := known[file.Path]; ok && old.SizeBytes == file.Size && old.ModifiedAtNS == file.ModifiedAtNS {
				skipped++
				return nil
			}
			changed = append(changed, file)
			return nil
		})
		if err != nil {
			http.Error(w, "Error durante el escaneo: "+err.Error(), 500)
			return
		}
		for _, result := range scanner.ProbeFiles(r.Context(), changed, 2) {
			if result.Err != nil || result.Duration <= 0 {
				failed++
				continue
			}
			if err := repo.UpsertMedia(r.Context(), sqlite.MediaFile{LibraryID: libraryID, FilePath: result.Path, Title: result.Title, Duration: result.Duration, VideoCodec: result.Codec, SizeBytes: result.Size, ModifiedAtNS: result.ModifiedAtNS}); err != nil {
				http.Error(w, "Error guardando catálogo: "+err.Error(), 500)
				return
			}
			added++
		}
		for path := range known {
			if !seen[path] {
				_ = repo.MarkAsUnavailable(r.Context(), path)
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]int{"indexed": added, "unchanged": skipped, "failed": failed})
	})

	return mux
}

type mediaRoot struct {
	mu                sync.RWMutex
	dir, settingsPath string
}

func (m *mediaRoot) get() string { m.mu.RLock(); defer m.mu.RUnlock(); return m.dir }
func (m *mediaRoot) set(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("la ruta no puede estar vacia")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("no se puede acceder a la carpeta: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("la ruta indicada no es una carpeta")
	}
	if err := config.Save(m.settingsPath, config.Settings{MediaDir: abs}); err != nil {
		return err
	}
	m.mu.Lock()
	m.dir = abs
	m.mu.Unlock()
	return nil
}

type libraryItem struct {
	ID         int     `json:"id"`
	Title      string  `json:"title"`
	Duration   float64 `json:"duration"`
	VideoCodec string  `json:"video_codec"`
	Type       string  `json:"type"`
	Collection string  `json:"collection"`
	Season     string  `json:"season"`
}

func presentLibrary(items []sqlite.MediaFile, root string) []libraryItem {
	output := make([]libraryItem, 0, len(items))
	for _, m := range items {
		i := libraryItem{ID: m.ID, Title: m.Title, Duration: m.Duration, VideoCodec: m.VideoCodec, Type: "other", Collection: "Otros"}
		rel, err := filepath.Rel(root, m.FilePath)
		if err != nil {
			output = append(output, i)
			continue
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) >= 2 && (strings.EqualFold(parts[0], "series") || strings.EqualFold(parts[0], "sagas")) {
			i.Type = "series"
			i.Collection = parts[1]
			i.Season = "Temporada 1"
			if len(parts) >= 4 {
				i.Season = parts[2]
			}
		}
		if len(parts) >= 2 && strings.EqualFold(parts[0], "peliculas") {
			i.Type = "movie"
			i.Collection = "Peliculas"
			if len(parts) >= 3 {
				i.Collection = parts[1]
			}
		}
		output = append(output, i)
	}
	return output
}
