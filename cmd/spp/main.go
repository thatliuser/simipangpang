package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	env "github.com/joho/godotenv"
	"github.com/thatliuser/simipangpang/pkg/discord"
)

func interruptCtx() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}

func main() {
	ctx, cancel := interruptCtx()
	defer cancel()

	if err := env.Load(); err != nil {
		log.Fatalf("Couldn't load dotenv file: %v", err)
	}
	/*
		riot, err := riot.New()
		if err != nil {
			log.Fatalf("Couldn't create Riot client: %v", err)
		}
	*/
	bot, err := discord.New(nil, os.Stderr)
	if err != nil {
		log.Fatalf("Couldn't create Discord bot: %v", err)
	}

	if err := bot.Run(ctx); err != nil {
		log.Fatalf("Couldn't run Discord bot: %v", err)
	}
}
