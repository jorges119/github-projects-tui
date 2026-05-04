package github

import (
	"context"
	"net/http"

	gogithub "github.com/google/go-github/v67/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type Client struct {
	REST    *gogithub.Client
	GraphQL *githubv4.Client
	token   string
}

func NewClient(token string) *Client {
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), src)

	return &Client{
		REST:    gogithub.NewClient(httpClient),
		GraphQL: githubv4.NewClient(httpClient),
		token:   token,
	}
}

func (c *Client) RawHTTP() *http.Client {
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: c.token})
	return oauth2.NewClient(context.Background(), src)
}
