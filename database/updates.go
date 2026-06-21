package database

import "context"

func UpdateDB() error {
	_, err := DB.NewRaw(
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS reputation integer NOT NULL DEFAULT 0`,
	).Exec(context.Background())
	if err != nil {
		return err
	}

	_, err = DB.NewRaw(`
		CREATE TABLE IF NOT EXISTS manual_opt_outs (
			discord_id numeric PRIMARY KEY,
			reason text,
			created_at timestamptz NOT NULL DEFAULT now()
		)
	`).Exec(context.Background())
	return err
}
