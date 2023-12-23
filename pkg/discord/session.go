// Look up user and role permissions

package discord

import (
	discord "github.com/bwmarrin/discordgo"
)

// Getter wrappers
func (b *Bot) ChannelByID(channelID string) (*discord.Channel, error) {
	channel, err := b.session.State.Channel(channelID)
	if err != nil {
		channel, err = b.session.Channel(channelID)
		if err != nil {
			return nil, err
		}
		b.session.State.ChannelAdd(channel)
	}

	return channel, nil
}

func (b *Bot) GuildByID(guildID string) (*discord.Guild, error) {
	// We can't used a cached guild because the roles aren't necessarily saved in the cache
	guild, err := b.session.Guild(guildID)
	if err != nil {
		return nil, err
	}
	b.session.State.GuildAdd(guild)

	return guild, nil
}

func (b *Bot) MemberByIDs(guildID string, userID string) (*discord.Member, error) {
	member, err := b.session.State.Member(guildID, userID)
	if err != nil {
		member, err = b.session.GuildMember(guildID, userID)
		if err != nil {
			return nil, err
		}
		b.session.State.MemberAdd(member)
	}
	return member, nil
}

// This is stupid
// The method discord.Session.UserChannelPermissions is deprecated so this is my own wrapper
// Tries the cache, then adds stuff required to the cache if that failed
func (b *Bot) PermsByIDs(userID string, channelID string) (int64, error) {
	// Try state cache
	perms, err := b.session.State.UserChannelPermissions(userID, channelID)
	if err == nil {
		return perms, nil
	}

	// If that doesn't work add the variables needed to the state cache before calling it again
	channel, err := b.ChannelByID(channelID)
	if err != nil {
		return 0, err
	}

	guild, err := b.GuildByID(channel.GuildID)
	if err != nil {
		return 0, err
	}

	_, err = b.MemberByIDs(guild.ID, userID)
	if err != nil {
		return 0, err
	}

	return b.session.State.UserChannelPermissions(userID, channelID)
}

func (b *Bot) User() *discord.User {
	return b.session.State.User
}
