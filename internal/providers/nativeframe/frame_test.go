package nativeframe

import (
	"bytes"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	var b bytes.Buffer
	if err := Write(&b, []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
	p, err := Read(&b)
	if err != nil {
		t.Fatal(err)
	}
	if string(p) != `{"ok":true}` {
		t.Fatalf("%s", p)
	}
}
