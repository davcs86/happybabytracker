package push

import (
	"context"
	"net/http"
	"sync"
)

// RecordedSend is what /v1/_test/pushes may show: never the endpoint or the keys.
type RecordedSend struct {
	Device     string
	Payload    []byte
	TTLSeconds int
	Urgency    string
	Topic      string
}

// Recorder is the HBT_ALLOW_FAKE_PUSH sender: no DNS, dial or signature, but the syntax, key and payload checks stay.
type Recorder struct {
	mu    sync.Mutex
	sends []RecordedSend
}

func (r *Recorder) Send(_ context.Context, sub Subscription, m Message) Result {
	_, ttl, err := prepare(sub, m)
	if err != nil {
		return Result{Outcome: Failed}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sends = append(r.sends, RecordedSend{
		Device: sub.ID, Payload: append([]byte(nil), m.Payload...), TTLSeconds: int(ttl), Urgency: urgency, Topic: m.Topic,
	})
	return Result{Outcome: Sent, Status: http.StatusCreated}
}

// Sends returns a copy, in send order.
func (r *Recorder) Sends() []RecordedSend {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]RecordedSend, len(r.sends))
	for i, s := range r.sends {
		s.Payload = append([]byte(nil), s.Payload...)
		out[i] = s
	}
	return out
}
