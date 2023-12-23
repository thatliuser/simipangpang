// Individual server (guild) state

package discord

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"
	"time"

	discord "github.com/bwmarrin/discordgo"
)

// Stuff that gets JSON'ed
type serverState struct {
	GuildID       string `json:"guild_id"`
	ChannelID     string `json:"channel_id"`
	PeriodMinutes int64  `json:"period_minutes"`
}

type Server struct {
	log     *log.Logger
	bot     *Bot
	guild   *discord.Guild
	channel *discord.Channel
	period  time.Duration // Should be in minutes
	ticker  *time.Ticker
	done    chan struct{}
}

const (
	stateDir = "state"
	saveExt  = ".json"
	fileMode = 0644 // rw-r--r--
	dirMode  = 0700 // rwx------
)

// Filename for saved information
func (s *Server) SaveFileName() string {
	return fmt.Sprintf("%v/%v%v", stateDir, s.guild.ID, saveExt)
}

func (s *Server) BackupFileName() string {
	return fmt.Sprintf("%v/%v-backup%v", stateDir, s.guild.ID, saveExt)
}

func (s *Server) readFile(name string) ([]byte, error) {
	contents, err := os.ReadFile(name)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("couldn't read contents of file %v: %v", name, err)
		} else {
			// No savefile but it's fine
			return nil, nil
		}
	}
	return contents, nil
}

func (s *Server) writeFile(name string, data []byte) error {
	if err := os.MkdirAll(stateDir, dirMode); err != nil {
		return fmt.Errorf("couldn't create state directory: %v", err)
	}

	if err := os.WriteFile(name, data, fileMode); err != nil {
		return fmt.Errorf("couldn't write data to file %v: %v", name, err)
	}

	return nil
}

func (s *Server) Load(state serverState) error {
	guild, err := s.bot.GuildByID(state.GuildID)
	if err != nil {
		return fmt.Errorf("couldn't lookup guild %v by id: %v", state.GuildID, err)
	}
	s.guild = guild

	if state.ChannelID != "" {
		// Validate channel ID since it's set
		channel, err := s.bot.ChannelByID(state.ChannelID)
		if err != nil {
			return fmt.Errorf("couldn't lookup channel %v by id: %v", state.ChannelID, err)
		}
		s.channel = channel
	} else {
		s.channel = nil
	}

	if state.PeriodMinutes != 0 {
		// Validate period since it's set
		if err := s.SetPeriod(state.PeriodMinutes); err != nil {
			return fmt.Errorf("invalid period: %v", err)
		}
	}

	return nil
}

func (s *Server) LoadFile() error {
	contents, err := s.readFile(s.SaveFileName())
	if err != nil {
		return fmt.Errorf("couldn't open server save: %v", err)
	} else if contents == nil {
		// No savefile but it's fine
		return nil
	}
	state := serverState{}
	if err := json.Unmarshal(contents, &state); err != nil {
		return fmt.Errorf("couldn't unmarshal contents: %v", err)
	}

	return s.Load(state)
}

func (s *Server) Backup() error {
	contents, err := s.readFile(s.SaveFileName())
	if err != nil {
		return fmt.Errorf("couldn't open server save: %v", err)
	} else if contents == nil {
		return nil
	}

	if err := s.writeFile(s.BackupFileName(), contents); err != nil {
		return fmt.Errorf("couldn't write to backup: %v", err)
	}

	return nil
}

func (s *Server) Save() error {
	// Backup the existing file if it exists
	if err := s.Backup(); err != nil {
		return fmt.Errorf("couldn't copy contents to backup: %v", err)
	}

	state := serverState{
		GuildID:       s.guild.ID,
		ChannelID:     "",
		PeriodMinutes: 0,
	}
	// Conditionally set these values
	if s.channel != nil {
		state.ChannelID = s.channel.ID
	}
	period := int64(s.period.Minutes())
	if period != 0 {
		state.PeriodMinutes = period
	}

	data, err := json.MarshalIndent(&state, "", "\t")
	if err != nil {
		return fmt.Errorf("couldn't marshal server data: %v", err)
	}
	if err := s.writeFile(s.SaveFileName(), data); err != nil {
		return fmt.Errorf("couldn't save server: %v", err)
	}

	s.log.Printf("Saved server %v", s.guild.ID)

	return nil
}

func (s *Server) SetChannel(channelID string) error {
	channel, err := s.bot.ChannelByID(channelID)
	if err != nil {
		return fmt.Errorf("couldn't validate channel from id %v: %v", channelID, err)
	}
	me := s.bot.User()
	perms, err := s.bot.PermsByIDs(me.ID, channel.ID)
	if err != nil {
		return fmt.Errorf("couldn't retrieve bot permissions for channel %v: %v", channel.Mention(), err)
	}
	if perms&discord.PermissionSendMessages == 0 {
		return fmt.Errorf("bot has no perms to send messages in channel %v", channel.Mention())
	}

	s.channel = channel
	s.log.Printf("Setting update channel for server %v to %v", s.guild.ID, s.channel.Mention())
	return nil
}

func (s *Server) GetChannel() string {
	if s.channel == nil {
		return "<unset>"
	} else {
		return s.channel.Mention()
	}
}

// This should be spawned in a Goroutine to listen to ticks
func (s *Server) tick() {
	for {
		select {
		case <-s.ticker.C:
			if s.channel != nil {
				s.bot.UpdateTick(s.channel)
			}
		case <-s.done:
			return
		}
	}
}

func (s *Server) refreshTicker() {
	if s.ticker == nil {
		s.log.Printf("Detected nil ticker, creating new ticker for server %v", s.guild.ID)
		// Setup ticker logic if not initialized
		s.ticker = time.NewTicker(s.period)
		s.done = make(chan struct{})
		go s.tick()
	}
	s.ticker.Reset(s.period)
}

func (s *Server) SetPeriod(minutes int64) error {
	if minutes <= 0 {
		return fmt.Errorf("negative or zero period not allowed")
	}

	period := time.Duration(minutes) * time.Minute
	s.period = period
	s.log.Printf("Set period for server %v to %v", s.guild.ID, s.period)

	s.refreshTicker()
	return nil
}

func (s *Server) GetPeriod() int64 {
	return int64(s.period.Minutes())
}

func (s *Server) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	if s.done != nil {
		s.done <- struct{}{}
	}
}

func NewServer(bot *Bot, output io.Writer, guildID string) (*Server, error) {
	s := &Server{
		bot: bot,
		log: log.New(output, "discord.Server: ", log.Ldate|log.Ltime),
	}
	state := serverState{
		GuildID: guildID,
	}
	if err := s.Load(state); err != nil {
		return nil, fmt.Errorf("couldn't create server: %v", err)
	}

	return s, nil
}

func ServerFromFile(bot *Bot, output io.Writer, guildID string) (*Server, error) {
	// Preliminary load so SaveFileName() doesn't deref a nil pointer
	s, err := NewServer(bot, output, guildID)
	if err != nil {
		return nil, err
	}

	if err := s.LoadFile(); err != nil {
		return nil, fmt.Errorf("couldn't load server from file: %v", err)
	}

	return s, nil
}

func AllServerIDs() ([]string, error) {
	ids := []string{}
	err := fs.WalkDir(os.DirFS(stateDir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || strings.Contains(path, "backup") {
			return nil
		}

		ids = append(ids, strings.TrimSuffix(path, saveExt))
		return nil
	})

	return ids, err
}
