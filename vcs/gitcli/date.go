package gitcli

import (
	"time"
)

// GIT_ISO8601 is the date format of git log
// 2020-08-17 16:26:10 -0700
const GIT_ISO8601 = "2006-01-02 15:04:05 -0700"

func ParseGitISO8601(s string) (time.Time, error) {
	return time.Parse(GIT_ISO8601, s)
}
