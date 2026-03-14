package dispatch

import (
	"fmt"
	"testing"
)

func TestQueryMode_String(t *testing.T) {
	tests := []struct {
		mode QueryMode
		want string
	}{
		{QueryLoose, "QueryLoose"},
		{QueryCanonical, "QueryCanonical"},
		{QueryStrict, "QueryStrict"},
		{QueryMode(99), "QueryMode(99)"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.mode.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCanonicalPolicy_String(t *testing.T) {
	tests := []struct {
		p    CanonicalPolicy
		want string
	}{
		{CanonicalIgnore, "CanonicalIgnore"},
		{CanonicalAnnotate, "CanonicalAnnotate"},
		{CanonicalRedirect, "CanonicalRedirect"},
		{CanonicalReject, "CanonicalReject"},
		{CanonicalPolicy(99), fmt.Sprintf("CanonicalPolicy(%d)", 99)},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.p.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSlashPolicy_String(t *testing.T) {
	tests := []struct {
		p    SlashPolicy
		want string
	}{
		{SlashIgnore, "SlashIgnore"},
		{SlashRedirect, "SlashRedirect"},
		{SlashPolicy(99), fmt.Sprintf("SlashPolicy(%d)", 99)},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.p.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
