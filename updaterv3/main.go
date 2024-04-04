package main

import (
	"context"
	"fmt"
	"net/http"
	"server-go/common"
	"server-go/database"
	"server-go/database/schemas"
	"strconv"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
)

var client = api.NewClient("Bot " + common.Config.UpdaterBotToken)

func main() {
	database.InitDB()
	bans, err := GetAllBans()

	if err != nil {
		panic(err)
	}

	print(len(bans))
	var allUsers []schemas.URUser

	// get all users that dont have Deleted User in their username
	err = database.DB.NewSelect().Model(&schemas.URUser{}).Where("username NOT like 'Deleted User%'").Where("opted_out = false").Where("id < 119397").Order("id desc").Scan(context.Background(), &allUsers)

	if err != nil {
		panic(err)
	}

	usersToBan := []schemas.URUser{}

	// we filter all users that are banned
	for _, dbUser := range allUsers {

		isUserFound := false
		for _, ban := range bans {

			// this is probably horrible for performance, maybe I will write another struct that will serilize into int64
			if ban.User.ID.String() == dbUser.DiscordID {
				isUserFound = true
				break
			}

		}
		
		if !isUserFound {
			usersToBan = append(usersToBan, dbUser)
		}
	}

	// release memory
	allUsers = []schemas.URUser{}
	bans = []discord.Ban{}

	// for _, user := range usersToBan {

	// 	dcid, _ := strconv.ParseInt(user.DiscordID, 10, 64)

	// 	err := client.Ban(1222916513804062770, discord.UserID(dcid), api.BanData{
	// 		AuditLogReason: "Register",
	// 	})

	// 	if err != nil {
	// 		panic(err)
	// 	} else {
	// 		println("Banned: " + user.DiscordID + " " + user.Username)
	// 	}
	// }
}

func getGuildBans(guildId string) ([]discord.Ban, error) {
	var bans []discord.Ban

	endpoint := api.EndpointGuilds + guildId + "/bans"

	var after int64 = 0

	for {
		response := []discord.Ban{}
		err := client.RequestJSON(&response, "GET", endpoint+"?after="+strconv.FormatInt(after, 10),
			httputil.WithHeaders(http.Header{
				"Authorization": {"Bot " + common.Config.UpdaterBotToken},
			}))

		println(strconv.FormatInt(after, 10))
		println(len(response))

		if err != nil {
			fmt.Println("Error getting bans: " + err.Error())
			break
		}

		if len(response) == 0 {
			break
		}

		println(response[len(response)-1].User.ID)
		after = int64(response[len(response)-1].User.ID)
		bans = append(bans, response...)
		response = []discord.Ban{}
	}

	return bans, nil
}

func GetAllBans() ([]discord.Ban, error) {
	var bannedUsers []discord.Ban

	for _, guild := range common.Config.GuildIDs {
		bans, err := getGuildBans(guild)
		if err != nil {
			//	return nil, err
			println("too bad")
			// we in fact ignore the error
		}

		bannedUsers = append(bannedUsers, bans...)
	}

	return bannedUsers, nil
}
