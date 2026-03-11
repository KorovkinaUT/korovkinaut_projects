package schedulerlink

import (
	"fmt"
	"net/url"
	"strings"
)

type GitHubLink struct {
	Owner string
	Repo  string
}

func (l GitHubLink) Type() LinkType {
	return TypeGitHub
}

type GitHubParser struct{}

func (p GitHubParser) Parse(raw string) (SchedulerLink, error) {
	parsedURL, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse github url: %w", err)
	}

	parts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid github url")
	}

	return GitHubLink{
		Owner: parts[0],
		Repo:  parts[1],
	}, nil
}
