package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

func main() {
	http.HandleFunc("/hello", sayHello)
	http.HandleFunc("/github/pull_requests", showPullRequestSummary)

	appengine.Main()
}

func sayHello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello!")
}

func showPullRequestSummary(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_AUTH_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	team := os.Getenv("TARGET_SLACK_TEAM")
	owner := os.Getenv("TARGET_GITHUB_ORG")
	repo := os.Getenv("TARGET_GITHUB_REPO")

	// https://api.slack.com/slash-commands#app_command_handling
	if team != r.PostFormValue("team_id") {
		err := fmt.Errorf("Invalid team ID")
		log.Errorf(ctx, "Can't tell you the detail because of %v", err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	prs, _, err := client.PullRequests.List(ctx, owner, repo, nil)
	if err != nil {
		log.Errorf(ctx, "Failed to client.PullRequests.List because of %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// {"UserLogin": "PR URL"}
	sum := map[string][]string{}
	for _, pr := range prs {
		if pr.URL == nil {
			continue
		}
		url := *pr.URL
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

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "Pull Request Reminder\n")
	for user, urls := range sum {
		fmt.Fprintln(w, "\n@%s\n", user)
		for _, url := range urls {
			fmt.Fprintln(w, "%s\n", url)
		}
	}
}
