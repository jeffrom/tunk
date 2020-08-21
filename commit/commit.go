// Package commit contains code for reading and processing commits.
package commit

type ReleaseType int

const (
	_ ReleaseType = iota

	ReleaseSkip
	ReleasePatch
	ReleaseMinor
	ReleaseMajor
)

func (t ReleaseType) String() string {
	switch t {
	case ReleaseSkip:
		return "SKIP"
	case ReleasePatch:
		return "PATCH"
	case ReleaseMinor:
		return "MINOR"
	case ReleaseMajor:
		return "MAJOR"
	case 0:
		return "<INVALID>"
	default:
		return "<UNKNOWN>"
	}
}
