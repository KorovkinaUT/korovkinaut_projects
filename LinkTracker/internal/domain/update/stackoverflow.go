package update

import "time"

type StackOverflowEventType string

const (
	StackOverflowEventAnswer  StackOverflowEventType = "answer"
	StackOverflowEventComment StackOverflowEventType = "comment"
)

type StackOverflowEvent struct {
	Type          StackOverflowEventType
	QuestionTitle string
	Username      string
	CreationTime  time.Time
	Preview       string
}

func (e StackOverflowEvent) EventType() string {
	return string(e.Type)
}

func (e StackOverflowEvent) EventTitle() string {
	return string(e.QuestionTitle)
}

func (e StackOverflowEvent) EventAuthor() string {
	return string(e.Username)
}

func (e StackOverflowEvent) CreatedAt() time.Time {
	return e.CreationTime
}

func (e StackOverflowEvent) EventPreview() string {
	return string(e.Preview)
}
