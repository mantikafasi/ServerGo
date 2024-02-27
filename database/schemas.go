package database

import (
	"context"
	"server-go/common"
	"server-go/database/schemas"
)

func CreateSchemas() (err error) {
	if !common.Config.Debug {
		err = CreateReviewDBSchemas()
		err = CreateStupidityDBSchemas()
		err = CreateTwitterReviewDBSchemas()
	}
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
		(*schemas.ReviewDBBanLog)(nil),
		(*schemas.Notification)(nil),
		(*schemas.Oauth2Token)(nil),
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
	models := []any{
		(*schemas.TwitterUser)(nil),
		(*schemas.TwitterUserReview)(nil),
		(*schemas.TwitterUserBadge)(nil),
		(*schemas.ReviewDBTwitterBanLog)(nil),
	}
	// soon

	for _, model := range models {
		if _, err := DB.NewCreateTable().IfNotExists().Model(model).Exec(context.Background()); err != nil {
			return err
		}
	}
	return nil
}
