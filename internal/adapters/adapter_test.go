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
