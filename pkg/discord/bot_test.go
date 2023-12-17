package discord

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

var servers = map[string]Server{
	"test": {
		UpdateChannel: "lol",
	},
	"test2": {
		UpdateChannel: "lol2",
	},
}

func makeBot() *Bot {
	bot := &Bot{
		servers: make(map[string]Server),
	}
	for key, val := range servers {
		bot.servers[key] = val
	}
	return bot
}

func TestMain(m *testing.M) {
	code := m.Run()
	matches, err := fs.Glob(os.DirFS("."), "servers*.json")
	if err == nil {
		for _, match := range matches {
			os.Remove(match)
		}
	}
	os.Exit(code)
}

func TestSave(t *testing.T) {
	bot := makeBot()
	if err := bot.Save(); err != nil {
		t.Fatalf("Couldn't save servers: %v", err)
	}
	if err := os.Remove(saveFile); err != nil {
		t.Fatalf("Couldn't remove saved file: %v", err)
	}
}

func TestLoad(t *testing.T) {
	bot := makeBot()
	data, err := json.Marshal(bot.servers)
	// Clear out the servers
	bot.servers = make(map[string]Server)
	if err != nil {
		t.Fatalf("Couldn't marshal server map: %v", err)
	}
	file, err := os.Create(saveFile)
	if err != nil {
		t.Fatalf("Couldn't create savefile: %v", err)
	}
	if _, err := file.Write(data); err != nil {
		t.Fatalf("Couldn't write data to savefile: %v", err)
	}
	file.Close()
	if err := bot.Load(); err != nil {
		t.Fatalf("Couldn't load data to bot: %v", err)
	}
	if !reflect.DeepEqual(bot.servers, servers) {
		t.Log("Didn't properly load data to bot.")
		t.Logf("Expected: '%v'", servers)
		t.Logf("Found: '%v'", bot.servers)
		t.FailNow()
	}
}

// This only works properly if you copy dotenv to the discord directory
// So it can be kind of ignored
func TestCreate(t *testing.T) {
	_, err := New(nil, io.Discard)
	if err == nil {
		t.Fatalf("Created a bot despite the token not existing")
	}
	if err := godotenv.Load(); err != nil {
		t.Log("Couldn't load dotenv file, skipping")
		t.SkipNow()
	}
	if _, ok := os.LookupEnv(tokenEnv); !ok {
		t.Logf("Couldn't load token environment key %v, skipping", tokenEnv)
		t.SkipNow()
	}
	bot, err := New(nil, io.Discard)
	if err != nil {
		t.Fatalf("Bot failed to create: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.After(time.Second * 5)
		cancel()
	}()
	if err := bot.Run(ctx); err != nil {
		t.Fatalf("Bot failed with error: %v", err)
	}
}
