// Look up user and role permissions

package discord

import (
	"log"

	discord "github.com/bwmarrin/discordgo"
)

// Getter wrappers
func Channel(s *discord.Session, channelID string) (*discord.Channel, error) {
	channel, err := s.State.Channel(channelID)
	if err != nil {
		channel, err = s.Channel(channelID)
		if err != nil {
			return nil, err
		}
		log.Printf("Adding channel %v to cache", channel.Mention())
		s.State.ChannelAdd(channel)
	} else {
		log.Printf("Cache hit on channel %v", channel.Mention())
	}
	return channel, nil
}

func Guild(s *discord.Session, guildID string) (*discord.Guild, error) {
	guild, err := s.State.Guild(guildID)
	if err != nil {
		guild, err = s.Guild(guildID)
		if err != nil {
			return nil, err
		}
		log.Printf("Adding guild %v to cache", guild.Name)
		s.State.GuildAdd(guild)
	} else {
		log.Printf("Cache hit on guild %v", guild.Name)
	}
	return guild, nil
}

func Member(s *discord.Session, guildID string, userID string) (*discord.Member, error) {
	member, err := s.State.Member(guildID, userID)
	if err != nil {
		member, err = s.GuildMember(guildID, userID)
		if err != nil {
			return nil, err
		}
		log.Printf("Adding member %v#%v to cache", member.User.Username, member.User.Discriminator)
		s.State.MemberAdd(member)
	} else {
		log.Printf("Cache hit on member %v#%v", member.User.Username, member.User.Discriminator)
	}
	return member, nil
}

// This is stupid
// The method discord.Session.UserChannelPermissions is deprecated so this is my own wrapper
// Tries the cache, then adds stuff required to the cache if that failed
func UserChannelPermissions(s *discord.Session, userID string, channelID string) (int64, error) {
	// Try state cache
	perms, err := s.State.UserChannelPermissions(userID, channelID)
	if err == nil {
		return perms, nil
	}

	// If that doesn't work add the variables needed to the state cache before calling it again
	channel, err := Channel(s, channelID)
	if err != nil {
		return 0, err
	}

	guild, err := Guild(s, channel.GuildID)
	if err != nil {
		return 0, err
	}

	_, err = Member(s, guild.ID, userID)
	if err != nil {
		return 0, err
	}

	return s.State.UserChannelPermissions(userID, channelID)
}
