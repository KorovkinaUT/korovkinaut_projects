package schedulerlink

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type StackOverflowLink struct {
	QuestionID int64
}

func (l StackOverflowLink) Type() LinkType {
	return TypeStackOverflow
}

type StackOverflowParser struct{}

func (p StackOverflowParser) Parse(raw string) (SchedulerLink, error) {
	parsedURL, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse stackoverflow url: %w", err)
	}

	parts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "questions" || parts[1] == "" {
		return nil, fmt.Errorf("invalid stackoverflow url")
	}

	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid question id")
	}

	return StackOverflowLink{
		QuestionID: id,
	}, nil
}
