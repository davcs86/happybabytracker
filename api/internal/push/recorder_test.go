package push

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRecorderKeepsSendsInOrder(t *testing.T) {
	r := &Recorder{}
	a := newSub(t, "https://push.example.org/secret-path-a")
	a.sub.ID = "dev-a"
	b := newSub(t, "https://push.example.org/secret-path-b")
	b.sub.ID = "dev-b"
	m1 := testMessage()
	m1.Payload = append([]byte(nil), testPayload...)
	m2 := Message{Payload: []byte(`{"v":1}`), TTL: 10 * time.Hour}
	for _, c := range []struct {
		s Subscription
		m Message
	}{{a.sub, m1}, {b.sub, m2}, {a.sub, m2}} {
		if got := r.Send(context.Background(), c.s, c.m); got != (Result{Outcome: Sent, Status: 201}) {
			t.Fatalf("%+v", got)
		}
	}
	want := []RecordedSend{
		{Device: "dev-a", Payload: testPayload, TTLSeconds: 1200, Urgency: "high", Topic: m1.Topic},
		{Device: "dev-b", Payload: m2.Payload, TTLSeconds: 3600, Urgency: "high"},
		{Device: "dev-a", Payload: m2.Payload, TTLSeconds: 3600, Urgency: "high"},
	}
	got := r.Sends()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sends\n got %+v\nwant %+v", got, want)
	}
	got[0].Payload[0] = 'X'
	got[0].Device = "changed"
	if again := r.Sends(); !reflect.DeepEqual(again, want) {
		t.Fatal("Sends returns the recorder's own storage")
	}
	m1.Payload[0] = 'Y'
	if again := r.Sends(); !bytes.Equal(again[0].Payload, testPayload) {
		t.Fatal("the recorder keeps the caller's payload slice")
	}
}

func TestRecorderRefusesLikeTheRealPath(t *testing.T) {
	r := &Recorder{}
	good := newSub(t, "https://push.example.org/secret-path")
	compressed := good.sub
	compressed.P256dh = b64.EncodeToString(good.ua.PublicKey().Bytes()[:33])
	invalid := good.sub
	invalid.P256dh = b64.EncodeToString(append([]byte{0x04}, bytes.Repeat([]byte{1}, 64)...))
	shortAuth := good.sub
	shortAuth.Auth = b64.EncodeToString(good.auth[:15])
	httpEP := good.sub
	httpEP.Endpoint = "http://push.example.org/secret-path"
	ipEP := good.sub
	ipEP.Endpoint = "https://10.0.0.1/secret-path"
	longEP := good.sub
	longEP.Endpoint = "https://push.example.org/" + strings.Repeat("a", 2049)
	for name, c := range map[string]struct {
		s Subscription
		m Message
	}{
		"compressed p256dh": {compressed, testMessage()},
		"invalid p256dh":    {invalid, testMessage()},
		"auth 15":           {shortAuth, testMessage()},
		"http endpoint":     {httpEP, testMessage()},
		"ip endpoint":       {ipEP, testMessage()},
		"long endpoint":     {longEP, testMessage()},
		"payload large":     {good.sub, Message{Payload: make([]byte, MaxPlaintext+1), TTL: time.Minute}},
		"topic chars":       {good.sub, Message{Payload: testPayload, TTL: time.Minute, Topic: "a/b"}},
	} {
		if got := r.Send(context.Background(), c.s, c.m); got != (Result{Outcome: Failed}) {
			t.Errorf("%s: %+v", name, got)
		}
	}
	if n := len(r.Sends()); n != 0 {
		t.Fatalf("%d refused sends recorded", n)
	}
}

func TestRecordedSendCarriesNoEndpointOrKey(t *testing.T) {
	var names []string
	rt := reflect.TypeFor[RecordedSend]()
	for i := range rt.NumField() {
		names = append(names, rt.Field(i).Name)
	}
	if got := strings.Join(names, ","); got != "Device,Payload,TTLSeconds,Urgency,Topic" {
		t.Fatalf("RecordedSend fields %s", got)
	}
	var _ Sender = (*Recorder)(nil)
}
