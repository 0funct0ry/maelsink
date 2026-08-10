package events

import "sync"

// subscriberBufSize is the per-subscriber channel buffer. It only needs to
// smooth bursts between "event published" and "subscriber's own goroutine
// drains it" — the WebSocket hub has its own outbound buffering per client
// (internal/ws), so this is not the client's network backpressure path.
const subscriberBufSize = 32

// Bus is a simple in-process pub/sub broadcaster over Go channels — no
// external broker, per SPEC.md §5.5 / M7.0. The zero value is not usable;
// construct with NewBus.
type Bus struct {
	mu     sync.Mutex
	subs   map[int]chan Event
	next   int
	closed bool
}

// NewBus returns a ready-to-use Bus.
func NewBus() *Bus {
	return &Bus{subs: make(map[int]chan Event)}
}

// Subscribe registers a new subscriber and returns a receive-only channel
// of future events plus an unsubscribe function. Callers must call
// unsubscribe exactly once they're done listening (typically via defer), or
// the subscriber's channel and map entry leak. unsubscribe is safe to call
// more than once and safe to call concurrently with Publish.
func (b *Bus) Subscribe() (<-chan Event, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, subscriberBufSize)
	if b.closed {
		close(ch)
		return ch, func() {}
	}

	id := b.next
	b.next++
	b.subs[id] = ch

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			if sub, ok := b.subs[id]; ok {
				delete(b.subs, id)
				close(sub)
			}
		})
	}
	return ch, unsubscribe
}

// Publish delivers ev to every current subscriber. Delivery is non-blocking
// per subscriber: if a subscriber's buffer is full (it's slow or stuck),
// the event is dropped for that subscriber only rather than blocking the
// mutating request (SMTP ingestion, a REST delete, ...) that called
// Publish. Publish is a no-op once Close has been called.
func (b *Bus) Publish(ev Event) {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	chans := make([]chan Event, 0, len(b.subs))
	for _, ch := range b.subs {
		chans = append(chans, ch)
	}
	b.mu.Unlock()

	for _, ch := range chans {
		select {
		case ch <- ev:
		default:
		}
	}
}

// Close unsubscribes and closes every live subscriber channel and marks the
// bus closed; subsequent Subscribe calls return an already-closed channel,
// and subsequent Publish calls are no-ops. Intended for process shutdown.
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for id, ch := range b.subs {
		delete(b.subs, id)
		close(ch)
	}
}
