// Package model contains abstract data models.
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
