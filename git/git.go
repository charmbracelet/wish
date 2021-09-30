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
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
	"github.com/go-git/go-git/v5/utils/ioutil"
)

func serveUploadPack(s ssh.Session, ups transport.UploadPackSession) (err error) {
	adv, err := ups.AdvertisedReferences()
	if err != nil {
		log.Print("adv ref")
		log.Print(err)
		return err
	}
	err = adv.Encode(s)
	if err != nil {
		log.Print("encode adv")
		log.Print(err)
		return err
	}
	r := packp.NewUploadPackRequest()
	if err = r.Decode(s); err != nil {
		log.Print("decode req")
		log.Print(err)
		return err
	}
	res, err := ups.UploadPack(s.Context(), r)
	if err != nil {
		log.Print("upload")
		log.Print(err)
		return err
	}
	err = res.Encode(s)
	if err != nil {
		log.Print("encode res")
		log.Print(err)
		return err
	}
	log.Printf("%+v", res)
	return err
}

func serveReceivePack(s ssh.Session, rps transport.ReceivePackSession) error {
	in := s
	out := ioutil.WriteNopCloser(s)
	adv, err := rps.AdvertisedReferencesContext(s.Context())
	if err != nil {
		log.Print("adv ref")
		log.Print(err)
		return err
	}
	err = adv.Encode(out)
	if err != nil {
		log.Print("encode adv")
		log.Print(err)
		return err
	}
	r := packp.NewReferenceUpdateRequest()
	if err = r.Decode(in); err != nil {
		log.Print("decode req")
		log.Print(err)
		return err
	}
	res, err := rps.ReceivePack(s.Context(), r)
	if res != nil {
		for _, c := range res.CommandStatuses {
			log.Printf("%+v", c)
		}
		if err := res.Encode(out); err != nil {
			log.Print("encode res")
			log.Print(err)
			return err
		}
	}
	if err != nil {
		log.Print("receive")
		log.Print(err)
		return err
	}
	return nil
}

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
				authed := len(authedKeys) == 0
				if len(authedKeys) > 0 {
					for _, pk := range authedKeys {
						if ssh.KeysEqual(pk, s.PublicKey()) {
							authed = true
						}
					}
				}
				log.Printf("authed: %v", authed)
				method := cmd[0]
				repo := cmd[1]
				rp := fmt.Sprintf("%s%s", repoDir, repo)
				loader := server.NewFilesystemLoader(osfs.New(rp))
				srv := server.NewServer(loader)
				ep, err := transport.NewEndpoint(fmt.Sprintf("ssh://%s", s.LocalAddr().String()))
				if err != nil {
					log.Print(err)
					fatalGit(s, err)
					goto OUT
				}

				switch method {
				case transport.UploadPackServiceName:
					ups, err := srv.NewUploadPackSession(ep, nil)
					if err != nil {
						log.Print("upload pack sess")
						log.Print(err)
						goto OUT
					}
					err = serveUploadPack(s, ups)
					if err != nil {
						log.Print("serve up pack")
						log.Print(err)
						goto OUT
					}
				case transport.ReceivePackServiceName:
					if authed && !repoExists(rp) {
						err = initRepo(s.Context(), rp)
						if err != nil {
							log.Print("init repo")
							log.Print(err)
							goto OUT
						}
					}
					rps, err := srv.NewReceivePackSession(ep, nil)
					if err != nil {
						log.Print("receive pack sess")
						log.Print(err)
						goto OUT
					}
					err = serveReceivePack(s, rps)
					if err != nil {
						log.Print("serve receive pack")
						log.Print(err)
						goto OUT
					}
				}
			}
		OUT:
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

func repoExists(path string) bool {
	_, err := git.PlainOpen(path)
	return err == nil
}

func initRepo(ctx context.Context, repoPath string) error {
	_, err := git.PlainInit(repoPath, true)
	if err != nil {
		return err
	}
	return nil
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
