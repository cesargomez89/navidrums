package repository

const Schema = `
CREATE TABLE IF NOT EXISTS jobs (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	status TEXT NOT NULL,
	title TEXT,
	artist TEXT,
	progress REAL DEFAULT 0,
	source_id TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	error TEXT
);

CREATE TABLE IF NOT EXISTS downloads (
	provider_id TEXT PRIMARY KEY,
	file_path TEXT NOT NULL,
	completed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cache (
	key TEXT PRIMARY KEY,
	data BLOB,
	expires_at DATETIME
);

DROP TABLE IF EXISTS job_items;
`
