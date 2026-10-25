package domain

import (
	"encoding/json"
	"testing"
)

func TestMessageUnmarshalDestinationIDStringForm(t *testing.T) {
	// 大 int64 destinationId 以字符串承载时不得被静默丢弃 ( B-2 )。
	val := `{"id":"01JD","destinationId":"134605228232032256","destinationType":"single","content":"hi","contentType":"text"}`

	m := &Message{}
	if err := json.Unmarshal([]byte(val), m); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if m.DestinationID != 134605228232032256 {
		t.Fatalf("expected destination id 134605228232032256, got %d", m.DestinationID)
	}
	if m.DestinationType != ChannelTypeSingle {
		t.Fatalf("expected single destination type, got %s", m.DestinationType)
	}
	if string(m.Content) != "hi" {
		t.Fatalf("expected content hi, got %s", m.Content)
	}
}

func TestMessageUnmarshalDestinationIDNumberForm(t *testing.T) {
	// 数字字面量形态保持兼容。
	val := `{"destinationId":10002,"destinationType":"group","content":"hi"}`

	m := &Message{}
	if err := json.Unmarshal([]byte(val), m); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if m.DestinationID != 10002 {
		t.Fatalf("expected destination id 10002, got %d", m.DestinationID)
	}
}

func TestMessageMarshalRoundTrip(t *testing.T) {
	m := Message{
		ID:              "01JD",
		DestinationID:   10002,
		DestinationType: ChannelTypeSingle,
		Content:         []byte("你好"),
		ContentType:     ContentTypeText,
	}

	val, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// content 必须按明文字符串 ( 而非 base64 ) 序列化。
	var raw map[string]any
	if err := json.Unmarshal(val, &raw); err != nil {
		t.Fatalf("failed to unmarshal as map: %v", err)
	}
	if raw["content"] != "你好" {
		t.Fatalf("expected plain content string, got %v", raw["content"])
	}

	got := &Message{}
	if err := json.Unmarshal(val, got); err != nil {
		t.Fatalf("failed to round trip: %v", err)
	}
	if got.ID != m.ID || got.DestinationID != m.DestinationID || string(got.Content) != string(m.Content) {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}
