package main

import (
	"context"
	"fmt"
	"net/http"
	"server-go/common"
	"server-go/database"
	"server-go/database/schemas"
	"slices"
	"strconv"
	"sync"
	"time"

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

	// BanAllUsers(&bans)

	UpdateAllUsers(&bans)
	println("Getting All Bans Complete")
}

func UpdateAllUsers(bans *[]discord.Ban) {
	wg := sync.WaitGroup{}
	wg.Add(len(*bans))
	for _, user := range *bans {
		go func(ban discord.Ban) {
			user := schemas.URUser{
				DiscordID: strconv.FormatInt(int64(ban.User.ID), 10),
				Username:  common.Ternary(ban.User.Discriminator == "0", ban.User.Username, ban.User.Username+"#"+ban.User.Discriminator),
				AvatarURL: ban.User.AvatarURL(),
			}

			err, _ := database.DB.NewUpdate().Model(&user).Where("discord_id = ?", user.DiscordID).Exec(context.Background())
			if err != nil {
				fmt.Println(err)
			} else {
				println("Updated user: " + ban.User.ID.String() + " " + ban.User.Username)
			}
			wg.Done()
		}(user)
	}
	wg.Wait()
}

func BanAllUsers(bans *[]discord.Ban) {
	var allUsers []schemas.URUser

	// get all users that dont have Deleted User in their username
	err := database.DB.NewSelect().Model(&schemas.URUser{}).Where("username NOT like 'Deleted User%'").Where("opted_out = false").Order("id desc").Scan(context.Background(), &allUsers)

	if err != nil {
		panic(err)
	}

	usersToBan := []schemas.URUser{}

	// sort bans by user id so we can binary search
	slices.SortFunc(*bans, func(a, b discord.Ban) int {
		if b.User.ID < a.User.ID {
			return 1
		} else if b.User.ID > a.User.ID {
			return -1
		} else {
			return 0
		}
	})

	println("Deduplicating users")
	dedupTime := time.Now()
	// we filter all users that are banned
	for _, dbUser := range allUsers {

		dbUserInt, _ := strconv.ParseInt(dbUser.DiscordID, 10, 64)
		dbUserSnowflake := discord.UserID(dbUserInt)

		_, found := slices.BinarySearchFunc(*bans, dbUserSnowflake, func(ban discord.Ban, userId discord.UserID) int {
			if ban.User.ID == userId {
				return 0
			} else if ban.User.ID < userId {
				return -1
			} else {
				return 1
			}
		})

		if !found {
			usersToBan = append(usersToBan, dbUser)
		}
	}

	println("Deduplicated users")
	println("Deduplication took: " + time.Since(dedupTime).String())
	println("Users to ban: " + strconv.Itoa(len(usersToBan)))

	// release memory
	allUsers = []schemas.URUser{}
	*bans = []discord.Ban{}

	banIx := 0
	guildIx := 0

	guildId, _ := strconv.ParseInt(common.Config.GuildIDs[guildIx], 10, 64)

	increaseGuildIx := func() error {
		banIx = 0
		guildIx++

		if guildIx >= len(common.Config.GuildIDs) {
			return fmt.Errorf("No more guilds to ban in")
		}

		guildId, _ = strconv.ParseInt(common.Config.GuildIDs[guildIx], 10, 64)
		return nil
	}

	for _, user := range usersToBan {

		dcid, _ := strconv.ParseInt(user.DiscordID, 10, 64)

		err := client.Ban(discord.GuildID(guildId), discord.UserID(dcid), api.BanData{
			AuditLogReason: "Register",
		})

		banIx++

		if banIx > 1999 {
			err = increaseGuildIx()
		}

		if err != nil {
			if err.Error() == "Discord 400 error: Max number of bans for non-guild members have been exceeded. Try again later" {

				println("Switching guilds")

				err = increaseGuildIx()

				if err != nil {
					panic(err)
				}

			} else {
				panic(err)
			}
		} else {
			println("Banned: " + user.DiscordID + " " + user.Username)
		}
	}
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
		println("Fetching bans for guild: " + guild)
		bans, err := getGuildBans(guild)
		println("Fetched bans for guild: " + guild + " \nBan count in guild: " + strconv.Itoa(len(bans)))

		if err != nil {
			//	return nil, err
			println("too bad")
			// we in fact ignore the error
		}

		bannedUsers = append(bannedUsers, bans...)
	}

	println("Total bans: " + strconv.Itoa(len(bannedUsers)))

	return bannedUsers, nil
}
