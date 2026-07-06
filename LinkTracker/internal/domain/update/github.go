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

func (e GitHubEvent) EventType() string {
	return string(e.Type)
}

func (e GitHubEvent) EventTitle() string {
	return string(e.Title)
}

func (e GitHubEvent) EventAuthor() string {
	return string(e.Username)
}

func (e GitHubEvent) CreatedAt() time.Time {
	return e.CreationTime
}

func (e GitHubEvent) EventPreview() string {
	return string(e.Preview)
}
