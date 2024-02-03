package main

import (
	"context"
	"errors"
	"log"
	"os"
	"server-go/common"
	"server-go/modules"
	"server-go/modules/bitmask"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

var commands = []api.CreateCommandData{
	{
		Name:        "addflag",
		Description: "Add Flag to a user (admin only)",
		Options: []discord.CommandOption{
			&discord.UserOption{
				OptionName:  "user",
				Description: "User to add flag to",
				Required:    true,
			},
			&discord.IntegerOption{
				OptionName:  "flag",
				Description: "Flag to add",
				Required:    true,
				Choices: []discord.IntegerChoice{
					{Name: "Donor", Value: bitmask.UserDonor},
					{Name: "Admin", Value: bitmask.UserAdmin},
					{Name: "Banned", Value: bitmask.UserBanned},
				},
			},
		},
	},
	{
		Name:        "removeflag",
		Description: "echo back the argument",
		Options: []discord.CommandOption{
			&discord.UserOption{
				OptionName:  "user",
				Description: "User to add flag to",
				Required:    true,
			},
			&discord.IntegerOption{
				OptionName:  "flag",
				Description: "Flag to remove",
				Required:    true,
				Choices: []discord.IntegerChoice{
					{Name: "Donor", Value: bitmask.UserDonor},
					{Name: "Admin", Value: bitmask.UserAdmin},
					{Name: "Banned", Value: bitmask.UserBanned},
				},
			},
		},
	},
	{
		Name:        "resettoken",
		Description: "Reset a user's token",
		Options: []discord.CommandOption{
			&discord.UserOption{
				OptionName:  "user",
				Description: "User to reset token for, leave blank for self",
				Required:    false,
			},
		},
	},
}

func main() {
	token := os.Getenv(common.Config.BotToken)
	if token == "" {
		log.Fatalln("No $BOT_TOKEN given.")
	}

	state := state.New("Bot " + token)
	state.AddIntents(gateway.IntentGuilds)
	state.AddHandler(func(*gateway.ReadyEvent) {
		me, _ := state.Me()
		log.Println("connected to the gateway as", me.Tag())
	})

	if err := cmdroute.OverwriteCommands(state, commands); err != nil {
		log.Fatalln("cannot update commands and its all vens fault:", err)
	}

	h := newHandler(state)
	state.AddInteractionHandler(h)

	if err := h.s.Connect(context.Background()); err != nil {
		log.Fatalln("cannot connect:", err)
	}

}

type handler struct {
	*cmdroute.Router
	s *state.State
}

func newHandler(s *state.State) *handler {
	h := &handler{s: s}

	h.Router = cmdroute.NewRouter()
	// Automatically defer handles if they're slow.
	h.Use(cmdroute.Deferrable(s, cmdroute.DeferOpts{}))
	h.AddFunc("addflag", h.addFlag)
	h.AddFunc("removeflag", h.removeFlag)
	h.AddFunc("resettoken", h.resetToken)
	return h
}

func (h *handler) resetToken(ctx context.Context, data cmdroute.CommandData) *api.InteractionResponseData {
	var options struct {
		User discord.Snowflake
	}

	if err := data.Options.Unmarshal(&options); err != nil {
		return errorResponse(err)
	}

	// if no user is given, use the user who sent the command
	if options.User == 0 {
		user := data.Event.User.ID

		err := modules.ResetToken(user.String())

		if err != nil {
			return errorResponse(errors.New("Error resetting token"))
		} else {
			return &api.InteractionResponseData{
				Content: option.NewNullableString("Successfully reset token"),
			}
		}

	} else {
		requester := data.Event.User.ID

		user, err := modules.GetDBUserViaDiscordID(string(requester))

		if err != nil {
			return errorResponse(errors.New("Error resetting token"))
		}

		if user.IsAdmin() {
			err := modules.ResetToken(options.User.String())

			if err != nil {
				return errorResponse(errors.New("Error resetting token"))
			} else {
				return &api.InteractionResponseData{
					Content: option.NewNullableString("Successfully reset token"),
				}
			}
		} else {
			return errorResponse(errors.New("You do not have permission to reset tokens"))
		}
	}
}

func (h *handler) addFlag(ctx context.Context, data cmdroute.CommandData) *api.InteractionResponseData {
	var options struct {
		User discord.Snowflake
		Flag int
	}

	if err := data.Options.Unmarshal(&options); err != nil {
		return errorResponse(err)
	}

	return &api.InteractionResponseData{
		Content: option.NewNullableString("Pong!"),
	}
}

func (h *handler) removeFlag(ctx context.Context, data cmdroute.CommandData) *api.InteractionResponseData {
	var options struct {
		User discord.Snowflake
		Flag int
	}

	if err := data.Options.Unmarshal(&options); err != nil {
		return errorResponse(err)
	}

	return &api.InteractionResponseData{
		Content: option.NewNullableString("explode"),
	}
}

func errorResponse(err error) *api.InteractionResponseData {
	return &api.InteractionResponseData{
		Content: option.NewNullableString("**Error:** " + err.Error()),
		Flags:   discord.EphemeralMessage,
	}
}
