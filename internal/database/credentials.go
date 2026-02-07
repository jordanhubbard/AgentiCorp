package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jordanhubbard/loom/pkg/models"
)

// UpsertCredential creates or updates a credential in the database
func (d *Database) UpsertCredential(cred *models.Credential) error {
	now := time.Now()
	if cred.CreatedAt.IsZero() {
		cred.CreatedAt = now
	}
	cred.UpdatedAt = now

	_, err := d.db.Exec(`
		INSERT INTO credentials (id, project_id, type, private_key_encrypted, public_key, key_id, description, created_at, updated_at, rotated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			private_key_encrypted = excluded.private_key_encrypted,
			public_key = excluded.public_key,
			key_id = excluded.key_id,
			description = excluded.description,
			updated_at = excluded.updated_at,
			rotated_at = excluded.rotated_at
	`, cred.ID, cred.ProjectID, cred.Type, cred.PrivateKeyEncrypted, cred.PublicKey,
		cred.KeyID, cred.Description, cred.CreatedAt, cred.UpdatedAt, cred.RotatedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert credential: %w", err)
	}
	return nil
}

// GetCredentialByProjectID retrieves a credential by project ID
func (d *Database) GetCredentialByProjectID(projectID string) (*models.Credential, error) {
	row := d.db.QueryRow(`
		SELECT id, project_id, type, private_key_encrypted, public_key, key_id, description, created_at, updated_at, rotated_at
		FROM credentials WHERE project_id = ? LIMIT 1
	`, projectID)

	return scanCredential(row)
}

// GetCredential retrieves a credential by its ID
func (d *Database) GetCredential(id string) (*models.Credential, error) {
	row := d.db.QueryRow(`
		SELECT id, project_id, type, private_key_encrypted, public_key, key_id, description, created_at, updated_at, rotated_at
		FROM credentials WHERE id = ?
	`, id)

	return scanCredential(row)
}

// DeleteCredential removes a credential from the database
func (d *Database) DeleteCredential(id string) error {
	_, err := d.db.Exec(`DELETE FROM credentials WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}
	return nil
}

func scanCredential(row *sql.Row) (*models.Credential, error) {
	var cred models.Credential
	var keyID, description sql.NullString
	var rotatedAt sql.NullTime

	err := row.Scan(
		&cred.ID, &cred.ProjectID, &cred.Type,
		&cred.PrivateKeyEncrypted, &cred.PublicKey,
		&keyID, &description,
		&cred.CreatedAt, &cred.UpdatedAt, &rotatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan credential: %w", err)
	}

	if keyID.Valid {
		cred.KeyID = keyID.String
	}
	if description.Valid {
		cred.Description = description.String
	}
	if rotatedAt.Valid {
		cred.RotatedAt = &rotatedAt.Time
	}

	return &cred, nil
}
