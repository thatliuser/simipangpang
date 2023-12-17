package riot

import (
	"fmt"
	"net/http"
	"os"

	riot "github.com/yuhanfang/riot/apiclient"
	"github.com/yuhanfang/riot/ratelimit"
)

type Client struct {
	client riot.Client
}

const tokenEnv = "RIOT_TOKEN"

func New() (*Client, error) {
	token, ok := os.LookupEnv(tokenEnv)
	if !ok {
		return nil, fmt.Errorf("couldn't lookup token for riot client (%v) in environment", tokenEnv)
	}
	client := riot.New(token, &http.Client{}, ratelimit.NewLimiter())
	return &Client{client}, nil
}

func (r *Client) Lookup(user string) (string, error) {
	return "Unimplemented", nil
}
