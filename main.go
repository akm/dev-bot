package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"

	"github.com/google/go-github/github"
	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

func main() {
	http.HandleFunc("/hello", sayHello)
	http.HandleFunc("/slack/subscribe", subscribeSlack)
	http.HandleFunc("/github/pull_requests", showPullRequestReminder)

	appengine.Main()
}

func sayHello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello!")
}

var FavoritePattern = regexp.MustCompile(`酒|ビール|ワイン|パクチー|肉|飲み`)
var PullRequestPattern = regexp.MustCompile(`/pr|pull request`)

func subscribeSlack(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	slack_api := slackApi(ctx, os.Getenv("SLACK_OAUTH_ACCESS_TOKEN"))

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

	switch eventsAPIEvent.Type {
	case slackevents.URLVerification:
		ReplyToVerification(w, body)
	case  slackevents.CallbackEvent:
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
			switch {
			case PullRequestPattern.MatchString(ev.Text):
				msg = replyToPullRequestReminderMentioned(ctx, r, eventsAPIEvent, os.Getenv("TARGET_SLACK_TEAM"))
			default:
				msg = fmt.Sprintf("<@%s> Sorry, I can't understand your message: %s", ev.User, ev.Text)
			}
		case *slackevents.MessageEvent: // Event Name: message.channels
			if botInfo.ID == ev.User {
				return
			}
			channel = ev.Channel
			msg = reactToFavorites(ev)
			if msg == "" {
				return
			}
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

func reactToFavorites(ev *slackevents.MessageEvent) string {
	favorites := FavoritePattern.FindAllString(ev.Text, -1)
	if len(favorites) < 1 {
		return ""
	}
	return fmt.Sprintf("<@%s> Did you say %s !?", ev.User, strings.Join(favorites, " and "))
}

func replyToPullRequestReminderMentioned(ctx context.Context, r *http.Request, eventsAPIEvent slackevents.EventsAPIEvent, team string) string {
	// https://api.slack.com/slash-commands#app_command_handling
	if team == eventsAPIEvent.TeamID {
		reminder, err := pullRequestReminder(ctx, r, team)
		if err != nil {
			return fmt.Sprintf("Failed to get the reminder of your pull requests because of %v", err)
		} else {
			b := bytes.NewBuffer([]byte{})
			reminder.write(b)
			return b.String()
		}
	} else {
		return "Can't tell you the detail because you are in another team"
	}
}

func pullRequestReminder(ctx context.Context, r *http.Request, team string) (*PullRequestReminder, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_AUTH_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// {"UserLogin": "PR URL"}
	sum, err := getUserToUrls(ctx, client, os.Getenv("TARGET_GITHUB_ORG"), os.Getenv("TARGET_GITHUB_REPO"))
	if err != nil {
		return nil, err
	}

	// https://github.com/nlopes/slack
	slack_api := slackApi(ctx, os.Getenv("SLACK_OAUTH_ACCESS_TOKEN"))
	userNameToID, err := getUserNameToID(ctx, slack_api)
	if err != nil {
		return nil, err
	}

	return &PullRequestReminder{
		UserToUrls: sum,
		UserNameToID: userNameToID,
	}, nil
}

func showPullRequestReminder(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	team := os.Getenv("TARGET_SLACK_TEAM")
	// https://api.slack.com/slash-commands#app_command_handling
	if team != r.PostFormValue("team_id") {
		err := fmt.Errorf("Invalid team ID")
		log.Errorf(ctx, "Can't tell you the detail because of %v", err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	reminder, err := pullRequestReminder(ctx, r, team)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	reminder.write(w)
}
