package riot

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/Kyagara/equinox"
	"github.com/Kyagara/equinox/api"
	"github.com/Kyagara/equinox/clients/ddragon"
	"github.com/Kyagara/equinox/clients/lol"
	"github.com/Kyagara/equinox/clients/riot"
	"github.com/rs/zerolog"
)

type Client struct {
	client *equinox.Equinox
}

const tokenEnv = "RIOT_TOKEN"

func New() (*Client, error) {
	token, ok := os.LookupEnv(tokenEnv)
	if !ok {
		return nil, fmt.Errorf("couldn't lookup token for riot client (%v) in environment", tokenEnv)
	}
	client := equinox.NewClientWithConfig(api.EquinoxConfig{
		Key:      token,
		LogLevel: zerolog.Disabled,
	})
	return &Client{client}, nil
}

func (r *Client) MasteryFor(ctx context.Context, user *riot.AccountV1DTO, champ string) (*lol.ChampionMasteryV4DTO, error) {
	version, err := r.client.DDragon.Version.Latest(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't lookup latest datadragon version: %v", err)
	}
	yi, err := r.client.DDragon.Champion.ByName(ctx, version, ddragon.EnUS, "MasterYi")
	if err != nil {
		return nil, fmt.Errorf("couldn't lookup champion by name: %v", err)
	}
	id, err := strconv.ParseInt(yi.Key, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("couldn't convert champion key %v to int: %v", yi.Key, err)
	}
	mastery, err := r.client.LOL.ChampionMasteryV4.MasteryByPUUID(ctx, lol.NA1, user.PUUID, id)
	if err != nil {
		return nil, fmt.Errorf("couldn't get mastery for champion %v (id %v): %v", yi.Name, id, err)
	}
	return mastery, nil
}

func (r *Client) Lookup(ctx context.Context, name string, discrim string) (string, error) {
	user, err := r.client.Riot.AccountV1.ByRiotID(ctx, api.AMERICAS, name, discrim)
	if err != nil {
		return "", fmt.Errorf("couldn't lookup user by name %v#%v: %v", name, discrim, err)
	}
	champ := "MasterYi"
	mastery, err := r.MasteryFor(ctx, user, champ)
	if err != nil {
		return "", fmt.Errorf("couldn't lookup mastery for champion: %v", err)
	}
	return fmt.Sprintf("%v has %v mastery on %v", name, mastery.ChampionPoints, champ), nil
}
