// Event listeners that tie everything together (and util to go along with them)
package discord

import (
	"fmt"
	"strings"

	discord "github.com/bwmarrin/discordgo"
	"github.com/thatliuser/simipangpang/pkg/riot"
)

func (b *Bot) updateChannelFromVerb(server *Server, verb string, opts ...*discord.Channel) (string, error) {
	switch verb {
	case "get":
		return fmt.Sprintf("The current update channel is %v", server.GetChannel()), nil
	case "set":
		if len(opts) != 1 {
			return "", fmt.Errorf("didn't pass a channel to be set")
		}

		channel := opts[0]
		if err := server.SetChannel(channel.ID); err != nil {
			return "", err
		}
		return fmt.Sprintf("Success! The new update channel is %v", server.GetChannel()), nil
	default:
		return "", fmt.Errorf("didn't pass a valid verb to the update channel command (%v)", verb)
	}
}

func (b *Bot) updatePeriodFromVerb(server *Server, verb string, opts ...int64) (string, error) {
	switch verb {
	case "get":
		period := server.GetPeriod()
		if period == 0 {
			return "The current update period is unset", nil
		} else {
			return fmt.Sprintf("The current update period is %v minutes", period), nil
		}
	case "set":
		if len(opts) != 1 {
			return "", fmt.Errorf("didn't pass a period to be set")
		}

		period := opts[0]
		if err := server.SetPeriod(period); err != nil {
			return "", err
		}
		return fmt.Sprintf("Success! The new update period is %v minutes", server.GetPeriod()), nil
	default:
		return "", fmt.Errorf("didn't pass a valid verb to the update period command (%v)", verb)
	}
}

func (b *Bot) updateSettingFromVerb(guildID string, setting string, verb string, opts ...*discord.ApplicationCommandInteractionDataOption) (string, error) {
	server, err := b.ServerFor(guildID)
	if err != nil {
		return "", fmt.Errorf("couldn't get server for guild id %v: %v", guildID, err)
	}
	switch setting {
	case "channel":
		// Realistically this should always be 0 or 1 but it's annoying to express that in Go
		channels := []*discord.Channel{}
		for _, opt := range opts {
			channels = append(channels, opt.ChannelValue(b.session))
		}
		return b.updateChannelFromVerb(server, verb, channels...)
	case "period":
		periods := []int64{}
		for _, opt := range opts {
			periods = append(periods, opt.IntValue())
		}
		return b.updatePeriodFromVerb(server, verb, periods...)
	default:
		return "", fmt.Errorf("didn't pass a valid option to the update setting command (%v)", setting)
	}
}

func (b *Bot) onUpdateConfig(i *discord.InteractionCreate) {
	// Validate input format
	settingList := i.ApplicationCommandData().Options
	if len(settingList) != 1 {
		return
	}

	// Setting option to be modified / viewed
	setting := settingList[0]
	if len(setting.Options) != 1 {
		return
	}
	// Whether to get or set the setting
	verb := setting.Options[0]
	opts := verb.Options

	resp, err := b.updateSettingFromVerb(i.GuildID, setting.Name, verb.Name, opts...)
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
	// Validate input format
	options := i.ApplicationCommandData().Options
	if len(options) != 1 {
		// Don't respond so it errors
		return
	}

	// Acknowledge the interaction first
	b.session.InteractionRespond(i.Interaction, &discord.InteractionResponse{
		Type: discord.InteractionResponseDeferredChannelMessageWithSource,
	})

	embeds, err := b.embedsFromVerb(options[0].Name)

	if err != nil {
		b.log.Printf("Error retrieving stats for user: %v", err)
		errString := fmt.Sprintf(":warning: Failed with error: %v", err)
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

func newUpdateSetting(name string, descName string, varType discord.ApplicationCommandOptionType) *discord.ApplicationCommandOption {
	return &discord.ApplicationCommandOption{
		Name:        name,
		Description: fmt.Sprintf("Set or get the %v for the server", descName),
		Type:        discord.ApplicationCommandOptionSubCommandGroup,
		Options: []*discord.ApplicationCommandOption{
			{
				Name:        "get",
				Description: fmt.Sprintf("Get the %v for the server", descName),
				Type:        discord.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "set",
				Description: fmt.Sprintf("Set the %v for the server", descName),
				Type:        discord.ApplicationCommandOptionSubCommand,
				Options: []*discord.ApplicationCommandOption{
					{
						Name:        name,
						Description: fmt.Sprintf("The new %v to be set", descName),
						Type:        varType,
						Required:    true,
					},
				},
			},
		},
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
				Name:        "update",
				Description: "Set or get the update channel for the server",
				Type:        discord.ChatApplicationCommand,
				// WTF is the point of making this a pointer
				DefaultMemberPermissions: &manage,
				Options: []*discord.ApplicationCommandOption{
					newUpdateSetting("channel", "update channel", discord.ApplicationCommandOptionChannel),
					newUpdateSetting("period", "update period (in minutes)", discord.ApplicationCommandOptionInteger),
				},
			},
			handler: b.onUpdateConfig,
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
