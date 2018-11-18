package main

import(
	"context"

	"google.golang.org/appengine/log"

	"github.com/google/go-github/github"
)


func getUserToUrls(ctx context.Context, client *github.Client, owner, repo string) (map[string][]string, error) {
	prs, _, err := client.PullRequests.List(ctx, owner, repo, nil)
	if err != nil {
		log.Errorf(ctx, "Failed to client.PullRequests.List because of %v", err)
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
