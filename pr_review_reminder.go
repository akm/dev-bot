package main

import (
	"context"
	"fmt"
	"io"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
)

type PRReviewReminder struct {
	UserToReviewUrls map[string][]string
}

func pullRequestReminder(ctx context.Context, team *SlackTeam) (*PRReviewReminder, error) {
	githubAuthToken, err := GetConfig(ctx, "GITHUB_AUTH_TOKEN")
	if err != nil {
		return nil, fmt.Errorf("Failed to get GITHUB_AUTH_TOKEN because of %v", err)
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubAuthToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// {"UserLogin": "PR URL"}
	sum := map[string][]string{}
	for _, repo := range team.Repositories {
		userToURLs, err := repo.getUserToReviewUrls(ctx, client)
		if err != nil {
			return nil, err
		}
		for user, urls := range userToURLs {
			if sum[user] == nil {
				sum[user] = []string{}
			}
			sum[user] = append(sum[user], urls...)
		}
	}

	return &PRReviewReminder{
		UserToReviewUrls: sum,
	}, nil
}

func (prs *PRReviewReminder) write(w io.Writer, mentionDictionary Dictionary) {
	fmt.Fprintf(w, "Pull Request Reminder\n")
	for user, urls := range prs.UserToReviewUrls {
		fmt.Fprintf(w, "\n%s\n", mentionDictionary.LookUp(user))
		for _, url := range urls {
			fmt.Fprintf(w, "%s\n", url)
		}
	}
}
