package parse

import (
	"testing"

	"github.com/may/bad-vibes/internal/model"
)

func TestParseRef(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		defaultRepo string
		want        model.PRRef
		wantErr     bool
	}{
		{
			name:        "bare number with default repo",
			raw:         "42",
			defaultRepo: "may1a/bad-vibes",
			want:        model.PRRef{Owner: "may1a", Repo: "bad-vibes", Number: 42},
		},
		{
			name:    "bare number without default repo",
			raw:     "42",
			wantErr: true,
		},
		{
			name: "short form",
			raw:  "may1a/bad-vibes#7",
			want: model.PRRef{Owner: "may1a", Repo: "bad-vibes", Number: 7},
		},
		{
			name: "full URL https",
			raw:  "https://github.com/may1a/bad-vibes/pull/99",
			want: model.PRRef{Owner: "may1a", Repo: "bad-vibes", Number: 99},
		},
		{
			name: "full URL http",
			raw:  "http://github.com/may1a/bad-vibes/pull/3",
			want: model.PRRef{Owner: "may1a", Repo: "bad-vibes", Number: 3},
		},
		{
			name:    "garbage input",
			raw:     "not-a-pr",
			wantErr: true,
		},
		{
			name:        "whitespace trimmed",
			raw:         "  42  ",
			defaultRepo: "owner/repo",
			want:        model.PRRef{Owner: "owner", Repo: "repo", Number: 42},
		},
		{
			name: "owner with dots and dashes",
			raw:  "my-org/my.repo#123",
			want: model.PRRef{Owner: "my-org", Repo: "my.repo", Number: 123},
		},
		{
			name: "URL with trailing content",
			raw:  "https://github.com/owner/repo/pull/42/files",
			want: model.PRRef{Owner: "owner", Repo: "repo", Number: 42},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRef(tt.raw, tt.defaultRepo)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseRef(%q, %q) error = %v, wantErr %v", tt.raw, tt.defaultRepo, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
