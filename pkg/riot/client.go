package riot

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Kyagara/equinox"
	"github.com/Kyagara/equinox/api"
	"github.com/Kyagara/equinox/clients/ddragon"
	"github.com/Kyagara/equinox/clients/lol"
	"github.com/rs/zerolog"
)

type Client struct {
	client   *equinox.Equinox
	timeout  time.Duration
	version  string
	region   api.RegionalRoute
	platform lol.PlatformRoute
}

const tokenEnv = "RIOT_TOKEN"

func New(timeout time.Duration) (*Client, error) {
	token, ok := os.LookupEnv(tokenEnv)
	if !ok {
		return nil, fmt.Errorf("couldn't lookup token for riot client (%v) in environment", tokenEnv)
	}
	c := equinox.NewClientWithConfig(api.EquinoxConfig{
		Key:      token,
		LogLevel: zerolog.Disabled,
	})
	client := &Client{
		client:   c,
		timeout:  timeout,
		region:   api.AMERICAS,
		platform: lol.NA1,
	}
	ctx, cancel := client.newContext()
	defer cancel()
	// This is used in a lot of other places so it's useful to cache
	version, err := client.client.DDragon.Version.Latest(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't lookup datadragon version: %v", err)
	}
	client.version = version
	return client, nil
}

func (r *Client) newContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), r.timeout)
}

func (r *Client) ChampionByName(name string) (*ddragon.FullChampion, error) {
	ctx, cancel := r.newContext()
	defer cancel()
	champ, err := r.client.DDragon.Champion.ByName(ctx, r.version, ddragon.EnUS, name)
	if err != nil {
		return nil, fmt.Errorf("couldn't lookup champion by name: %v", err)
	} else {
		return champ, nil
	}
}

type Account struct {
	Name       string
	Discrim    string
	PUUID      string
	SummonerID string
	IconURL    string
	Rank       string
	RankURL    string
	Wins       int32
	Losses     int32
	Points     int32
}

func (r *Client) AccountByRiotID(name string, discrim string) (*Account, error) {
	ctx, cancel := r.newContext()
	defer cancel()
	user, err := r.client.Riot.AccountV1.ByRiotID(ctx, r.region, name, discrim)
	if err != nil {
		return nil, fmt.Errorf("couldn't lookup user by name %v#%v: %v", name, discrim, err)
	}
	summoner, err := r.client.LOL.SummonerV4.ByPUUID(ctx, r.platform, user.PUUID)
	if err != nil {
		return nil, fmt.Errorf("couldn't lookup summoner by puuid %v: %v", user.PUUID, err)
	}
	iconURL := fmt.Sprintf("https://ddragon.leagueoflegends.com/cdn/%v/img/profileicon/%v.png", r.version, summoner.ProfileIconID)
	leagues, err := r.client.LOL.LeagueV4.SummonerEntries(ctx, r.platform, summoner.ID)
	if err != nil {
		return nil, fmt.Errorf("couldn't lookup leagues for summoner by id %v: %v", summoner.ID, err)
	}
	if len(leagues) < 1 {
		return nil, fmt.Errorf("user is not ranked (league length is 0)")
	}
	league := leagues[0]
	tier := string(league.Tier)
	// Convert to not screaming case
	tier = fmt.Sprintf("%v%v", tier[0:1], strings.ToLower(tier[1:]))
	rank := fmt.Sprintf("%v %v", tier, league.Rank)
	rankURL := fmt.Sprintf("https://raw.communitydragon.org/latest/plugins/rcp-fe-lol-shared-components/global/default/%v.png", strings.ToLower(tier))
	return &Account{
		Name:       user.GameName,
		Discrim:    user.TagLine,
		PUUID:      user.PUUID,
		SummonerID: summoner.ID,
		IconURL:    iconURL,
		Rank:       rank,
		RankURL:    rankURL,
		Wins:       league.Wins,
		Losses:     league.Losses,
		Points:     league.LeaguePoints,
	}, nil
}

func (r *Client) MasteryForChamp(account *Account, champ *ddragon.FullChampion) (*lol.ChampionMasteryV4DTO, error) {
	ctx, cancel := r.newContext()
	defer cancel()
	id, err := strconv.ParseInt(champ.Key, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("couldn't convert champion key %v to int: %v", champ.Key, err)
	}
	mastery, err := r.client.LOL.ChampionMasteryV4.MasteryByPUUID(ctx, r.platform, account.PUUID, id)
	if err != nil {
		return nil, fmt.Errorf("couldn't get mastery for champion %v (id %v): %v", champ.Name, id, err)
	} else {
		return mastery, nil
	}
}

type Match struct {
	Kills   int32
	Deaths  int32
	Assists int32
	Won     bool
	Champ   int32
}

func (r *Client) MatchesByIDs(account *Account, ids []string) ([]Match, error) {
	ctx, cancel := r.newContext()
	defer cancel()
	matches := []Match{}

	infoForPlayer := func(account *Account, players []lol.ParticipantV5DTO) *lol.ParticipantV5DTO {
		for _, player := range players {
			if player.PUUID == account.PUUID {
				return &player
			}
		}
		return nil
	}

	for _, id := range ids {
		info, err := r.client.LOL.MatchV5.ByID(ctx, r.region, id)
		if err != nil {
			return nil, fmt.Errorf("error looking up match id %v: %v", id, err)
		}
		player := infoForPlayer(account, info.Info.Participants)
		if player == nil {
			return nil, fmt.Errorf("couldn't find player %v in match %v", account.Name, id)
		}

		matches = append(matches, Match{
			Kills:   player.Kills,
			Deaths:  player.Deaths,
			Assists: player.Assists,
			Won:     player.Win,
			Champ:   player.ChampionID,
		})
	}
	return matches, nil
}

func (r *Client) RankedMatchesSince(account *Account, since time.Time) ([]string, error) {
	ctx, cancel := r.newContext()
	defer cancel()
	start := since.Unix()
	end := time.Now().Unix()
	list, err := r.client.LOL.MatchV5.ListByPUUID(
		ctx, r.region, account.PUUID,
		start, end, -1, "ranked", 0, 100,
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't get match history for %v: %v", account.Name, err)
	} else {
		return list, nil
	}
}
