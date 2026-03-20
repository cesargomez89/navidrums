package store

// Provider represents a music provider for fallback support.
type Provider struct {
	ID       int64
	Position int
	URL      string
	Name     string
}

type ProvidersRepo struct {
	db *DB
}

func NewProvidersRepo(db *DB) *ProvidersRepo {
	return &ProvidersRepo{db: db}
}

func (r *ProvidersRepo) Create(url, name string) int64 {
	var id int64
	err := r.db.RunInTx(func(txDB *DB) error {
		var maxPos int
		err := txDB.QueryRow(`SELECT COALESCE(MAX(position), -1) FROM providers`).Scan(&maxPos)
		if err != nil {
			return err
		}
		query := `INSERT INTO providers (url, name, position) VALUES (?, ?, ?) RETURNING id`
		row := txDB.QueryRowx(query, url, name, maxPos+1)
		return row.Scan(&id)
	})
	if err != nil {
		return 0
	}
	return id
}

func (r *ProvidersRepo) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM providers WHERE id = ?`, id)
	return err
}

func (r *ProvidersRepo) ListOrdered() ([]Provider, error) {
	var providers []Provider
	query := `SELECT id, url, name, position FROM providers ORDER BY position ASC`
	err := r.db.Select(&providers, query)
	return providers, err
}

func (r *ProvidersRepo) GetByPosition(pos int) (*Provider, error) {
	query := `SELECT id, url, name, position FROM providers WHERE position = ?`
	var provider Provider
	err := r.db.Get(&provider, query, pos)
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
		_, err := txDB.Exec(`UPDATE providers SET position = position + 1000`)
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
