package events

import "sync"
import "github.com/EchoPBX/echopbx-gateway/pkg/sdk"

type Event struct {
	Type string
	Data any
}

type Bus struct {
	mu   sync.RWMutex
	subs map[chan sdk.Event]struct{}
}

func NewBus() *Bus {
	return &Bus{
		subs: make(map[chan sdk.Event]struct{}),
	}
}

func (b *Bus) Subscribe() chan sdk.Event {
	ch := make(chan sdk.Event, 64)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *Bus) Unsubscribe(ch chan sdk.Event) {
	b.mu.Lock()
	delete(b.subs, ch)
	close(ch)
	b.mu.Unlock()
}

func (b *Bus) Publish(ev sdk.Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}
