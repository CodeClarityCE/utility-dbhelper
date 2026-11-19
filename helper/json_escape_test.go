package helper

import (
	"encoding/json"
	"testing"

	"github.com/uptrace/bun/dialect/pgdialect"
)

// Text that spells \u0000 must stay text in the SQL bun builds; bun 1.3.0 broke this (uptrace/bun#1472).
func TestAppendJSONKeepsEscapedBackslash(t *testing.T) {
	encoded, err := json.Marshal(map[string]string{"d": `x\u0000y`})
	if err != nil {
		t.Fatal(err)
	}
	got := string(pgdialect.New().AppendJSON(nil, encoded))
	want := `'{"d":"x\\u0000y"}'`
	if got != want {
		t.Fatalf("AppendJSON(%s) = %s, want %s", encoded, got, want)
	}
}
