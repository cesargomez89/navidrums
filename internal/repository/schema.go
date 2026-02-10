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

CREATE TABLE IF NOT EXISTS job_items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	job_id TEXT NOT NULL,
	track_id TEXT NOT NULL,
	status TEXT DEFAULT 'pending',
	progress REAL DEFAULT 0,
	title TEXT,
	file_path TEXT,
	FOREIGN KEY(job_id) REFERENCES jobs(id)
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
`
