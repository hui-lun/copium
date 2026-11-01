package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Wire format of the generateContent request, decoded by the fake server.
type part struct {
	Text string `json:"text"`
}

type content struct {
	Role  string `json:"role"`
	Parts []part `json:"parts"`
}

func TestChatSendsConversationAndParsesReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-goog-api-key") != "test-key" {
			t.Errorf("missing API key header")
		}
		if !strings.HasSuffix(r.URL.Path, "/models/test-model:generateContent") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var body struct {
			SystemInstruction struct {
				Parts []part `json:"parts"`
			} `json:"systemInstruction"`
			Contents []content `json:"contents"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.SystemInstruction.Parts[0].Text != "be brief" {
			t.Errorf("system instruction not sent")
		}
		if len(body.Contents) != 3 || body.Contents[1].Role != "model" {
			t.Errorf("conversation not sent correctly: %+v", body.Contents)
		}
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"Hello "},{"text":"there"}]}}]}`))
	}))
	defer srv.Close()

	c := newClient("test-key", "test-model", srv.URL)
	reply, err := c.Chat(context.Background(), "be brief", []Message{
		{Role: "user", Text: "hi"}, {Role: "model", Text: "hello"}, {Role: "user", Text: "how are you"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply != "Hello there" {
		t.Fatalf("got %q", reply)
	}
}

func TestChatReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"API key not valid"}}`))
	}))
	defer srv.Close()

	c := newClient("bad", "m", srv.URL)
	if _, err := c.Chat(context.Background(), "", []Message{{Role: "user", Text: "hi"}}); err == nil || !strings.Contains(err.Error(), "API key not valid") {
		t.Fatalf("expected API error, got %v", err)
	}
}

func TestChatFallsBackWhenModelBusy(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		if strings.Contains(r.URL.Path, "/busy:") {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"code":503,"message":"high demand","status":"UNAVAILABLE"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`))
	}))
	defer srv.Close()

	c := newClient("k", "busy, backup", srv.URL)
	reply, err := c.Chat(context.Background(), "", []Message{{Role: "user", Text: "hi"}})
	if err != nil || reply != "ok" || len(calls) != 2 {
		t.Fatalf("expected fallback to second model, got reply=%q err=%v calls=%v", reply, err, calls)
	}

	all := newClient("k", "busy", srv.URL)
	if _, err := all.Chat(context.Background(), "", []Message{{Role: "user", Text: "hi"}}); !errors.Is(err, ErrBusy) {
		t.Fatalf("expected ErrBusy when every model is busy, got %v", err)
	}
}

func TestChatDisabledWithoutKey(t *testing.T) {
	if _, err := NewGemini("", "m").Chat(context.Background(), "", nil); !errors.Is(err, ErrDisabled) {
		t.Fatalf("expected ErrDisabled, got %v", err)
	}
}
