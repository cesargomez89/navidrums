package store

// ProviderRecord represents a music provider stored in the database for fallback support.
type ProviderRecord struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	Name     string `json:"name"`
}

type ProvidersRepo struct {
	db *DB
}

func NewProvidersRepo(db *DB) *ProvidersRepo {
	return &ProvidersRepo{db: db}
}

func (r *ProvidersRepo) Create(providerType, url, name string) (int64, error) {
	var id int64
	err := r.db.RunInTx(func(txDB *DB) error {
		var maxPos int
		err := txDB.QueryRow(`SELECT COALESCE(MAX(position), -1) FROM providers WHERE type = ?`, providerType).Scan(&maxPos)
		if err != nil {
			return err
		}
		query := `INSERT INTO providers (type, url, name, position) VALUES (?, ?, ?, ?) RETURNING id`
		row := txDB.QueryRowx(query, providerType, url, name, maxPos+1)
		return row.Scan(&id)
	})
	return id, err
}

func (r *ProvidersRepo) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM providers WHERE id = ?`, id)
	return err
}

func (r *ProvidersRepo) ListByType(providerType string) ([]ProviderRecord, error) {
	var providers []ProviderRecord
	query := `SELECT id, type, url, name, position FROM providers WHERE type = ? ORDER BY position ASC`
	err := r.db.Select(&providers, query, providerType)
	return providers, err
}

func (r *ProvidersRepo) GetByPosition(providerType string, pos int) (*ProviderRecord, error) {
	query := `SELECT id, type, url, name, position FROM providers WHERE type = ? AND position = ?`
	var provider ProviderRecord
	err := r.db.Get(&provider, query, providerType, pos)
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

func (r *ProvidersRepo) Reorder(ids []int64) error {
	return r.db.RunInTx(func(txDB *DB) error {
		if len(ids) == 0 {
			return nil
		}
		_, err := txDB.Exec(`
			UPDATE providers SET position = position + 1000
			WHERE type = (SELECT type FROM providers WHERE id = ?)`, ids[0])
		if err != nil {
			return err
		}
		for i, id := range ids {
			_, err := txDB.Exec(`UPDATE providers SET position = ? WHERE id = ?`, i, id)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ProvidersRepo) Update(id int64, url, name string) error {
	query := `UPDATE providers SET url = ?, name = ? WHERE id = ?`
	result, err := r.db.Exec(query, url, name, id)
	if err != nil {
		return err
	}
	return checkRowsAffected(result, "provider", id)
}

func (r *ProvidersRepo) Exists(url string) bool {
	var count int
	query := `SELECT COUNT(*) FROM providers WHERE url = ?`
	_ = r.db.Get(&count, query, url)
	return count > 0
}
