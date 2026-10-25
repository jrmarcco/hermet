package xgin

import (
	"encoding/json"
	"testing"
)

func TestU64MarshalAsDecimalString(t *testing.T) {
	// 超出 JS 安全整数范围的 int64 ID。
	val := U64(134605228232032256)

	b, err := json.Marshal(val)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	if string(b) != `"134605228232032256"` {
		t.Fatalf("expected quoted decimal string, got %s", b)
	}
}

func TestU64UnmarshalStringAndNumber(t *testing.T) {
	cases := []struct {
		name string
		json string
		want U64
	}{
		{"string form", `"134605228232032256"`, 134605228232032256},
		{"number form", `134605228232032256`, 134605228232032256},
		{"null", `null`, 0},
		{"empty string", `""`, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got U64
			if err := json.Unmarshal([]byte(tc.json), &got); err != nil {
				t.Fatalf("failed to unmarshal %s: %v", tc.json, err)
			}
			if got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got)
			}
		})
	}
}

func TestU64UnmarshalInvalid(t *testing.T) {
	var got U64
	if err := json.Unmarshal([]byte(`"not-a-number"`), &got); err == nil {
		t.Fatal("expected error for non-numeric string")
	}
}
