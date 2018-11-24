package main

import (
	"context"

	"google.golang.org/appengine/log"

	"github.com/mjibson/goon"
)

type Config struct {
	_kind   string `goon:"kind,DevBotConfig"`
	Name    string `datastore:"-" goon:"id"`
	Value   string
	Comment string
}

func GetConfig(ctx context.Context, name string) (string, error) {
	c, err := FindConfig(ctx, name)
	if err != nil {
		log.Errorf(ctx, "Config not found for %s because of %v\n", name, err)
		return "", err
	}
	return c.Value, nil
}

func GetConfigWithDefault(ctx context.Context, name, defaultValue string) string {
	c, err := FindConfig(ctx, name)
	if err != nil {
		return defaultValue
	}
	return c.Value
}

func FindConfig(ctx context.Context, name string) (*Config, error) {
	g := goon.FromContext(ctx)
	c := &Config{Name: name}
	if err := g.Get(c); err != nil {
		return nil, err
	}
	return c, nil
}
