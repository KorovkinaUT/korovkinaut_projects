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

func (e StackOverflowEvent) CreatedAt() time.Time {
	return e.CreationTime
}
