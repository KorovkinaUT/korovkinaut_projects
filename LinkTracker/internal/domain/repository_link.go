package domain

// Link type stored in repository
type RepositoryLink struct {
	ID   int64
	URL  string
	Tags []string
}