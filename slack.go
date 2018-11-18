package main

import (
	"context"

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

func getUserNameToID(ctx context.Context, slack_api *slack.Client) (map[string]string, error) {
	// Don't forget adding scopes at `OAuth & Permissions` page.
	// See https://api.slack.com/methods/users.list about scopes.
	users, err := slack_api.GetUsers()
	if err != nil {
		log.Errorf(ctx, "Failed to slack_api.GetUsers because of %v", err)
		return nil, err
	}

	userNameToID := map[string]string{}
	for _, user := range users {
		log.Debugf(ctx, "user: %v\n", user)
		userNameToID[user.Profile.DisplayName] = user.ID
	}

	return userNameToID, nil
}
