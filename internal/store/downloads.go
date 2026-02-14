package store

import (
	"database/sql"

	"github.com/cesargomez89/navidrums/internal/domain"
)

func (db *DB) CreateDownload(download *domain.Download) error {
	query := `INSERT OR REPLACE INTO downloads (provider_id, file_path, file_extension, completed_at) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, download.ProviderID, download.FilePath, download.FileExtension, download.CompletedAt)
	return err
}

func (db *DB) GetDownload(providerID string) (*domain.Download, error) {
	query := `SELECT provider_id, file_path, file_extension, completed_at FROM downloads WHERE provider_id = ?`
	row := db.QueryRow(query, providerID)

	download := &domain.Download{}
	var ext sql.NullString
	err := row.Scan(&download.ProviderID, &download.FilePath, &ext, &download.CompletedAt)
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, err
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
	query := `SELECT provider_id, file_path, file_extension, completed_at FROM downloads ORDER BY completed_at DESC LIMIT ?`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var downloads []*domain.Download
	for rows.Next() {
		d := &domain.Download{}
		var ext sql.NullString
		err := rows.Scan(&d.ProviderID, &d.FilePath, &ext, &d.CompletedAt)
		if err != nil {
			return nil, err
		}
		if ext.Valid {
			d.FileExtension = ext.String
		} else {
			d.FileExtension = ".flac" // Default for backward compatibility
		}
		downloads = append(downloads, d)
	}
	return downloads, nil
}
