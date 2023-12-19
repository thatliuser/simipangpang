// Actual command logic that gets wrapped

package discord

import (
	"fmt"
	"slices"
	"time"

	discord "github.com/bwmarrin/discordgo"
	"github.com/thatliuser/simipangpang/pkg/riot"
)

func (b *Bot) stats(name string, discrim string) ([]*discord.MessageEmbed, error) {
	account, err := b.client.AccountByRiotID(name, discrim)
	if err != nil {
		return nil, fmt.Errorf("couldn't retrieve account for %v#%v: %v", name, discrim, err)
	}
	author := &discord.MessageEmbedAuthor{
		Name:    fmt.Sprintf("%v#%v", account.Name, account.Discrim),
		IconURL: account.IconURL,
	}
	// This is so ugly but whatever lol
	const (
		descTemplate = `
**%v** / %v LP
**%v** wins / **%v** losses (%v%% winrate)
`
	)

	embedForMatch := func(match *riot.Match) (*discord.MessageEmbed, error) {
		colorForWin := func(won bool) int {
			if won {
				return 0x6EEB34
			} else {
				return 0xEB4C34
			}
		}
		descForMatch := func(match *riot.Match) string {
			won := ""
			if match.Won {
				won = "Victory"
			} else {
				won = "Defeat"
			}
			return fmt.Sprintf("**%v** (played <t:%v:R>)", won, match.Time.Unix())
		}
		champ, err := b.client.ChampionByID(int(match.Champ))
		if err != nil {
			return nil, err
		}
		champURL := b.client.IconURLForChamp(champ)
		return &discord.MessageEmbed{
			Color:  colorForWin(match.Won),
			Author: author,
			Thumbnail: &discord.MessageEmbedThumbnail{
				URL: champURL,
			},
			Description: descForMatch(match),
			Fields: []*discord.MessageEmbedField{
				{
					Name:   "Kills",
					Value:  fmt.Sprint(match.Kills),
					Inline: true,
				},
				{
					Name:   "Deaths",
					Value:  fmt.Sprint(match.Deaths),
					Inline: true,
				},
				{
					Name:   "Assists",
					Value:  fmt.Sprint(match.Assists),
					Inline: true,
				},
			},
		}, nil
	}

	lastWeek := time.Now().AddDate(0, 0, -7)
	matches, err := b.client.RankedMatchesSince(account, lastWeek)
	if err != nil {
		return nil, fmt.Errorf("couldn't get ranked matches: %v", err)
	}
	slices.SortFunc(matches, riot.CompareMatches)
	bestMatch, err := embedForMatch(matches[len(matches)-1])
	if err != nil {
		return nil, fmt.Errorf("couldn't create embed for match: %v", err)
	}
	worstMatch, err := embedForMatch(matches[0])
	if err != nil {
		return nil, fmt.Errorf("couldn't create embed for match: %v", err)
	}
	return []*discord.MessageEmbed{
		// Short profile stats
		{
			Color:  0xF7F12F,
			Author: author,
			Thumbnail: &discord.MessageEmbedThumbnail{
				URL: account.RankURL,
			},
			Description: fmt.Sprintf(descTemplate,
				account.Rank, account.Points,
				account.Wins, account.Losses, account.Wins*100/(account.Wins+account.Losses),
			),
		},
		bestMatch,
		worstMatch,
	}, nil
}
