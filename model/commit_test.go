package model

import "testing"

func TestCommit(t *testing.T) {
	cmt := &Commit{ID: "deadbeefdeadbeef"}
	short := cmt.ShortID()
	expect := "deadbeef"
	if short != expect {
		t.Fatal("expected", expect, "got", short)
	}
}
