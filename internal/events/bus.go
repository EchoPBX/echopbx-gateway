package events

import "sync"

type Event struct {
	Type string
	Data any
}

type Bus struct {
	mu   sync.RWMutex
	subs map[chan Event]struct{}
}

func NewBus() *Bus {
	return &Bus{
		subs: make(map[chan Event]struct{}),
	}
}

func (b *Bus) Subscribe() chan Event {
	ch := make(chan Event, 64)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *Bus) Unsubscribe(ch chan Event) {
	b.mu.Lock()
	delete(b.subs, ch)
	close(ch)
	b.mu.Unlock()
}

func (b *Bus) Publish(ev Event) {
	b.mu.RLock()
	for ch := range b.subs {
		select {
		case ch <- ev:
		default:
		}
	}
	b.mu.RUnlock()
}
