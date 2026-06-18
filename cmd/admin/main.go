package main

import (
	"context"
	"fmt"
	"os"
	"server-go/database"
	"server-go/database/schemas"
	"strconv"

	"github.com/uptrace/bun"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	database.InitDB()
	if err := database.UpdateDB(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "backfill-reputation":
		batchSize, err := parseBatchSize(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := backfillReputation(batchSize); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: go run ./cmd/admin <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  backfill-reputation [batch-size]  Recalculate users.reputation from review_votes")
}

func parseBatchSize(args []string) (int, error) {
	if len(args) == 0 {
		return 250, nil
	}

	batchSize, err := strconv.Atoi(args[0])
	if err != nil || batchSize <= 0 {
		return 0, fmt.Errorf("invalid batch size: %s", args[0])
	}
	return batchSize, nil
}

func backfillReputation(batchSize int) error {
	var lastID int32
	totalUpdated := 0

	for {
		var userIDs []int32
		err := database.DB.NewSelect().
			Model((*schemas.URUser)(nil)).
			Column("id").
			Where("id > ?", lastID).
			OrderExpr("id ASC").
			Limit(batchSize).
			Scan(context.Background(), &userIDs)
		if err != nil {
			return err
		}
		if len(userIDs) == 0 {
			fmt.Printf("done, updated %d users\n", totalUpdated)
			return nil
		}

		res, err := database.DB.NewRaw(`
			UPDATE users AS u
			SET reputation = calculated.reputation
			FROM (
				SELECT
					u2.id,
					COALESCE(SUM(CASE WHEN rv.id IS NULL THEN 0 WHEN rv.is_upvote THEN 1 ELSE -1 END), 0) AS reputation
				FROM users AS u2
				LEFT JOIN reviews AS r ON r.reviewer_id = u2.id
				LEFT JOIN review_votes AS rv ON rv.review_id = r.id
				WHERE u2.id IN (?)
				GROUP BY u2.id
			) AS calculated
			WHERE u.id = calculated.id
		`, bun.In(userIDs)).Exec(context.Background())
		if err != nil {
			return err
		}

		updated, _ := res.RowsAffected()
		totalUpdated += int(updated)
		lastID = userIDs[len(userIDs)-1]
		fmt.Printf("updated %d users, last id %d\n", totalUpdated, lastID)
	}
}
