package git

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Middleware adds Git server functionality to the ssh.Server. Repos are stored
// in the provided repoDir. If an authorizedKeys string or authorizedKeysFile
// path are provided, they will be used to authorize all pushes otherwise
// anyone can push. All repos are publicly readable.
func Middleware(repoDir, authorizedKeys, authorizedKeysFile string) wish.Middleware {
	var err error
	var ak1, ak2 []ssh.PublicKey
	if authorizedKeys != "" {
		ak1, err = parseKeysFromString(strings.Trim(authorizedKeys, "\n"))
		if err != nil {
			log.Fatal(err)
		}
	}
	if authorizedKeysFile != "" {
		ak2, err = parseKeysFromFile(authorizedKeysFile)
		if err != nil {
			log.Fatal(err)
		}
	}
	authedKeys := append(ak1, ak2...)
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			cmd := s.Command()
			if len(cmd) == 2 {
				switch cmd[0] {
				case "git-upload-pack", "git-upload-archive", "git-receive-pack":
					if len(authedKeys) > 0 && cmd[0] == "git-receive-pack" {
						authed := false
						for _, pk := range authedKeys {
							if ssh.KeysEqual(pk, s.PublicKey()) {
								authed = true
							}
						}
						if !authed {
							fatalGit(s, fmt.Errorf("you are not authorized to do this"))
							break
						}
					}
					r := cmd[1]
					rp := fmt.Sprintf("%s%s", repoDir, r)
					ctx := s.Context()
					err := ensureRepo(ctx, repoDir, r)
					if err != nil {
						fatalGit(s, err)
						break
					}
					err = runCmd(s, "./", cmd[0], rp)
					if err != nil {
						fatalGit(s, err)
						break
					}
					err = runCmd(s, rp, "git", "update-server-info")
					if err != nil {
						fatalGit(s, err)
						break
					}
					if cmd[0] == "git-receive-pack" {
						err = ensureDefaultBranch(s, rp)
						if err != nil {
							fatalGit(s, err)
							break
						}
					}
				}
			}
			sh(s)
		}
	}
}

// MiddlewareWithKeys will create Middleware with the provided authorizedKeys.
// The authorizedKeys string content should be of the same format as an ssh
// authorized_keys file.
func MiddlewareWithKeys(repoDir, authorizedKeys string) wish.Middleware {
	return Middleware(repoDir, authorizedKeys, "")
}

// MiddlewareWithKeyPath will create Middleware with the specified
// authorized_keys file.
func MiddlewareWithKeyPath(repoDir, authorizedKeysFile string) wish.Middleware {
	return Middleware(repoDir, "", authorizedKeysFile)
}

func parseKeysFromFile(path string) ([]ssh.PublicKey, error) {
	authedKeys := make([]ssh.PublicKey, 0)
	hasAuth, err := fileExists(path)
	if err != nil {
		return nil, err
	}
	if hasAuth {
		f, err := os.Open(path)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		err = addKeys(scanner, &authedKeys)
		if err != nil {
			return nil, err
		}
	}
	return authedKeys, nil
}

func parseKeysFromString(keys string) ([]ssh.PublicKey, error) {
	authedKeys := make([]ssh.PublicKey, 0)
	scanner := bufio.NewScanner(strings.NewReader(keys))
	err := addKeys(scanner, &authedKeys)
	if err != nil {
		return nil, err
	}
	return authedKeys, nil
}

func addKeys(s *bufio.Scanner, keys *[]ssh.PublicKey) error {
	for s.Scan() {
		pt := s.Text()
		if pt == "" {
			continue
		}
		log.Printf("Adding authorized key: %s", pt)
		pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pt))
		if err != nil {
			return err
		}
		*keys = append(*keys, pk)
	}
	if err := s.Err(); err != nil {
		return err
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
	s.Exit(1)
}

func ensureRepo(ctx context.Context, dir string, repo string) error {
	exists, err := fileExists(dir)
	if err != nil {
		return err
	}
	if !exists {
		err = os.MkdirAll(dir, os.ModeDir|os.FileMode(0700))
		if err != nil {
			return err
		}
	}
	rp := fmt.Sprintf("%s%s", dir, repo)
	exists, err = fileExists(rp)
	if err != nil {
		return err
	}
	if !exists {
		c := exec.CommandContext(ctx, "git", "init", "--bare", rp)
		err = c.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func runCmd(s ssh.Session, dir, name string, args ...string) error {
	usi := exec.CommandContext(s.Context(), name, args...)
	usi.Dir = dir
	usi.Stdout = s
	usi.Stdin = s
	err := usi.Run()
	if err != nil {
		return err
	}
	return nil
}

func ensureDefaultBranch(s ssh.Session, repoPath string) error {
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
	if err == plumbing.ErrReferenceNotFound {
		err = runCmd(s, repoPath, "git", "branch", "-M", fb.Name().Short())
		if err != nil {
			return err
		}
	}
	if err != nil && err != plumbing.ErrReferenceNotFound {
		return err
	}
	return nil
}
