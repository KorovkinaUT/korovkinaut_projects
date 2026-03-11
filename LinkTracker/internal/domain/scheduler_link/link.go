package schedulerlink

type LinkType string

const (
	TypeGitHub        LinkType = "github"
	TypeStackOverflow LinkType = "stackoverflow"
)

// Common interface for tracked links
type SchedulerLink interface {
	Type() LinkType
}