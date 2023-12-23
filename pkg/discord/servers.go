// Server*s*. Emphasis on the plural

package discord

import "fmt"

func (b *Bot) ServerFor(id string) (*Server, error) {
	server, ok := b.servers[id]
	if ok {
		return server, nil
	}

	server, err := NewServer(b.session, ServerState{
		GuildID:   id,
		ChannelID: "",
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't create new server from id %v: %v", id, err)
	}

	b.servers[id] = server
	return server, nil
}
