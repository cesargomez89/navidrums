package repository

import (
	"database/sql"

	"github.com/cesargomez89/navidrums/internal/models"
)

func (db *DB) CreateDownload(download *models.Download) error {
	query := `INSERT OR REPLACE INTO downloads (provider_id, file_path, completed_at) VALUES (?, ?, ?)`
	_, err := db.Exec(query, download.ProviderID, download.FilePath, download.CompletedAt)
	return err
}

func (db *DB) GetDownload(providerID string) (*models.Download, error) {
	query := `SELECT provider_id, file_path, completed_at FROM downloads WHERE provider_id = ?`
	row := db.QueryRow(query, providerID)

	download := &models.Download{}
	err := row.Scan(&download.ProviderID, &download.FilePath, &download.CompletedAt)
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, err
	}
	return download, nil
}

// Stats or history?
func (db *DB) ListDownloads(limit int) ([]*models.Download, error) {
	query := `SELECT provider_id, file_path, completed_at FROM downloads ORDER BY completed_at DESC LIMIT ?`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var downloads []*models.Download
	for rows.Next() {
		d := &models.Download{}
		err := rows.Scan(&d.ProviderID, &d.FilePath, &d.CompletedAt)
		if err != nil {
			return nil, err
		}
		downloads = append(downloads, d)
	}
	return downloads, nil
}
