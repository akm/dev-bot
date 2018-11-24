package main

import (
	"context"
	"fmt"

	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"

	"github.com/nlopes/slack"
)

// https://github.com/nlopes/slack
func slackApi(ctx context.Context, oauthAccessToken string) *slack.Client {
	slack_api := slack.New(oauthAccessToken)
	slack.OptionHTTPClient(urlfetch.Client(ctx))(slack_api)
	return slack_api
}

type SlackUser struct {
	ID    string
	Names []string
}

type SlackUsers []*SlackUser

func getSlackUsers(ctx context.Context) (SlackUsers, error) {
	// https://github.com/nlopes/slack
	accessToken, err := GetConfig(ctx, "SLACK_OAUTH_ACCESS_TOKEN")
	if err != nil {
		return nil, fmt.Errorf("Failed to get SLACK_OAUTH_ACCESS_TOKEN because of %v", err)
	}

	slack_api := slackApi(ctx, accessToken)

	// Don't forget adding scopes at `OAuth & Permissions` page.
	// See https://api.slack.com/methods/users.list about scopes.
	users, err := slack_api.GetUsers()
	if err != nil {
		log.Errorf(ctx, "Failed to slack_api.GetUsers because of %v", err)
		return nil, err
	}

	r := SlackUsers{}
	for _, user := range users {
		log.Debugf(ctx, "user: %v\n", user)
		r = append(r, &SlackUser{
			ID: user.ID,
			Names: []string{
				user.Profile.DisplayName,
			},
		})
	}
	return r, nil
}

func (users SlackUsers) MaxNameLength() int {
	r := 0
	for _, user := range users {
		l := len(user.Names)
		if r < l {
			r = l
		}
	}
	return r
}

func (users SlackUsers) ToMention(name string) string {
	// https://api.slack.com/docs/message-formatting#linking_to_channels_and_users
	maxLen := users.MaxNameLength()
	for i := 0; i < maxLen; i++ {
		for _, user := range users {
			if i < len(user.Names) && user.Names[i] == name {
				return fmt.Sprintf("<@%s>", user.ID)
			}
		}
	}
	return fmt.Sprintf("@%s", name)
}
