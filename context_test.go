package dispatch

import (
	"context"
	"testing"
)

func TestRouteNameFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	name, ok := RouteNameFromContext(ctx)
	if ok {
		t.Error("expected ok=false for empty context")
	}
	if name != "" {
		t.Errorf("expected empty name, got %q", name)
	}
}

func TestParamsFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	p, ok := ParamsFromContext(ctx)
	if ok {
		t.Error("expected ok=false for empty context")
	}
	if p != nil {
		t.Error("expected nil params")
	}
}

func TestMatchFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	m, ok := MatchFromContext(ctx)
	if ok {
		t.Error("expected ok=false for empty context")
	}
	if m != nil {
		t.Error("expected nil match")
	}
}

func TestMatchFromContext_Stored(t *testing.T) {
	m := &Match{Name: "test", Params: Params{"id": "42"}}
	ctx := storeMatchInContext(context.Background(), m)

	got, ok := MatchFromContext(ctx)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got.Name != "test" {
		t.Errorf("expected name=test, got %q", got.Name)
	}

	name, ok := RouteNameFromContext(ctx)
	if !ok || name != "test" {
		t.Errorf("expected RouteNameFromContext=test, got %q", name)
	}

	params, ok := ParamsFromContext(ctx)
	if !ok || params.Get("id") != "42" {
		t.Error("expected ParamsFromContext to return id=42")
	}
}
