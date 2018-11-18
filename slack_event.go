package main

import (
	"encoding/json"
	"net/http"

	"github.com/nlopes/slack/slackevents"
)

func ReplyToVerification(w http.ResponseWriter, reqBody string) {
	var r *slackevents.ChallengeResponse
	err := json.Unmarshal([]byte(reqBody), &r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "text")
	w.Write([]byte(r.Challenge))
}

func ChannelFromInnerEvent(innerEvent slackevents.EventsAPIInnerEvent) string {
	switch ev := innerEvent.Data.(type) {
	case *slackevents.AppMentionEvent: // Event Name: app_mention
		return ev.Channel
	case *slackevents.MessageEvent: // Event Name: message.channels
		return ev.Channel
	default:
		return ""
	}
}
