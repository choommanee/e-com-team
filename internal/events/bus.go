// Package events provides an in-memory pub/sub bus used to push realtime
// progress and stats to connected dashboards over SSE.
package events

import (
	"sync"

	"ecomteam/internal/domain"
)

// Bus fans out domain.Event values to subscribers, filtered by user.
type Bus struct {
	mu     sync.RWMutex
	nextID int
	subs   map[int]*subscriber
}

type subscriber struct {
	userID string
	ch     chan domain.Event
}

// New returns an empty bus.
func New() *Bus {
	return &Bus{subs: map[int]*subscriber{}}
}

// Subscribe registers a listener for a user's events. It returns the receive
// channel and an unsubscribe function that must be called when done.
func (b *Bus) Subscribe(userID string) (<-chan domain.Event, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.nextID
	b.nextID++
	s := &subscriber{userID: userID, ch: make(chan domain.Event, 64)}
	b.subs[id] = s

	unsub := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subs[id]; ok {
			close(s.ch)
			delete(b.subs, id)
		}
	}
	return s.ch, unsub
}

// Publish delivers an event to all subscribers of its UserID. Delivery is
// non-blocking: if a subscriber's buffer is full, the event is dropped for that
// subscriber (the dashboard will catch up on the next stats refresh).
func (b *Bus) Publish(e domain.Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, s := range b.subs {
		if s.userID != e.UserID {
			continue
		}
		select {
		case s.ch <- e:
		default:
		}
	}
}

// SubscriberCount reports how many active subscribers exist (used in tests).
func (b *Bus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs)
}
