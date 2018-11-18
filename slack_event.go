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
