// Bot creation / run

package discord

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	discord "github.com/bwmarrin/discordgo"
	"github.com/thatliuser/simipangpang/pkg/riot"
)

type Bot struct {
	session *discord.Session
	client  *riot.Client
	log     *log.Logger
	servers map[string]*Server
	ticker  *time.Ticker
}

const (
	tokenEnv = "DISCORD_TOKEN"
)

func (b *Bot) Load() error {
	ids, err := AllServerIDs()
	if err != nil {
		return fmt.Errorf("couldn't get all server ids: %v", err)
	}
	for _, id := range ids {
		server, err := ServerFromFile(b.session, id)
		if err != nil {
			b.log.Printf("Couldn't load server with ID %v: %v", id, err)
		} else {
			b.servers[id] = server
		}
	}

	return nil
}

func (b *Bot) Save() {
	for id, server := range b.servers {
		b.log.Printf("Saving server %v", id)
		if err := server.Save(); err != nil {
			b.log.Printf("Failed to save server with ID %v: %v", id, err)
		}
	}
}

func New(client *riot.Client, output io.Writer) (*Bot, error) {
	token, ok := os.LookupEnv(tokenEnv)
	if !ok {
		return nil, fmt.Errorf("couldn't lookup token for discord bot (%v) in environment", tokenEnv)
	}
	session, err := discord.New(fmt.Sprintf("Bot %v", token))
	if err != nil {
		return nil, fmt.Errorf("couldn't create discord session: %v", err)
	}
	b := &Bot{
		session: (session),
		client:  client,
		log:     log.New(output, "discord.Bot: ", log.Ldate|log.Ltime),
		servers: make(map[string]*Server),
	}
	b.session.Identify.Intents = discord.IntentMessageContent | discord.IntentGuildMessages
	if err := b.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("couldn't load savefile: %v", err)
	}
	return b, nil
}

func (b *Bot) Run(ctx context.Context) error {
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("couldn't open discord session: %v", err)
	}
	defer b.session.Close()
	if err := b.addListeners(); err != nil {
		return fmt.Errorf("couldn't add slash commands: %v", err)
	}
	defer b.Stop()

	b.log.Println("Discord bot up!")

	// Wait for context to expire
	<-ctx.Done()

	return nil
}

func (b *Bot) Stop() {
	if b.ticker != nil {
		b.ticker.Stop()
	}
}
