package store

const Schema = `
CREATE TABLE IF NOT EXISTS jobs (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	status TEXT NOT NULL,
	progress REAL DEFAULT 0,
	source_id TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	error TEXT
);

-- Prevent duplicate active jobs for same source
CREATE UNIQUE INDEX IF NOT EXISTS idx_jobs_active_source ON jobs(source_id, type) 
WHERE status IN ('queued', 'running');

CREATE TABLE IF NOT EXISTS tracks (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	provider_id TEXT UNIQUE NOT NULL,
	
	-- Metadata
	title TEXT NOT NULL,
	artist TEXT,
	artists TEXT,  -- JSON array
	album TEXT,
	album_artist TEXT,
	album_artists TEXT,  -- JSON array
	track_number INTEGER,
	disc_number INTEGER,
	total_tracks INTEGER,
	total_discs INTEGER,
	year INTEGER,
	genre TEXT,
	label TEXT,
	isrc TEXT,
	copyright TEXT,
	composer TEXT,
	duration INTEGER,
	explicit BOOLEAN,
	compilation BOOLEAN,
	album_art_url TEXT,
	lyrics TEXT,
	subtitles TEXT,
	
	-- Extended metadata
	bpm INTEGER,
	key_name TEXT,
	key_scale TEXT,
	replay_gain REAL,
	peak REAL,
	version TEXT,
	description TEXT,
	url TEXT,
	audio_quality TEXT,
	audio_modes TEXT,
	release_date TEXT,
	
	-- Processing
	status TEXT NOT NULL,
	error TEXT,
	parent_job_id TEXT,
	
	-- File
	file_path TEXT,
	file_extension TEXT,
	
	-- Timestamps
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	completed_at DATETIME,
	
	FOREIGN KEY (parent_job_id) REFERENCES jobs(id)
);

CREATE INDEX IF NOT EXISTS idx_tracks_provider_id ON tracks(provider_id);
CREATE INDEX IF NOT EXISTS idx_tracks_parent_job_id ON tracks(parent_job_id);
CREATE INDEX IF NOT EXISTS idx_tracks_status ON tracks(status);

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
