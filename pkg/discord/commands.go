// Actual command logic that gets wrapped

package discord

import (
	"fmt"

	discord "github.com/bwmarrin/discordgo"
)

const descTemplate = `
**%v** / %v LP
**%v** wins / **%v** losses (%v%% winrate)
`

func (b *Bot) stats(name string, discrim string) ([]*discord.MessageEmbed, error) {
	account, err := b.client.AccountByRiotID(name, discrim)
	if err != nil {
		return nil, fmt.Errorf("couldn't retrieve account for %v#%v: %v", name, discrim, err)
	}
	return []*discord.MessageEmbed{
		{
			Color: 0xF7F12F,
			Author: &discord.MessageEmbedAuthor{
				Name:    fmt.Sprintf("%v#%v", account.Name, account.Discrim),
				IconURL: account.IconURL,
			},
			Thumbnail: &discord.MessageEmbedThumbnail{
				URL: account.RankURL,
			},
			Description: fmt.Sprintf(descTemplate,
				account.Rank, account.Points,
				account.Wins, account.Losses, account.Wins*100/(account.Wins+account.Losses),
			),
		},
	}, nil
}
