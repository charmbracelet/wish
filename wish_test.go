package wish

import (
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	_, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewServerWithOptions(t *testing.T) {
	_, err := NewServer(
		WithMaxTimeout(time.Second),
		WithAddress(":2222"),
	)
	if err != nil {
		t.Fatal(err)
	}
}
