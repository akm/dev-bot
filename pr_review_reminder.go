package main

import (
	"fmt"
	"io"
)

type PRReviewReminder struct {
	UserToUrls map[string][]string
	UserNameToID map[string]string
}

func (prs *PRReviewReminder) write(w io.Writer) {
	fmt.Fprintf(w, "Pull Request Reminder\n")
	for user, urls := range prs.UserToUrls {
		// https://api.slack.com/docs/message-formatting#linking_to_channels_and_users
		userId := prs.UserNameToID[user]
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
