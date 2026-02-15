package store

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

-- Prevent duplicate active jobs for same source
CREATE UNIQUE INDEX IF NOT EXISTS idx_jobs_active_source ON jobs(source_id, type) 
WHERE status IN ('queued', 'resolving_tracks', 'downloading');

CREATE TABLE IF NOT EXISTS downloads (
	provider_id TEXT PRIMARY KEY,
	title TEXT,
	artist TEXT,
	album TEXT,
	file_path TEXT NOT NULL,
	file_extension TEXT DEFAULT '.flac',
	completed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cache (
	key TEXT PRIMARY KEY,
	data BLOB,
	expires_at DATETIME
);

CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER PRIMARY KEY,
	description TEXT NOT NULL,
	applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`
