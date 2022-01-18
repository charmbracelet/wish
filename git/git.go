package git

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
)

// ErrNotAuthed represents unauthorized access.
var ErrNotAuthed = fmt.Errorf("you are not authorized to do this")

// ErrSystemMalfunction represents a general system error returned to clients.
var ErrSystemMalfunction = fmt.Errorf("something went wrong")

// ErrInvalidRepo represents an attempt to access a non-existent repo.
var ErrInvalidRepo = fmt.Errorf("invalid repo")

// AccessLevel is the level of access allowed to a repo.
type AccessLevel int

const (
	// NoAccess does not allow access to the repo.
	NoAccess AccessLevel = iota

	// ReadOnlyAccess allows read-only access to the repo.
	ReadOnlyAccess

	// ReadWriteAccess allows read and write access to the repo.
	ReadWriteAccess

	// AdminAccess allows read, write, and admin access to the repo.
	AdminAccess
)

// GitHooks is an interface that allows for custom authorization
// implementations and post push/fetch notifications. Prior to git access,
// AuthRepo will be called with the ssh.Session public key and the repo name.
// Implementers return the appropriate AccessLevel.
//
// Deprecated: use Hooks instead.
type GitHooks = Hooks // nolint: revive

// Hooks is an interface that allows for custom authorization
// implementations and post push/fetch notifications. Prior to git access,
// AuthRepo will be called with the ssh.Session public key and the repo name.
// Implementers return the appropriate AccessLevel.
type Hooks interface {
	AuthRepo(string, ssh.PublicKey) AccessLevel
	Push(string, ssh.PublicKey)
	Fetch(string, ssh.PublicKey)
}

// Middleware adds Git server functionality to the ssh.Server. Repos are stored
// in the specified repo directory. The provided Hooks implementation will be
// checked for access on a per repo basis for a ssh.Session public key.
// Hooks.Push and Hooks.Fetch will be called on successful completion of
// their commands.
func Middleware(repoDir string, gh Hooks) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			cmd := s.Command()
			if len(cmd) == 2 {
				gc := cmd[0]
				repo := cmd[1] // cmd[1] should be `/REPO`
				repo = filepath.Clean(repo)
				repo = filepath.Base(repo)
				pk := s.PublicKey()
				access := gh.AuthRepo(repo, pk)
				switch gc {
				case transport.ReceivePackServiceName:
					switch access {
					case ReadWriteAccess, AdminAccess:
						if err := ensureRepo(repoDir, repo); err != nil {
							fatalGit(s, ErrSystemMalfunction)
						}
						if err := gitReceivePack(s, filepath.Join(repoDir, repo)); err != nil {
							fatalGit(s, ErrSystemMalfunction)
						}
						gh.Push(repo, pk)
					default:
						fatalGit(s, ErrNotAuthed)
					}
				case transport.UploadPackServiceName:
					switch access {
					case ReadOnlyAccess, ReadWriteAccess, AdminAccess:
						err := gitUploadPack(s, filepath.Join(repoDir, repo))
						if err == nil {
							gh.Fetch(repo, pk)
						} else if errors.Is(err, transport.ErrRepositoryNotFound) {
							fatalGit(s, ErrInvalidRepo)
						} else {
							fatalGit(s, ErrSystemMalfunction)
						}
					default:
						fatalGit(s, ErrNotAuthed)
					}
				}
			}
			sh(s)
		}
	}
}

func gitReceivePack(sess ssh.Session, path string) error {
	ep, err := transport.NewEndpoint(path)
	if err != nil {
		return err
	}

	s, err := server.DefaultServer.NewReceivePackSession(ep, nil)
	if err != nil {
		return fmt.Errorf("error creating session: %w", err)
	}
	ar, err := s.AdvertisedReferences()
	if err != nil {
		return fmt.Errorf("internal error in advertised references: %w", err)
	}

	if err := ar.Encode(sess); err != nil {
		return fmt.Errorf("error in advertised references encoding: %w", err)
	}

	req := packp.NewReferenceUpdateRequest()
	if err := req.Decode(io.NopCloser(sess)); err != nil {
		return fmt.Errorf("error decoding: %w", err)
	}

	rs, err := s.ReceivePack(sess.Context(), req)
	if rs != nil {
		if err := rs.Encode(sess); err != nil && err != io.EOF {
			return fmt.Errorf("error in encoding report status %w", err)
		}
	}

	if err != nil {
		return fmt.Errorf("error in receive pack: %w", err)
	}

	if err := ensureDefaultBranch(path); err != nil {
		log.Println("failed to ensure default branch", err)
		return err
	}
	return nil
}

func gitUploadPack(sess ssh.Session, path string) error {
	ep, err := transport.NewEndpoint(path)
	if err != nil {
		return err
	}

	s, err := server.DefaultServer.NewUploadPackSession(ep, nil)
	if err != nil {
		return fmt.Errorf("error creating session: %w", err)
	}

	ar, err := s.AdvertisedReferences()
	if err != nil {
		return fmt.Errorf("internal error in advertised references: %w", err)
	}

	if err := ar.Encode(sess); err != nil {
		return fmt.Errorf("error in advertised references encoding: %w", err)
	}

	req := packp.NewUploadPackRequest()
	if err := req.Decode(sess); err != nil && err != io.EOF {
		return fmt.Errorf("error decoding: %w", err)
	}

	var resp *packp.UploadPackResponse
	resp, err = s.UploadPack(sess.Context(), req)
	if err != nil {
		return fmt.Errorf("error in upload pack: %w", err)
	}

	if err := resp.Encode(sess); err != nil {
		return fmt.Errorf("error in encoding report status %w", err)
	}
	return nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func fatalGit(s ssh.Session, err error) {
	// hex length includes 4 byte length prefix and ending newline
	msg := err.Error()
	pktLine := fmt.Sprintf("%04x%s\n", len(msg)+5, msg)
	_, _ = s.Write([]byte(pktLine))
	s.Exit(1) // nolint: errcheck
}

func ensureRepo(dir string, repo string) error {
	exists, err := fileExists(dir)
	if err != nil {
		return err
	}
	if !exists {
		err = os.MkdirAll(dir, os.ModeDir|os.FileMode(0o700))
		if err != nil {
			return err
		}
	}
	rp := filepath.Join(dir, repo)
	exists, err = fileExists(rp)
	if err != nil {
		return err
	}
	if !exists {
		_, err := git.PlainInit(rp, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func ensureDefaultBranch(repoPath string) error {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}
	brs, err := r.Branches()
	if err != nil {
		return err
	}
	defer brs.Close()
	fb, err := brs.Next()
	if err != nil {
		return err
	}
	// Rename the default branch to the first branch available
	_, err = r.Head()
	if err == nil {
		return nil
	}
	if err == plumbing.ErrReferenceNotFound {
		if err := r.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, fb.Name())); err != nil {
			return err
		}
		return nil
	}
	return err
}
