-- Habilitar el modo WAL para permitir lecturas y escrituras simultáneas
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 5000;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS libraries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    path TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS media_files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL,
    file_path TEXT NOT NULL UNIQUE,
    title TEXT,
    duration REAL DEFAULT 0,
    video_codec TEXT,
    size_bytes INTEGER NOT NULL,
    modified_at_ns INTEGER NOT NULL DEFAULT 0,
    is_available BOOLEAN DEFAULT 1,
    last_scanned_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(library_id) REFERENCES libraries(id) ON DELETE CASCADE
);

-- Índices de alto rendimiento para el escáner y la API
CREATE INDEX IF NOT EXISTS idx_media_path ON media_files(file_path);
CREATE INDEX IF NOT EXISTS idx_media_available ON media_files(is_available);
