package main

import (
	"context"

	"google.golang.org/appengine/log"

	"github.com/google/go-github/github"
)

type GtihubRepo struct {
	Org  string
	Name string
}

func (repo *GtihubRepo) String() string {
	return repo.Org + "/" + repo.Name
}

func getUserToReviewUrls(ctx context.Context, client *github.Client, repo *GtihubRepo) (map[string][]string, error) {
	prs, _, err := client.PullRequests.List(ctx, repo.Org, repo.Name, nil)
	if err != nil {
		log.Errorf(ctx, "Failed to client.PullRequests.List for %s because of %v", repo.String(), err)
		return nil, err
	}

	// {"UserLogin": "PR URL"}
	sum := map[string][]string{}
	for _, pr := range prs {
		if pr.URL == nil {
			continue
		}
		url := *pr.HTMLURL
		for _, user := range pr.RequestedReviewers {
			if user.Login == nil {
				continue
			}
			login := *user.Login
			if sum[login] == nil {
				sum[login] = []string{}
			}
			sum[login] = append(sum[login], url)
		}
	}

	return sum, nil
}
