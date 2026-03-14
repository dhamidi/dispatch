package dispatch

import "testing"

func TestCandidateScore_Beats_AllLevels(t *testing.T) {
	// Level 1: LiteralSegments
	t.Run("LiteralSegments", func(t *testing.T) {
		a := candidateScore{LiteralSegments: 3}
		b := candidateScore{LiteralSegments: 2}
		if !a.beats(b) {
			t.Error("more literal segments should win")
		}
		if b.beats(a) {
			t.Error("fewer literal segments should lose")
		}
	})

	// Level 2: ConstrainedVars
	t.Run("ConstrainedVars", func(t *testing.T) {
		a := candidateScore{LiteralSegments: 2, ConstrainedVars: 3}
		b := candidateScore{LiteralSegments: 2, ConstrainedVars: 1}
		if !a.beats(b) {
			t.Error("more constrained vars should win")
		}
		if b.beats(a) {
			t.Error("fewer constrained vars should lose")
		}
	})

	// Level 3: BroadVars (fewer is better)
	t.Run("BroadVars", func(t *testing.T) {
		a := candidateScore{LiteralSegments: 2, ConstrainedVars: 1, BroadVars: 1}
		b := candidateScore{LiteralSegments: 2, ConstrainedVars: 1, BroadVars: 3}
		if !a.beats(b) {
			t.Error("fewer broad vars should win")
		}
		if b.beats(a) {
			t.Error("more broad vars should lose")
		}
	})

	// Level 4: QueryMatches
	t.Run("QueryMatches", func(t *testing.T) {
		a := candidateScore{LiteralSegments: 2, ConstrainedVars: 1, BroadVars: 1, QueryMatches: 3}
		b := candidateScore{LiteralSegments: 2, ConstrainedVars: 1, BroadVars: 1, QueryMatches: 1}
		if !a.beats(b) {
			t.Error("more query matches should win")
		}
		if b.beats(a) {
			t.Error("fewer query matches should lose")
		}
	})

	// Level 5: Priority
	t.Run("Priority", func(t *testing.T) {
		a := candidateScore{LiteralSegments: 2, ConstrainedVars: 1, BroadVars: 1, QueryMatches: 1, Priority: 10}
		b := candidateScore{LiteralSegments: 2, ConstrainedVars: 1, BroadVars: 1, QueryMatches: 1, Priority: 5}
		if !a.beats(b) {
			t.Error("higher priority should win")
		}
		if b.beats(a) {
			t.Error("lower priority should lose")
		}
	})

	// Level 6: Registration (lower is better)
	t.Run("Registration", func(t *testing.T) {
		a := candidateScore{LiteralSegments: 2, ConstrainedVars: 1, BroadVars: 1, QueryMatches: 1, Priority: 5, Registration: 0}
		b := candidateScore{LiteralSegments: 2, ConstrainedVars: 1, BroadVars: 1, QueryMatches: 1, Priority: 5, Registration: 5}
		if !a.beats(b) {
			t.Error("earlier registration should win")
		}
		if b.beats(a) {
			t.Error("later registration should lose")
		}
	})
}
