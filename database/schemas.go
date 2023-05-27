package database

import (
	"context"
	"server-go/database/schemas"
)

func CreateSchemas() (err error) {
	err = CreateReviewDBSchemas()
	err = CreateStupidityDBSchemas()
	err = CreateTwitterReviewDBSchemas()
	return
}

func CreateReviewDBSchemas() error {
	models := []interface{}{
		(*schemas.URUser)(nil),
		(*schemas.ReviewReport)(nil),
		(*schemas.ReviewDBBanLog)(nil),
		(*schemas.ActionLog)(nil),
		(*schemas.ReviewDBAppeal)(nil),
		(*schemas.UserReview)(nil),
		(*schemas.UserBadge)(nil),
	}

	for _, model := range models {
		if _, err := DB.NewCreateTable().IfNotExists().Model(model).Exec(context.Background()); err != nil {
			return err
		}
	}

	return nil
}

func CreateStupidityDBSchemas() error {
	models := []any{
		(*schemas.StupitStat)(nil),
		(*schemas.UserInfo)(nil),
	}

	for _, model := range models {
		if _, err := DB.NewCreateTable().IfNotExists().Model(model).Exec(context.Background()); err != nil {
			return err
		}
	}
	return nil
}

func CreateTwitterReviewDBSchemas() error {
	models := []any{}
	// soon

	for _, model := range models {
		if _, err := DB.NewCreateTable().IfNotExists().Model(model).Exec(context.Background()); err != nil {
			return err
		}
	}
	return nil
}
