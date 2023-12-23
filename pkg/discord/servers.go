// Server interface code

package discord

import (
	"fmt"

	discord "github.com/bwmarrin/discordgo"
)

func (b *Bot) ServerFor(id string) (*Server, error) {
	server, ok := b.servers[id]
	if ok {
		return server, nil
	}

	b.log.Printf("Creating new server for id %v", id)
	server, err := NewServer(b, b.log.Writer(), id)
	if err != nil {
		return nil, fmt.Errorf("couldn't create new server from id %v: %v", id, err)
	}

	b.servers[id] = server
	return server, nil
}

func (b *Bot) UpdateTick(channel *discord.Channel) {
	b.log.Printf("Sending update embed to channel %v", channel.Mention())
	embeds, err := b.embedsFromVerb("all")
	if err != nil {
		b.log.Printf("Couldn't get embeds for update tick: %v", err)
	}

	_, err = b.session.ChannelMessageSendEmbeds(channel.ID, embeds)
	if err != nil {
		b.log.Printf("Error sending update tick message: %v", err)
	}
}
