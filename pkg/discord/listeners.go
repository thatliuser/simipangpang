// Event listeners that tie everything together (and util to go along with them)

package discord

import (
	"fmt"
	"strings"

	discord "github.com/bwmarrin/discordgo"
	"github.com/thatliuser/simipangpang/pkg/riot"
)

func (b *Bot) onUpdateChannel(i *discord.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	// Unsure if this is necessary but redundancy is good
	if len(options) < 1 {
		b.log.Printf("Got an empty set of options to update channel")
		return
	}
	cmd := options[0]
	resp := ""
	switch cmd.Name {
	case "get":
		// Respond the current channel if set
		if val, ok := b.servers[i.GuildID]; ok {
			resp = fmt.Sprintf("The current update channel is <#%v>", val.UpdateChannel)
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
			b.log.Printf("Setting update channel to %v for server %v", channel.ID, i.GuildID)
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

const (
	name    = "simipangpang"
	discrim = "NA1"
)

func (b *Bot) embedsFromVerb(verb string) ([]*discord.MessageEmbed, error) {
	account, err := b.client.AccountByRiotID(name, discrim)
	if err != nil {
		return nil, err
	}
	// We only need the account for this one
	if verb == "short" {
		return b.shortEmbed(account), nil
	}

	matches, err := b.matchesByPerformance(account)
	if err != nil {
		return nil, err
	}

	embedFunc := (func(*riot.Account, []*riot.Match) ([]*discord.MessageEmbed, error))(nil)
	switch verb {
	case "best":
		embedFunc = b.bestMatchEmbed
	case "worst":
		embedFunc = b.worstMatchEmbed
	case "all":
		embedFunc = b.allEmbed
	default:
		return nil, fmt.Errorf("verb not recognized: %v", verb)
	}

	return embedFunc(account, matches)
}

func (b *Bot) onStats(i *discord.InteractionCreate) {
	// Acknowledge the interaction first
	b.session.InteractionRespond(i.Interaction, &discord.InteractionResponse{
		Type: discord.InteractionResponseDeferredChannelMessageWithSource,
	})

	options := i.ApplicationCommandData().Options
	if len(options) != 1 {
		b.log.Printf("Length of options is too short or too long (%v)", len(options))
		return
	}

	embeds, err := b.embedsFromVerb(options[0].Name)

	if err != nil {
		b.log.Printf("Error retrieving stats for user: %v", err)
		errString := err.Error()
		if _, err := b.session.InteractionResponseEdit(i.Interaction, &discord.WebhookEdit{
			Content: &errString,
		}); err != nil {
			b.log.Printf("Error sending error replying to message: %v", err)
		}
	} else {
		// Edit the response for later
		if _, err := b.session.InteractionResponseEdit(i.Interaction, &discord.WebhookEdit{
			Embeds: &embeds,
		}); err != nil {
			b.log.Printf("Error sending reply to message: %v", err)
		}
	}
}

func (b *Bot) onMessage(_ *discord.Session, m *discord.MessageCreate) {
	// Ignore messages sent by ourselves
	if m.Author.ID == b.session.State.User.ID {
		return
	}

	if !strings.Contains(strings.ToLower(m.Content), name) {
		return
	}

	b.log.Printf("Got message '%v' from %v", m.Content, m.Author.Username)

	embeds, err := b.embedsFromVerb("all")
	if err != nil {
		b.log.Printf("Error retrieving stats for user: %v", err)
		// Ignoring error if reply isn't sent because it isn't so useful
		if _, err := b.session.ChannelMessageSendReply(m.ChannelID, err.Error(), m.Reference()); err != nil {
			b.log.Printf("Error sending error replying to message: %v", err)
		}
	} else {
		if _, err := b.session.ChannelMessageSendEmbedsReply(m.ChannelID, embeds, m.Reference()); err != nil {
			b.log.Printf("Error sending reply to message: %v", err)
		}
	}
}

// Actually add the functionality to the bot
func (b *Bot) addListeners() error {
	manage := int64(discord.PermissionManageServer)
	// This is declared locally because you should only call this once on init
	type Handler func(*discord.InteractionCreate)

	type Command struct {
		command *discord.ApplicationCommand
		handler Handler
	}

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
				Options: []*discord.ApplicationCommandOption{
					{
						Name:        "short",
						Description: "Get a stats summary",
						Type:        discord.ApplicationCommandOptionSubCommand,
					},
					{
						Name:        "best",
						Description: "Get the best match for the week",
						Type:        discord.ApplicationCommandOptionSubCommand,
					},
					{
						Name:        "worst",
						Description: "Get the worst match for the week",
						Type:        discord.ApplicationCommandOptionSubCommand,
					},
					{
						Name:        "all",
						Description: "Get all available stats",
						Type:        discord.ApplicationCommandOptionSubCommand,
					},
				},
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
	b.session.AddHandler(b.onMessage)

	return nil
}
