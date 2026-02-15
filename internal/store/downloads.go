package store

import (
	"database/sql"

	"github.com/cesargomez89/navidrums/internal/domain"
)

func (db *DB) CreateDownload(download *domain.Download) error {
	query := `INSERT OR REPLACE INTO downloads (provider_id, title, artist, album, file_path, file_extension, completed_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(query, download.ProviderID, download.Title, download.Artist, download.Album, download.FilePath, download.FileExtension, download.CompletedAt)
	return err
}

func (db *DB) GetDownload(providerID string) (*domain.Download, error) {
	query := `SELECT provider_id, title, artist, album, file_path, file_extension, completed_at FROM downloads WHERE provider_id = ?`
	row := db.QueryRow(query, providerID)

	download := &domain.Download{}
	var ext, title, artist, album sql.NullString
	err := row.Scan(&download.ProviderID, &title, &artist, &album, &download.FilePath, &ext, &download.CompletedAt)
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, err
	}
	if title.Valid {
		download.Title = title.String
	}
	if artist.Valid {
		download.Artist = artist.String
	}
	if album.Valid {
		download.Album = album.String
	}
	if ext.Valid {
		download.FileExtension = ext.String
	} else {
		download.FileExtension = ".flac" // Default for backward compatibility
	}
	return download, nil
}

// Stats or history?
func (db *DB) ListDownloads(limit int) ([]*domain.Download, error) {
	query := `SELECT provider_id, title, artist, album, file_path, file_extension, completed_at FROM downloads ORDER BY completed_at DESC LIMIT ?`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var downloads []*domain.Download
	for rows.Next() {
		d, err := scanDownload(rows)
		if err != nil {
			return nil, err
		}
		downloads = append(downloads, d)
	}
	return downloads, nil
}

func (db *DB) DeleteDownload(providerID string) error {
	_, err := db.Exec("DELETE FROM downloads WHERE provider_id = ?", providerID)
	return err
}

func scanDownload(row scanner) (*domain.Download, error) {
	d := &domain.Download{}
	var ext, title, artist, album sql.NullString
	err := row.Scan(&d.ProviderID, &title, &artist, &album, &d.FilePath, &ext, &d.CompletedAt)
	if err != nil {
		return nil, err
	}
	if title.Valid {
		d.Title = title.String
	}
	if artist.Valid {
		d.Artist = artist.String
	}
	if album.Valid {
		d.Album = album.String
	}
	if ext.Valid {
		d.FileExtension = ext.String
	} else {
		d.FileExtension = ".flac"
	}
	return d, nil
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func (db *DB) SearchDownloads(query string, limit int) ([]*domain.Download, error) {
	sqlQuery := `SELECT provider_id, title, artist, album, file_path, file_extension, completed_at 
		FROM downloads 
		WHERE title LIKE ? OR artist LIKE ? OR album LIKE ?
		ORDER BY completed_at DESC LIMIT ?`
	searchTerm := "%" + query + "%"
	rows, err := db.Query(sqlQuery, searchTerm, searchTerm, searchTerm, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var downloads []*domain.Download
	for rows.Next() {
		d, err := scanDownload(rows)
		if err != nil {
			return nil, err
		}
		downloads = append(downloads, d)
	}
	return downloads, nil
}
