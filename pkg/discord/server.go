// Per-server stuff

package discord

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	discord "github.com/bwmarrin/discordgo"
)

// Stuff that gets JSON'ed
type ServerState struct {
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
}

type Server struct {
	session *discord.Session
	guild   *discord.Guild
	channel *discord.Channel
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
	return fmt.Sprintf("%v/%v-back%v", stateDir, s.guild.ID, saveExt)
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

func (s *Server) Load(state ServerState) error {
	guild, err := Guild(s.session, state.GuildID)
	if err != nil {
		return fmt.Errorf("couldn't lookup guild %v by id: %v", state.GuildID, err)
	}
	s.guild = guild

	if state.ChannelID == "" {
		// No update channel set, so return early
		return nil
	}

	channel, err := Channel(s.session, state.ChannelID)
	if err != nil {
		return fmt.Errorf("couldn't lookup channel %v by id: %v", state.ChannelID, err)
	}
	s.channel = channel

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
	state := ServerState{}
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

	state := ServerState{
		GuildID:   s.guild.ID,
		ChannelID: s.channel.ID,
	}
	data, err := json.MarshalIndent(&state, "", "\t")
	if err != nil {
		return fmt.Errorf("couldn't marshal server data: %v", err)
	}
	if err := s.writeFile(s.SaveFileName(), data); err != nil {
		return fmt.Errorf("couldn't save server: %v", err)
	}

	return nil
}

func (s *Server) SetChannel(channelID string) error {
	channel, err := Channel(s.session, channelID)
	if err != nil {
		return fmt.Errorf("couldn't validate channel from id %v: %v", channelID, err)
	}
	me := s.session.State.User
	perms, err := UserChannelPermissions(s.session, me.ID, channel.ID)
	if err != nil {
		return fmt.Errorf("couldn't retrieve bot permissions for channel %v: %v", channel.Mention(), err)
	}
	if perms&discord.PermissionSendMessages == 0 {
		return fmt.Errorf("bot has no perms to send messages in channel %v", channel.Mention())
	}

	s.channel = channel
	return nil
}

func (s *Server) GetChannel() string {
	if s.channel == nil {
		return "<unset>"
	} else {
		return s.channel.Mention()
	}
}

func NewServer(session *discord.Session, state ServerState) (*Server, error) {
	s := &Server{
		session: session,
	}
	if err := s.Load(state); err != nil {
		return nil, fmt.Errorf("couldn't create server: %v", err)
	}

	return s, nil
}

func ServerFromFile(session *discord.Session, guildID string) (*Server, error) {
	s := &Server{
		session: session,
	}
	if err := s.LoadFile(); err != nil {
		return nil, fmt.Errorf("couldn't load server from file: %v", err)
	}

	return s, nil
}

func AllServerIDs() ([]string, error) {
	return fs.Glob(os.DirFS(stateDir), fmt.Sprintf("*%v", saveExt))
}
