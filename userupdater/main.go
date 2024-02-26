package main

import (
	"context"
	"fmt"
	"server-go/common"
	"server-go/database"
	"server-go/database/schemas"
	discord_utlils "server-go/modules/discord"
	"strconv"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
)

func main() {
	threadCount := 50
	database.InitDB()

	ArikawaState, err := state.New("Bot " + common.Config.BotToken)

	if err != nil {
		panic(err)
	}

	var allUsers []schemas.URUser
	botUsers := []schemas.URUser{}

	// get all users that dont have Deleted User in their username
	err = database.DB.NewSelect().Model(&schemas.URUser{}).Where("username NOT like 'Deleted User%'").Where("opted_out = false").Order("id desc").Scan(context.Background(), &allUsers)
	if err != nil {
		panic(err)
	}
	var lock sync.Mutex

	wg := sync.WaitGroup{}
	wg.Add(threadCount)

	for i := 0; i < threadCount; i++ {
		go func() {
			for true {
				lock.Lock()
				if len(allUsers) == 0 {
					wg.Done()
					lock.Unlock()
					break
				}
				user := allUsers[len(allUsers)-1]

				allUsers = allUsers[:len(allUsers)-1]

				// DO NOT UPDATE WARNING
				if user.DiscordID == "1134864775000629298" {
					lock.Unlock()
					continue
				}

				if user.RefreshToken == "" {
					botUsers = append(botUsers, user)

					lock.Unlock()
					continue
				}
				lock.Unlock()

				if user.AccessTokenExpiry.Before(time.Now()) {
					token, err := discord_utlils.RefreshToken(user.RefreshToken)
					if err != nil {

						if err.Error() == "oauth2: \"invalid_grant\"" {
							// refresh token expired, delete it
							user.RefreshToken = ""
							user.AccessToken = ""
							user.AccessTokenExpiry = time.Time{}

							// ven explode
							lock.Lock()
							botUsers = append(botUsers, user)
							lock.Unlock()

							_, err = database.DB.NewUpdate().Model(&user).Where("id = ?", user.ID).Exec(context.Background())
							if err != nil {
								fmt.Println(err)
							}

							continue
						}

						fmt.Println(err, " ", user.Username, " ", user.DiscordID)
						continue
					}

					user.AccessToken = token.AccessToken
					user.RefreshToken = token.RefreshToken
					user.AccessTokenExpiry = token.Expiry
				}

				discordUser, err := discord_utlils.GetUser(user.AccessToken)
				if err == nil {
					user.Username = common.Ternary(discordUser.Discriminator == "0", discordUser.Username, discordUser.Username+"#"+discordUser.Discriminator)
					user.AvatarURL = discordUser.AvatarURL()
				} else {
					fmt.Println(err)
				}

				database.DB.NewUpdate().Model(&user).Where("id = ?", user.ID).OmitZero().Exec(context.Background())

				println("Updated user via oauth: ", discordUser.Username, " ", user.DiscordID)
			}
		}()
	}

	wg.Wait()

	for _, user := range botUsers {
		discordID, err := strconv.ParseInt(user.DiscordID, 10, 64)
		if err != nil {
			fmt.Println(err)
			continue
		}

		discordUser, err := ArikawaState.User(discord.UserID(discordID))

		if err != nil {
			fmt.Println(err)
			continue
		}

		user.Username = common.Ternary(discordUser.Discriminator == "0", discordUser.Username, discordUser.Username+"#"+discordUser.Discriminator)
		user.AvatarURL = discordUser.AvatarURL()

		database.DB.NewUpdate().Model(&user).Where("id = ?", user.ID).OmitZero().Exec(context.Background())
		println("Updated user via bot ", discordUser.Username, " ", user.DiscordID)
	}
}
