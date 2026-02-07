package database

import "log"

// migrateCredentials creates the credentials table for storing encrypted SSH keys
func (d *Database) migrateCredentials() error {
	schema := `
	CREATE TABLE IF NOT EXISTS credentials (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'ssh_ed25519',
		private_key_encrypted TEXT NOT NULL,
		public_key TEXT NOT NULL,
		key_id TEXT,
		description TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		rotated_at DATETIME,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_credentials_project_id ON credentials(project_id);
	`

	if _, err := d.db.Exec(schema); err != nil {
		return err
	}

	log.Println("Credentials table migrated successfully")
	return nil
}
