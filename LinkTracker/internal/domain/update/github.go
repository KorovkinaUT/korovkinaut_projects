package update

import "time"

type GitHubEventType string

const (
	GitHubEventIssue       GitHubEventType = "issue"
	GitHubEventPullRequest GitHubEventType = "pull_request"
)

type GitHubEvent struct {
	Type         GitHubEventType
	Title        string
	Username     string
	CreationTime time.Time
	Preview      string
}

func (e GitHubEvent) CreatedAt() time.Time {
	return e.CreationTime
}
