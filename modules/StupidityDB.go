package modules
//TODO write addUser Function
//test code

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

func GetStupidity(DB *bun.DB, discordID int64) (int, error) {
	var stat []StupitStat
	err := DB.NewSelect().Where("discordid = ?", discordID).Model(&stat).Scan(context.Background())
	if err != nil {
		return 0, err
	}
	stupidity := CalculateStupidity(DB, stat)
	return stupidity, nil
}

func CalculateStupidity(DB *bun.DB, votes []StupitStat) int32 {
	var stupidity int32
	for _, vote := range votes {
		stupidity += vote.Stupidity
	}
	return int32(stupidity/int32(len(votes)))
} 




