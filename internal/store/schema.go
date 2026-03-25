package store

const Schema = `
CREATE TABLE IF NOT EXISTS jobs (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	status TEXT NOT NULL,
	progress REAL DEFAULT 0,
	source_id TEXT,
	parent_job_id TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	error TEXT
);

-- Prevent duplicate active jobs for same source
CREATE UNIQUE INDEX IF NOT EXISTS idx_jobs_active_source ON jobs(source_id, type) 
WHERE status IN ('queued', 'running');

CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);

CREATE TABLE IF NOT EXISTS playlists (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	provider_id TEXT UNIQUE NOT NULL,
	title TEXT NOT NULL,
	description TEXT,
	image_url TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_playlists_provider_id ON playlists(provider_id);

CREATE TABLE IF NOT EXISTS playlist_tracks (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	playlist_id INTEGER NOT NULL,
	track_id INTEGER NOT NULL,
	position INTEGER DEFAULT 0,
	added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
	FOREIGN KEY (track_id) REFERENCES tracks(id) ON DELETE CASCADE,
	UNIQUE(playlist_id, track_id)
);

CREATE INDEX IF NOT EXISTS idx_playlist_tracks_playlist ON playlist_tracks(playlist_id);
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_track ON playlist_tracks(track_id);

CREATE TABLE IF NOT EXISTS tracks (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	provider_id TEXT UNIQUE NOT NULL,
	
	-- Metadata
	title TEXT NOT NULL,
	artist TEXT,
	artists TEXT,  -- JSON array
	album TEXT,
	album_id TEXT,
	album_artist TEXT,
	album_artists TEXT,  -- JSON array
	artist_ids TEXT,     -- JSON array
	album_artist_ids TEXT, -- JSON array
	track_number INTEGER,
	disc_number INTEGER,
	total_tracks INTEGER,
	total_discs INTEGER,
	year INTEGER,
	genre TEXT,
	mood TEXT,
	style TEXT,
	language TEXT,
	country TEXT,
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
	barcode TEXT,
	catalog_number TEXT,
	release_type TEXT,
	release_id TEXT,
	recording_id TEXT,
	tags TEXT,  -- JSON array
	
	-- Processing
	status TEXT NOT NULL DEFAULT 'missing',
	error TEXT,
	parent_job_id TEXT,
	
	-- File
	file_path TEXT,
	file_extension TEXT,
	file_hash TEXT,
	etag TEXT,

	-- Timestamps
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	completed_at DATETIME,
	last_verified_at DATETIME,
	
	FOREIGN KEY (parent_job_id) REFERENCES jobs(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_tracks_provider_id ON tracks(provider_id);
CREATE INDEX IF NOT EXISTS idx_tracks_parent_job_id ON tracks(parent_job_id);
CREATE INDEX IF NOT EXISTS idx_tracks_status ON tracks(status);
CREATE INDEX IF NOT EXISTS idx_tracks_album_id ON tracks(album_id);
CREATE INDEX IF NOT EXISTS idx_tracks_created_at ON tracks(created_at DESC);

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

CREATE TABLE IF NOT EXISTS providers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	url TEXT UNIQUE NOT NULL,
	name TEXT,
	position INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_providers_position ON providers(position);
`
