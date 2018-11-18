package main

import (
	"context"

	"google.golang.org/appengine/urlfetch"

	"github.com/nlopes/slack"
)

// https://github.com/nlopes/slack
func slackApi(ctx context.Context, oauthAccessToken string) *slack.Client {
	slack_api := slack.New(oauthAccessToken)
	slack.OptionHTTPClient(urlfetch.Client(ctx))(slack_api)
	return slack_api
}
