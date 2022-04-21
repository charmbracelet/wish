// go:genarate mockgen -package mocks -destination mocks/session.go github.com/gliderlabs/ssh Session
package wish

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/charmbracelet/wish/mocks"
	"github.com/golang/mock/gomock"
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

func TestError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	s := mocks.NewMockSession(ctrl)
	var w bytes.Buffer
	s.EXPECT().Stderr().Return(&w)

	Error(s, fmt.Errorf("foo"))
	requireEqual(t, "foo\n", w.String())
}

func TestFatal(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	s := mocks.NewMockSession(ctrl)
	var w bytes.Buffer
	s.EXPECT().Stderr().Return(&w)
	s.EXPECT().Exit(gomock.Eq(1))
	s.EXPECT().Close()

	Fatal(s, fmt.Errorf("foo"))
	requireEqual(t, "foo\n", w.String())
}
