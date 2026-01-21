package database

// Migration to add owner_id and is_shared to providers table
func (d *Database) migrateProviderOwnership() error {
	// Check if columns already exist
	var hasOwnerID, hasIsShared bool

	rows, err := d.db.Query("PRAGMA table_info(providers)")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var dfltValue interface{}

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			continue
		}

		if name == "owner_id" {
			hasOwnerID = true
		}
		if name == "is_shared" {
			hasIsShared = true
		}
	}

	// Add columns if they don't exist
	if !hasOwnerID {
		if _, err := d.db.Exec("ALTER TABLE providers ADD COLUMN owner_id TEXT"); err != nil {
			return err
		}
	}

	if !hasIsShared {
		if _, err := d.db.Exec("ALTER TABLE providers ADD COLUMN is_shared BOOLEAN NOT NULL DEFAULT 1"); err != nil {
			return err
		}
	}

	return nil
}
