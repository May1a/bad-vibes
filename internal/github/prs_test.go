package github

import (
	"testing"

	"github.com/may/bad-vibes/internal/model"
)

func TestListStates(t *testing.T) {
	got := ListStates(false)
	if len(got) != 1 || got[0] != model.PRStateOpen {
		t.Fatalf("expected open state, got %+v", got)
	}

	got = ListStates(true)
	if len(got) != 2 || got[0] != model.PRStateClosed || got[1] != model.PRStateMerged {
		t.Fatalf("expected closed and merged states, got %+v", got)
	}
}
