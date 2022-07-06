
package modules

import (
	"github.com/uptrace/bun"
	"context"
)

type StupitStat struct {
	bun.BaseModel `bun:"table:stupit_table,"`

	DiscordID int64 `bun:"discordid,"`
	Stupidity int32 `bun:"stupidity,"`
	SenderID int64 `bun:"senderdiscordid,"`
}

func GetStupidity(DB *bun.DB, discordID int64) (StupitStat, error) {
	var stat StupitStat
	err := DB.NewSelect().Where("discordid = ?", discordID).Model(&stat).Scan(context.Background())
	if err != nil {
		return stat, err
	}
	return stat, nil
}
