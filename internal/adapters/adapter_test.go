package adapters

import (
	"context"
	"testing"
)

func TestRule34VideoSubscriptionIdentity(t *testing.T) {
	a := Rule34Video{}
	if !a.MatchURL("https://rule34video.com/models/akt/") {
		t.Fatal("URL should match")
	}
	sub, err := a.ParseSubscription(context.Background(), "https://rule34video.com/models/akt/", "creator")
	if err != nil {
		t.Fatal(err)
	}
	if sub.Platform != "rule34video" || sub.SourceID != "akt" || sub.Slug != "rule34video:akt" || sub.Type != "creator" {
		t.Fatalf("unexpected subscription: %+v", sub)
	}
}

func TestRule34VideoSatisfiesAdapterContract(t *testing.T) {
	var _ Adapter = Rule34Video{}
	registry := NewRegistry(Rule34Video{})
	got, err := registry.Match("https://rule34video.com/models/akt/")
	if err != nil || got.ID() != "rule34video" {
		t.Fatalf("adapter routing failed: %v", err)
	}
	if _, err = registry.ByID("unsupported"); err == nil {
		t.Fatal("unknown platform must not silently fall back")
	}
}
