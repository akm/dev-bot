package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"

	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

func main() {
	// for Slack Commands
	http.HandleFunc("/hello", sayHello)
	http.HandleFunc("/github/pull_requests", showPRReviewReminder)

	// for Slack Events API
	http.HandleFunc("/slack/subscribe", subscribeSlack)

	appengine.Main()
}

func sayHello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello!")
}

var FavoritePattern = regexp.MustCompile(`酒|ビール|ワイン|パクチー|肉|飲み`)
var PullRequestPattern = regexp.MustCompile(`/pr|pull request`)

func subscribeSlack(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	accessToken, err := GetConfig(ctx, "SLACK_OAUTH_ACCESS_TOKEN")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	slack_api := slackApi(ctx, accessToken)

	verificationToken, err := GetConfig(ctx, "SLACK_VERIFICATION_TOKEN")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// https://api.slack.com/events-api#subscriptions
	// https://github.com/nlopes/slack
	// https://github.com/nlopes/slack/blob/master/examples/eventsapi/events.go
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	body := buf.String()
	eventsAPIEvent, e := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionVerifyToken(&slackevents.TokenComparator{verificationToken}))
	if e != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	log.Debugf(ctx, "eventsAPIEvent: %v\n", eventsAPIEvent)
	log.Debugf(ctx, "eventsAPIEvent.Type: %v\n", eventsAPIEvent.Type)

	switch eventsAPIEvent.Type {
	case slackevents.URLVerification:
		ReplyToVerification(w, body)
	case slackevents.CallbackEvent:
		channel := ChannelFromInnerEvent(eventsAPIEvent.InnerEvent)
		var msg string
		botInfo, err := slack_api.GetBotInfo("")
		if err != nil {
			msg = fmt.Sprintf("Failed to slack_api.GetBotInfo because of %v\n", err)
		} else {
			msg = replyToCallbackEvent(ctx, r, eventsAPIEvent, botInfo.ID)
		}
		if msg == "" {
			return
		}
		postParams := slack.PostMessageParameters{}
		channelID, timestamp, err := slack_api.PostMessage(channel, msg, postParams)
		if err != nil {
			log.Errorf(ctx, "Failed to slack_api.PostMessage(%q, &q) because of %v\n", channel, msg, err)
			return
		}
		log.Debugf(ctx, "Succeed to slack_api.PostMessage(%q, &q) channedID: %v, timestap: %v\n", channel, msg, channelID, timestamp)
	}
}

func replyToCallbackEvent(ctx context.Context, r *http.Request, eventsAPIEvent slackevents.EventsAPIEvent, botID string) string {
	innerEvent := eventsAPIEvent.InnerEvent

	log.Debugf(ctx, "innerEvent: [%T] %v\n", innerEvent, innerEvent)
	log.Debugf(ctx, "innerEvent.Data: [%T] %v\n", innerEvent.Data, innerEvent.Data)

	switch ev := innerEvent.Data.(type) {
	case *slackevents.AppMentionEvent: // Event Name: app_mention
		if botID == ev.User {
			return ""
		}
		switch {
		case PullRequestPattern.MatchString(ev.Text):
			return replyToPRReviewReminderMentioned(ctx, r, eventsAPIEvent)
		default:
			return fmt.Sprintf("<@%s> Sorry, I can't understand your message: %s", ev.User, ev.Text)
		}
	case *slackevents.MessageEvent: // Event Name: message.channels
		if botID == ev.User {
			return ""
		}
		return reactToFavorites(ev)
	default:
		return ""
	}
}

func reactToFavorites(ev *slackevents.MessageEvent) string {
	favorites := FavoritePattern.FindAllString(ev.Text, -1)
	if len(favorites) < 1 {
		return ""
	}
	return fmt.Sprintf("<@%s> Did you say %s !?", ev.User, strings.Join(favorites, " and "))
}

func replyToPRReviewReminderMentioned(ctx context.Context, r *http.Request, eventsAPIEvent slackevents.EventsAPIEvent) string {
	team, err := GetSlackTeam(ctx, eventsAPIEvent.TeamID)
	if err != nil {
		return fmt.Sprintf("No configuration found for %s because of %v", eventsAPIEvent.TeamID, err)
	}

	// https://api.slack.com/slash-commands#app_command_handling
	reminder, err := pullRequestReminder(ctx, team)
	if err != nil {
		return fmt.Sprintf("Failed to get the reminder of your pull requests because of %v", err)
	}

	slackUsers, err := getSlackUsers(ctx)
	if err != nil {
		return fmt.Sprintf("Failed to get slack users because of %v", err)
	}

	b := bytes.NewBuffer([]byte{})
	reminder.write(b, slackUsers)
	return b.String()
}

func showPRReviewReminder(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	team, err := GetSlackTeam(ctx, r.PostFormValue("team_id"))
	if err != nil {
		err := fmt.Errorf("Invalid team ID")
		log.Errorf(ctx, "Can't tell you the detail because of %v", err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	reminder, err := pullRequestReminder(ctx, team)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slackUsers, err := getSlackUsers(ctx)
	if err != nil {
		log.Errorf(ctx, "Failed to get slack users because of %v", err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	reminder.write(w, slackUsers)
}
