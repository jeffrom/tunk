package model

import "time"

type Commit struct {
	ID             string `json:"commit"`
	Author         string
	AuthorEmail    string
	AuthorDate     time.Time
	Committer      string
	CommitterEmail string
	CommitterDate  time.Time
	Subject        string
	Body           string
	// Branch string `json:"branch,omitempty"`
}

func (c *Commit) ShortID() string {
	if len(c.ID) < 8 {
		return c.ID
	}
	return c.ID[:8]
}
