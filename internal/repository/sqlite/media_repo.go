package sqlite

import (
	"context"
	"database/sql"
)

// MediaFile representa el modelo de dominio interno
type MediaFile struct {
	ID           int
	LibraryID    int
	FilePath     string
	Title        string
	Duration     float64
	VideoCodec   string
	SizeBytes    int64
	ModifiedAtNS int64
	IsAvailable  bool
}

type MediaRepo struct {
	db *sql.DB
}

func NewMediaRepo(db *sql.DB) *MediaRepo {
	return &MediaRepo{db: db}
}

// UpsertMedia inserta un nuevo archivo o actualiza uno existente si la ruta ya existe
func (r *MediaRepo) UpsertMedia(ctx context.Context, media MediaFile) error {
	query := `
		INSERT INTO media_files (library_id, file_path, title, duration, video_codec, size_bytes, modified_at_ns, is_available, last_scanned_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, CURRENT_TIMESTAMP)
		ON CONFLICT(file_path) DO UPDATE SET
			title = excluded.title,
			duration = excluded.duration,
			video_codec = excluded.video_codec,
			size_bytes = excluded.size_bytes,
			modified_at_ns = excluded.modified_at_ns,
			is_available = 1,
			last_scanned_at = CURRENT_TIMESTAMP;
	`
	_, err := r.db.ExecContext(ctx, query,
		media.LibraryID, media.FilePath, media.Title,
		media.Duration, media.VideoCodec, media.SizeBytes, media.ModifiedAtNS,
	)
	return err
}

// EnsureLibrary creates (or returns) the logical library for a media directory.
func (r *MediaRepo) EnsureLibrary(ctx context.Context, name, path string) (int, error) {
	if _, err := r.db.ExecContext(ctx, `INSERT INTO libraries (name, path) VALUES (?, ?) ON CONFLICT(path) DO UPDATE SET name = excluded.name`, name, path); err != nil {
		return 0, err
	}
	var id int
	err := r.db.QueryRowContext(ctx, `SELECT id FROM libraries WHERE path = ?`, path).Scan(&id)
	return id, err
}

// GetLibraryMedia returns the fingerprints needed for a cheap differential scan.
func (r *MediaRepo) GetLibraryMedia(ctx context.Context, libraryID int) (map[string]MediaFile, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, file_path, size_bytes, modified_at_ns FROM media_files WHERE library_id = ?`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make(map[string]MediaFile)
	for rows.Next() {
		var m MediaFile
		if err := rows.Scan(&m.ID, &m.FilePath, &m.SizeBytes, &m.ModifiedAtNS); err != nil {
			return nil, err
		}
		items[m.FilePath] = m
	}
	return items, rows.Err()
}

// GetMediaByID obtiene los datos físicos de un video para el streaming VOD
func (r *MediaRepo) GetMediaByID(ctx context.Context, id int) (*MediaFile, error) {
	query := `SELECT id, library_id, file_path, title, duration, video_codec, size_bytes, is_available 
	          FROM media_files WHERE id = ? AND is_available = 1`

	row := r.db.QueryRowContext(ctx, query, id)

	var m MediaFile
	err := row.Scan(&m.ID, &m.LibraryID, &m.FilePath, &m.Title, &m.Duration, &m.VideoCodec, &m.SizeBytes, &m.IsAvailable)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No encontrado
		}
		return nil, err
	}
	return &m, nil
}

// MarkAsUnavailable marca archivos desconectados sin borrar su historial
func (r *MediaRepo) MarkAsUnavailable(ctx context.Context, filePath string) error {
	query := `UPDATE media_files SET is_available = 0 WHERE file_path = ?`
	_, err := r.db.ExecContext(ctx, query, filePath)
	return err
}

// GetAllAvailableMedia obtiene el catálogo para el frontend
func (r *MediaRepo) GetAllAvailableMedia(ctx context.Context) ([]MediaFile, error) {
	query := `SELECT id, file_path, title, duration, video_codec FROM media_files WHERE is_available = 1 ORDER BY title ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Un slice inicializado se codifica como [] (no como null), que es el
	// contrato del endpoint incluso cuando la biblioteca está vacía.
	results := make([]MediaFile, 0)
	for rows.Next() {
		var m MediaFile
		if err := rows.Scan(&m.ID, &m.FilePath, &m.Title, &m.Duration, &m.VideoCodec); err != nil {
			continue
		}
		results = append(results, m)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// GetAvailableMediaForPath isolates the active root selected by the admin.
func (r *MediaRepo) GetAvailableMediaForPath(ctx context.Context, libraryPath string) ([]MediaFile, error) {
	query := `SELECT m.id, m.file_path, m.title, m.duration, m.video_codec FROM media_files m JOIN libraries l ON l.id=m.library_id WHERE m.is_available=1 AND l.path=? ORDER BY m.title ASC`
	rows, err := r.db.QueryContext(ctx, query, libraryPath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := make([]MediaFile, 0)
	for rows.Next() {
		var m MediaFile
		if err := rows.Scan(&m.ID, &m.FilePath, &m.Title, &m.Duration, &m.VideoCodec); err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, rows.Err()
}
