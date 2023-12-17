// Guild-specific stuff

package discord

import (
	"fmt"

	discord "github.com/bwmarrin/discordgo"
)

type Server struct {
	UpdateChannel string
}

type Handler func(*discord.InteractionCreate)

type Command struct {
	command *discord.ApplicationCommand
	handler Handler
}

func (b *Bot) onUpdateChannel(i *discord.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	// Unsure if this is necessary but redundancy is good
	if len(options) < 1 {
		b.log.Printf("Got an empty set of options to update channel")
	}
	cmd := options[0]
	resp := ""
	switch cmd.Name {
	case "get":
		// Respond the current channel if set
		if val, ok := b.servers[i.GuildID]; ok {
			resp = fmt.Sprintf("The current update channel is <#%v>", val)
		} else {
			resp = "The current update channel is unset"
		}
	case "set":
		// Check if the channel is valid and if so, set the channel and respond with a status
		if len(cmd.Options) < 1 {
			b.log.Printf("Got no channel ID for set operation")
			return
		}
		channel := cmd.Options[0].ChannelValue(b.session)
		if channel != nil && channel.Type == discord.ChannelTypeGuildText {
			b.servers[i.GuildID] = Server{
				UpdateChannel: channel.ID,
			}
			resp = fmt.Sprintf("Success! The new update channel is <#%v>", channel.ID)
		} else {
			resp = "Channel passed is invalid; please try again with a valid channel"
		}
	default:
		b.log.Printf("Got invalid subcommand name for update channel command")
		return
	}

	if err := b.session.InteractionRespond(i.Interaction, &discord.InteractionResponse{
		Type: discord.InteractionResponseChannelMessageWithSource,
		Data: &discord.InteractionResponseData{
			Flags:   discord.MessageFlagsEphemeral,
			Content: resp,
		},
	}); err != nil {
		b.log.Printf("Error sending reply to message: %v", err)
	}
}

// TODO: Move the lookup into a different function so that the other Lookup call isn't weirdly distributed
func (b *Bot) onStats(i *discord.InteractionCreate) {
	resp, err := b.client.Lookup(riotUser)
	if err != nil {
		resp = "Couldn't look up user!"
	}
	if err := b.session.InteractionRespond(i.Interaction, &discord.InteractionResponse{
		Type: discord.InteractionResponseChannelMessageWithSource,
		Data: &discord.InteractionResponseData{
			Content: resp,
		},
	}); err != nil {
		b.log.Printf("Error sending reply to message: %v", err)
	}
}

// Add slash commands
func (b *Bot) addCommands() error {
	manage := int64(discord.PermissionManageServer)
	// This is declared locally because you should only call this once on init
	commands := []Command{
		{
			command: &discord.ApplicationCommand{
				Name:        "update_channel",
				Description: "Set or get the update channel for the server",
				Type:        discord.ChatApplicationCommand,
				// WTF is the point of this
				DefaultMemberPermissions: &manage,
				Options: []*discord.ApplicationCommandOption{
					{
						Name:        "get",
						Description: "Get the update channel for the server",
						Type:        discord.ApplicationCommandOptionSubCommand,
						Options:     []*discord.ApplicationCommandOption{},
					},
					{
						Name:        "set",
						Description: "Set the update channel for the server",
						Type:        discord.ApplicationCommandOptionSubCommand,
						Options: []*discord.ApplicationCommandOption{
							{
								Name:        "channel",
								Description: "The new channel to be set as the update channel",
								Type:        discord.ApplicationCommandOptionChannel,
								Required:    true,
							},
						},
					},
				},
			},
			handler: b.onUpdateChannel,
		},
		{
			command: &discord.ApplicationCommand{
				Name:        "stats",
				Description: "Get simipangpang's stats",
				Type:        discord.ChatApplicationCommand,
			},
			handler: b.onStats,
		},
	}

	handlerMap := make(map[string]Handler)
	for _, c := range commands {
		if _, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, "", c.command); err != nil {
			return fmt.Errorf("couldn't register command %v: %v", c.command.Name, err)
		}
		handlerMap[c.command.Name] = c.handler
	}

	b.session.AddHandler(func(_ *discord.Session, i *discord.InteractionCreate) {
		command := i.ApplicationCommandData().Name
		if handler, ok := handlerMap[command]; ok {
			handler(i)
		} else {
			b.log.Printf("Passed invalid command name %v", command)
		}
	})

	return nil
}
