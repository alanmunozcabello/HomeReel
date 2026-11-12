package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// NewSQLiteDB inicializa la base de datos con la configuración óptima para RPi3
func NewSQLiteDB(dbPath string) (*sql.DB, error) {
	// Parámetros por cadena de conexión para modernc.org/sqlite
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Límites estrictos para evitar fugas de memoria y bloqueos en SQLite
	db.SetMaxOpenConns(5)                   // Máximo 5 conexiones simultáneas
	db.SetMaxIdleConns(2)                   // Mantiene 2 en reposo para respuestas rápidas
	db.SetConnMaxLifetime(30 * time.Minute) 

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}