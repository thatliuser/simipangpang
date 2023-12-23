// Event listeners that tie everything together (and util to go along with them)

package discord

import (
	"fmt"
	"strings"
	"time"

	discord "github.com/bwmarrin/discordgo"
	"github.com/thatliuser/simipangpang/pkg/riot"
)

func (b *Bot) updateChannelFromOpts(guildID string, opts ...string) (string, error) {
	if len(opts) < 1 {
		return "", fmt.Errorf("didn't pass a verb to the update channel command")
	}
	server, err := b.ServerFor(guildID)
	if err != nil {
		return "", fmt.Errorf("couldn't get server for guild id %v: %v", guildID, err)
	}
	verb := opts[0]
	if verb == "get" {
		return fmt.Sprintf("The current update channel is %v", server.GetChannel()), nil
	} else if verb == "set" {
		if len(opts) < 2 {
			return "", fmt.Errorf("didn't pass a channel to be set")
		}

		channelID := opts[1]
		if err := server.SetChannel(channelID); err != nil {
			return "", err
		}

		return fmt.Sprintf("Success! The new update channel is %v", server.GetChannel()), nil
	} else {
		return "", fmt.Errorf("didn't pass a valid verb to the update channel command (%v)", verb)
	}
}

func (b *Bot) onUpdateChannel(i *discord.InteractionCreate) {
	opts := []string{}
	// First is a verb, if it exists
	for _, verb := range i.ApplicationCommandData().Options {
		opts = append(opts, verb.Name)
		// Second are actual options with values passed by the user
		for _, opt := range verb.Options {
			// This sucks
			opts = append(opts, opt.ChannelValue(b.session).ID)
		}
	}

	resp, err := b.updateChannelFromOpts(i.GuildID, opts...)
	if err != nil {
		resp = fmt.Sprintf(":warning: Failed with error: %v", err)
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
		return b.shortEmbed(account)
	}

	matches, err := b.matchesByPerformance(account)
	if err != nil {
		return nil, err
	}

	// Who needs clean code??? What is that even???
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
	} else {
		if _, err := b.session.ChannelMessageSendEmbedsReply(m.ChannelID, embeds, m.Reference()); err != nil {
			b.log.Printf("Error sending reply to message: %v", err)
		}
	}
}

func (b *Bot) onStatTick() {
	b.log.Printf("Ticked, sending updated stats")
	// TODO: Add back tickers but custom per server
	/*
		for _, server := range b.servers {
			embeds, err := b.embedsFromVerb("all")
			if err != nil {
				b.log.Printf("Error retrieving stats for user: %v", err)
			} else {
				if _, err := b.session.ChannelMessageSendEmbeds(server.UpdateChannel, embeds); err != nil {
					b.log.Printf("Error sending update message: %v", err)
				}
			}
		}
	*/
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
	// Create a ticker to send stats every once in a while
	b.ticker = time.NewTicker(time.Minute)
	go func() {
		for range b.ticker.C {
			b.onStatTick()
		}
	}()

	return nil
}
