package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"

	"github.com/google/go-github/github"
	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

func main() {
	http.HandleFunc("/hello", sayHello)
	http.HandleFunc("/slack/subscribe", subscribeSlack)
	http.HandleFunc("/github/pull_requests", showPullRequestSummary)

	appengine.Main()
}

func sayHello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello!")
}

var FavoritePattern = regexp.MustCompile(`酒|ビール|ワイン|パクチー|肉|飲み`)

func subscribeSlack(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	slack_api := slack.New(os.Getenv("SLACK_OAUTH_ACCESS_TOKEN"))
	slack.OptionHTTPClient(urlfetch.Client(ctx))(slack_api)

	// https://api.slack.com/events-api#subscriptions
	// https://github.com/nlopes/slack
	// https://github.com/nlopes/slack/blob/master/examples/eventsapi/events.go
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	body := buf.String()
	eventsAPIEvent, e := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionVerifyToken(&slackevents.TokenComparator{os.Getenv("SLACK_VERIFICATION_TOKEN")}))
	if e != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	log.Debugf(ctx, "eventsAPIEvent: %v\n", eventsAPIEvent)
	log.Debugf(ctx, "eventsAPIEvent.Type: %v\n", eventsAPIEvent.Type)

	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "text")
		w.Write([]byte(r.Challenge))
	}
	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent

		log.Debugf(ctx, "innerEvent: [%T] %v\n", innerEvent, innerEvent)
		log.Debugf(ctx, "innerEvent.Data: [%T] %v\n", innerEvent.Data, innerEvent.Data)

		botInfo, err := slack_api.GetBotInfo("")
		if err != nil {
			log.Errorf(ctx, "Failed to slack_api.GetBotInfo because of %v\n", err)
			return
		}

		var msg string
		var channel string
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent: // Event Name: app_mention
			if botInfo.ID == ev.User {
				return
			}
			channel = ev.Channel
			msg = fmt.Sprintf("<@%s> said to me %s", ev.User, ev.Text)
		case *slackevents.MessageEvent: // Event Name: message.channels
			if botInfo.ID == ev.User {
				return
			}
			favorites := FavoritePattern.FindAllString(ev.Text, -1)
			if len(favorites) < 0 {
				return
			}
			channel = ev.Channel
			msg = fmt.Sprintf("<@%s> Did you say %s !?", ev.User, strings.Join(favorites, " and "))
		default:
			return
		}
		postParams := slack.PostMessageParameters{}
		channelID, timestamp, err := slack_api.PostMessage(channel, msg, postParams)
		if err != nil {
			log.Errorf(ctx, "Failed to slack_api.PostMessage because of %v\n", err)
			return
		}
		log.Debugf(ctx, "Succeed to slack_api.PostMessage channedID: %v, timestap: %v\n", channelID, timestamp)
	}
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

	// https://github.com/nlopes/slack
	slack_api := slack.New(os.Getenv("SLACK_OAUTH_ACCESS_TOKEN"))
	slack.OptionHTTPClient(urlfetch.Client(ctx))(slack_api)

	// Don't forget adding scopes at `OAuth & Permissions` page.
	// See https://api.slack.com/methods/users.list about scopes.
	users, err := slack_api.GetUsers()
	if err != nil {
		log.Errorf(ctx, "Failed to slack_api.GetUsers because of %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userNameToID := map[string]string{}
	for _, user := range users {
		log.Debugf(ctx, "user: %v\n", user)
		userNameToID[user.Profile.DisplayName] = user.ID
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Pull Request Reminder\n")
	for user, urls := range sum {
		// https://api.slack.com/docs/message-formatting#linking_to_channels_and_users
		userId := userNameToID[user]
		var mention string
		if userId == "" {
			mention = fmt.Sprintf("@%s", user)
		} else {
			mention = fmt.Sprintf("<@%s>", userId)
		}
		fmt.Fprintf(w, "\n%s\n", mention)
		for _, url := range urls {
			fmt.Fprintf(w, "%s\n", url)
		}
	}
}
