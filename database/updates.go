package database

import "context"

func UpdateDB() error {
	_, err := DB.NewRaw(
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS reputation integer NOT NULL DEFAULT 0`,
	).Exec(context.Background())
	return err
}
