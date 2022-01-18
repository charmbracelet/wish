package git

import (
	"fmt"
	"io"
	"log"

	"github.com/gliderlabs/ssh"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
)

type gitServer struct {
	transport.Transport
}

func newServer(path string) gitServer {
	return gitServer{
		server.DefaultServer, //NewServer(server.NewFilesystemLoader(osfs.New(path))),
	}
}

func (src gitServer) uploadPack(ssess ssh.Session, path string) error {
	ep, err := transport.NewEndpoint(path)
	if err != nil {
		return err
	}

	// TODO: define and implement a server-side AuthMethod
	s, err := server.DefaultServer.NewUploadPackSession(ep, nil)
	if err != nil {
		return fmt.Errorf("error creating session: %w", err)
	}

	ar, err := s.AdvertisedReferences()
	if err != nil {
		return fmt.Errorf("internal error in advertised references: %w", err)
	}

	if err := ar.Encode(ssess); err != nil {
		return fmt.Errorf("error in advertised references encoding: %w", err)
	}

	req := packp.NewUploadPackRequest()
	if err := req.Decode(ssess); err != nil && err != io.EOF {
		return fmt.Errorf("error decoding: %w", err)
	}

	var resp *packp.UploadPackResponse
	resp, err = s.UploadPack(ssess.Context(), req)
	if err != nil {
		return fmt.Errorf("error in upload pack: %w", err)
	}

	if err := resp.Encode(ssess); err != nil {
		return fmt.Errorf("error in encoding report status %w", err)
	}
	return nil
}

func (srv gitServer) receivePack(ssess ssh.Session, path string) error {
	ep, err := transport.NewEndpoint(path)
	if err != nil {
		return err
	}

	s, err := srv.NewReceivePackSession(ep, nil)
	if err != nil {
		return fmt.Errorf("error creating session: %w", err)
	}
	ar, err := s.AdvertisedReferences()
	if err != nil {
		return fmt.Errorf("internal error in advertised references: %w", err)
	}

	if err := ar.Encode(ssess); err != nil {
		return fmt.Errorf("error in advertised references encoding: %w", err)
	}

	req := packp.NewReferenceUpdateRequest()
	if err := req.Decode(io.NopCloser(ssess)); err != nil {
		return fmt.Errorf("error decoding: %w", err)
	}

	rs, err := s.ReceivePack(ssess.Context(), req)
	if rs != nil {
		if err := rs.Encode(ssess); err != nil && err != io.EOF {
			return fmt.Errorf("error in encoding report status %w", err)
		}
	}

	if err != nil {
		return fmt.Errorf("error in receive pack: %w", err)
	}

	if err := srv.ensureDefaultBranch(ssess, path); err != nil {
		log.Println("failed to ensure default branch", err)
		return err
	}
	return nil
}

func (srv gitServer) ensureDefaultBranch(s ssh.Session, repoPath string) error {
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
