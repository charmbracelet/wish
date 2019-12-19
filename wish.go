package wish

import (
	"fmt"
	"log"
	"os"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

type Server struct {
	router *Router
	server *ssh.Server
	port   int
}

type Router struct {
	routes       map[string]SessionHandler
	defaultRoute SessionHandler
}

func NewServer(port int) *Server {
	s := &Server{}
	routes := make(map[string]SessionHandler)
	s.router = &Router{
		routes: routes,
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Can't read user home: %s\n", err)
	}
	kf := ssh.HostKeyFile(fmt.Sprintf("%s/.ssh/id_rsa", home))
	s.server = &ssh.Server{
		Version:          "OpenSSH_7.6p1",
		Addr:             fmt.Sprintf(":%d", port),
		Handler:          s.sessionHandler,
		PublicKeyHandler: s.authHandler,
		// PasswordHandler:      s.passHandler,
		// ServerConfigCallback: s.serverConfigCallback,
	}
	s.server.SetOption(kf)
	return s
}

func (me *Server) AddHandler(route string, h SessionHandler) {
	if len(me.router.routes) == 0 {
		me.router.defaultRoute = h
	}
	me.router.routes[route] = h
}

func (me *Server) Start() error {
	if len(me.router.routes) == 0 {
		return fmt.Errorf("no routes specified")
	}
	log.Printf("starting server on %s", me.server.Addr)
	return me.server.ListenAndServe()
}

func (me *Server) sessionHandler(s ssh.Session) {
	// s.Write([]byte("\x1b[2J\x1b[1;1H")) // TODO middleware
	var route string
	cmds := s.Command()
	if len(cmds) > 0 {
		route = cmds[0]
	}
	me.router.Route(route, Session{s})
}

func (me *Server) authHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	return true
}

func (me *Server) passHandler(ctx ssh.Context, pass string) bool {
	return false
}

func (me *Server) bannerCallback(cm gossh.ConnMetadata) string {
	return fmt.Sprintf("\nHello %s put whatever you want as a password. It's no big whoop!\n\n", cm.User())
}

func (me *Server) serverConfigCallback(ctx ssh.Context) *gossh.ServerConfig {
	return &gossh.ServerConfig{
		BannerCallback: me.bannerCallback,
	}
}

func (r *Router) Route(route string, s Session) {
	h, ok := r.routes[route]
	if !ok {
		r.defaultRoute(s)
		return
	}
	h(s)
}
