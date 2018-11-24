package main

import (
	"context"

	"google.golang.org/appengine/log"

	"github.com/mjibson/goon"
)

type SlackTeam struct {
	_kind        string `goon:"kind,DevBotSlackTeam"`
	TeamID       string `datastore:"-" goon:"id"`
	Comment      string
	Repositories []GithubRepo
}

func GetSlackTeam(ctx context.Context, teamID string) (*SlackTeam, error) {
	t, err := FindSlackTeam(ctx, teamID)
	if err != nil {
		log.Errorf(ctx, "Config not found for %s because of %v\n", teamID, err)
		return nil, err
	}
	return t, nil
}

func FindSlackTeam(ctx context.Context, teamID string) (*SlackTeam, error) {
	g := goon.FromContext(ctx)
	c := &SlackTeam{TeamID: teamID}
	if err := g.Get(c); err != nil {
		return nil, err
	}
	return c, nil
}
