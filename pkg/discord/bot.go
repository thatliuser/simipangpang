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

type Server struct {
	UpdateChannel string
}

type Bot struct {
	session *discord.Session
	client  *riot.Client
	log     *log.Logger
	servers map[string]Server
}

const (
	tokenEnv = "DISCORD_TOKEN"
	saveName = "servers"
	saveExt  = ".json"
	saveFile = saveName + saveExt
)

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
		session: session,
		client:  client,
		log:     log.New(output, "discord.Bot: ", log.Ldate|log.Ltime),
		servers: make(map[string]Server),
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

	b.log.Println("Discord bot up!")

	// Create a ticker to send stats every once in a while
	tick := time.NewTicker(time.Second * 10)
	defer tick.Stop()
	go func() {
		for range tick.C {
		}
	}()
	// Wait for context to expire
	<-ctx.Done()

	return nil
}
