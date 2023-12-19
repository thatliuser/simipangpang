// Functions that generate Discord embeds from Riot info.

package discord

import (
	"fmt"
	"slices"
	"time"

	discord "github.com/bwmarrin/discordgo"
	"github.com/thatliuser/simipangpang/pkg/riot"
)

func (b *Bot) matchEmbed(account *riot.Account, match *riot.Match, caption string) ([]*discord.MessageEmbed, error) {
	colorForWin := func(won bool) int {
		if won {
			// Green-ish
			return 0x6EEB34
		} else {
			// Red-ish
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
		return nil, fmt.Errorf("couldn't create match stats: %v", err)
	}
	champURL := b.client.IconURLForChamp(champ)
	return []*discord.MessageEmbed{
		{
			Color: colorForWin(match.Won),
			Author: &discord.MessageEmbedAuthor{
				Name:    fmt.Sprintf("%v#%v", account.Name, account.Discrim),
				IconURL: account.IconURL,
			},
			Thumbnail: &discord.MessageEmbedThumbnail{
				URL: champURL,
			},
			Description: descForMatch(match),
			Footer: &discord.MessageEmbedFooter{
				Text: caption,
			},
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
		},
	}, nil
}

// Assumes matches are sorted by performance
func (b *Bot) bestMatchEmbed(account *riot.Account, matches []*riot.Match) ([]*discord.MessageEmbed, error) {
	bestMatch := matches[len(matches)-1]
	return b.matchEmbed(account, bestMatch, "Best match this week")
}

// Assumes matches are sorted by performance
func (b *Bot) worstMatchEmbed(account *riot.Account, matches []*riot.Match) ([]*discord.MessageEmbed, error) {
	worstMatch := matches[0]
	return b.matchEmbed(account, worstMatch, "Worst match this week")
}

func (b *Bot) matchesByPerformance(account *riot.Account) ([]*riot.Match, error) {
	matches, err := b.client.RankedMatchesSince(account, time.Now().AddDate(0, 0, -7))
	if err != nil {
		return nil, err
	} else {
		slices.SortFunc(matches, riot.CompareMatches)
		return matches, nil
	}
}

func (b *Bot) shortEmbed(account *riot.Account) []*discord.MessageEmbed {
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
			Description: fmt.Sprintf(
				"**%v** / %v LP\n"+
					"**%v** wins / **%v** losses (%v%% winrate)\n",
				account.Rank, account.Points,
				account.Wins, account.Losses, account.Wins*100/(account.Wins+account.Losses),
			),
			Footer: &discord.MessageEmbedFooter{
				Text: "Account stats",
			},
		},
	}
}

func concatSlices[Slice ~[]Elem, Elem any](slices ...Slice) Slice {
	out := Slice{}
	for _, slice := range slices {
		out = append(out, slice...)
	}
	return out
}

func (b *Bot) allEmbed(account *riot.Account, matches []*riot.Match) ([]*discord.MessageEmbed, error) {
	bestMatch, err := b.bestMatchEmbed(account, matches)
	if err != nil {
		return nil, err
	}
	worstMatch, err := b.worstMatchEmbed(account, matches)
	if err != nil {
		return nil, err
	}
	return concatSlices(
		b.shortEmbed(account),
		bestMatch,
		worstMatch,
	), nil
}
