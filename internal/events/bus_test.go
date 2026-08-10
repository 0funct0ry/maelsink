package events

import (
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestBus_PublishDeliversToAllSubscribers(t *testing.T) {
	b := NewBus()
	ch1, unsub1 := b.Subscribe()
	defer unsub1()
	ch2, unsub2 := b.Subscribe()
	defer unsub2()

	b.Publish(MessagesCleared())

	for _, ch := range []<-chan Event{ch1, ch2} {
		select {
		case ev := <-ch:
			if ev.Type != TypeMessagesCleared {
				t.Fatalf("got type %q, want %q", ev.Type, TypeMessagesCleared)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event")
		}
	}
}

func TestBus_UnsubscribeStopsDelivery(t *testing.T) {
	b := NewBus()
	ch, unsub := b.Subscribe()
	unsub()

	b.Publish(MessagesCleared())

	select {
	case ev, ok := <-ch:
		if ok {
			t.Fatalf("got unexpected event after unsubscribe: %+v", ev)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("channel neither closed nor received from after unsubscribe")
	}
}

func TestBus_SlowSubscriberDropsWithoutBlockingPublish(t *testing.T) {
	b := NewBus()
	slow, unsubSlow := b.Subscribe()
	defer unsubSlow()
	fast, unsubFast := b.Subscribe()
	defer unsubFast()

	// Fill the slow subscriber's buffer without draining it.
	for i := 0; i < subscriberBufSize+5; i++ {
		done := make(chan struct{})
		go func() {
			b.Publish(MessagesCleared())
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("Publish blocked on a full subscriber")
		}
	}

	// The fast subscriber should still have received at least one event.
	select {
	case <-fast:
	case <-time.After(time.Second):
		t.Fatal("fast subscriber received nothing")
	}

	// Drain slow to avoid leaving it as a red herring for the leak check.
	for {
		select {
		case <-slow:
		default:
			return
		}
	}
}

func TestBus_CloseUnblocksSubscribers(t *testing.T) {
	b := NewBus()
	ch, unsub := b.Subscribe()
	defer unsub()

	b.Close()

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected channel to be closed after Bus.Close")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for channel close")
	}

	// Publish after Close must not panic and must be a no-op.
	b.Publish(MessagesCleared())

	// Subscribe after Close returns an already-closed channel.
	ch2, unsub2 := b.Subscribe()
	defer unsub2()
	select {
	case _, ok := <-ch2:
		if ok {
			t.Fatal("expected post-Close Subscribe to return a closed channel")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for post-close channel close")
	}
}

func TestBus_Leak(t *testing.T) {
	defer goleak.VerifyNone(t)

	b := NewBus()
	for i := 0; i < 50; i++ {
		_, unsub := b.Subscribe()
		b.Publish(MessagesCleared())
		unsub()
	}
}
