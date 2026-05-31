package events

import (
	"testing"

	"ecomteam/internal/domain"
)

func TestBusDeliversToMatchingUser(t *testing.T) {
	b := New()
	ch, unsub := b.Subscribe("user-1")
	defer unsub()

	b.Publish(domain.Event{Type: domain.EventProgress, UserID: "user-1", Agent: "BENEFIT", Percent: 50})

	got := <-ch
	if got.Agent != "BENEFIT" || got.Percent != 50 {
		t.Fatalf("unexpected event: %+v", got)
	}
}

func TestBusFiltersByUser(t *testing.T) {
	b := New()
	ch, unsub := b.Subscribe("user-1")
	defer unsub()

	// Event for a different user must not arrive.
	b.Publish(domain.Event{Type: domain.EventProgress, UserID: "user-2"})

	select {
	case e := <-ch:
		t.Fatalf("did not expect an event, got %+v", e)
	default:
	}
}

func TestUnsubscribeRemovesSubscriber(t *testing.T) {
	b := New()
	_, unsub := b.Subscribe("user-1")
	if b.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber, got %d", b.SubscriberCount())
	}
	unsub()
	if b.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers after unsub, got %d", b.SubscriberCount())
	}
	// Double unsubscribe must be safe.
	unsub()
}

func TestPublishDoesNotBlockWhenBufferFull(t *testing.T) {
	b := New()
	_, unsub := b.Subscribe("user-1")
	defer unsub()
	// Flood beyond the buffer; must not deadlock.
	for i := 0; i < 1000; i++ {
		b.Publish(domain.Event{Type: domain.EventProgress, UserID: "user-1"})
	}
}
