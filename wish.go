package wish

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/charm/keygen"
	"github.com/gliderlabs/ssh"
)

type Middleware func(ssh.Handler) ssh.Handler

func NewServer(addr string, keyPath string, mw ...Middleware) (*ssh.Server, error) {
	s := &ssh.Server{}
	s.Version = "OpenSSH_7.6p1"
	s.Addr = addr
	s.PasswordHandler = passHandler
	s.PublicKeyHandler = authHandler
	kps := strings.Split(keyPath, string(filepath.Separator))
	kp := strings.Join(kps[:len(kps)-1], string(filepath.Separator))
	n := strings.TrimRight(kps[len(kps)-1], "_ed25519")
	_, err := keygen.NewSSHKeyPair(kp, n, nil, "ed25519")
	if err != nil {
		return nil, err
	}
	k := ssh.HostKeyFile(keyPath)
	err = s.SetOption(k)
	if err != nil {
		return nil, err
	}
	h := func(s ssh.Session) {}
	for _, m := range mw {
		h = m(h)
	}
	s.Handler = h
	return s, nil
}

func authHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	return true
}

func passHandler(ctx ssh.Context, pass string) bool {
	return true
}
