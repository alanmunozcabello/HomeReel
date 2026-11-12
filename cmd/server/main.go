package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"homereel/internal/config"
	"homereel/internal/console"
	"homereel/internal/handler"
	"homereel/internal/repository/sqlite"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/streamer.db"
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatalf("No se pudo crear el directorio de datos: %v", err)
	}

	// 1. Inicializar conexión a SQLite (con configuración WAL)
	db, err := sqlite.NewSQLiteDB(dbPath)
	if err != nil {
		log.Fatalf("Error conectando a SQLite: %v", err)
	}
	defer db.Close()

	// 2. Ejecutar esquema inicial (crear tablas si no existen)
	initSQL, err := os.ReadFile("migrations/000001_init_schema.sql")
	if err == nil {
		if _, err := db.Exec(string(initSQL)); err != nil {
			log.Fatalf("Error ejecutando migración: %v", err)
		}
	} else {
		log.Println("Advertencia: No se encontró archivo de migración en migrations/000001_init_schema.sql")
	}
	_, _ = db.Exec(`ALTER TABLE media_files ADD COLUMN modified_at_ns INTEGER NOT NULL DEFAULT 0`)

	// 3. Crear el repositorio inyectando la conexión
	mediaRepo := sqlite.NewMediaRepo(db)

	// 4. Configurar rutas inyectando el repositorio
	mediaDir := os.Getenv("MEDIA_DIR")
	if mediaDir == "" {
		mediaDir = "./media"
	}
	settingsPath := filepath.Join(filepath.Dir(dbPath), "settings.json")
	if saved, loadErr := config.Load(settingsPath); loadErr != nil {
		log.Printf("Advertencia leyendo ajustes: %v", loadErr)
	} else if os.Getenv("MEDIA_DIR") == "" && saved.MediaDir != "" {
		mediaDir = saved.MediaDir
	}
	mediaDir, err = filepath.Abs(mediaDir)
	if err != nil {
		log.Fatalf("Directorio de medios inválido: %v", err)
	}
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		log.Fatalf("No se pudo crear MEDIA_DIR: %v", err)
	}
	mux := handler.SetupRoutes(mediaRepo, mediaDir, settingsPath)

	// 5. Servir el frontend (index.html)
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fs)

	// Cambiamos localhost por 0.0.0.0 para compatibilidad absoluta en Docker
	fmt.Printf("Iniciando HomeReel en http://0.0.0.0:%s\n", port)
	console.Listen(port)

	// Forzar escucha en todas las interfaces de red
	if err := http.ListenAndServe("0.0.0.0:"+port, mux); err != nil {
		log.Fatalf("Error iniciando el servidor: %v", err)
	}
}
