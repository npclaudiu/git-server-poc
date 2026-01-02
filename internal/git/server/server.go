package server

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
	"github.com/npclaudiu/git-server-poc/internal/git/storage"
	"github.com/npclaudiu/git-server-poc/internal/metastore"
	"github.com/npclaudiu/git-server-poc/internal/objectstore"
)

type GitHandler struct {
	ms *metastore.MetaStore
	os *objectstore.ObjectStore
}

func New(ms *metastore.MetaStore, os *objectstore.ObjectStore) *GitHandler {
	return &GitHandler{ms: ms, os: os}
}

type repoLoader struct {
	storer storer.Storer
}

func (l *repoLoader) Load(_ *transport.Endpoint) (storer.Storer, error) {
	return l.storer, nil
}

// InfoRefs handles GET /repositories/:id/info/refs
func (h *GitHandler) InfoRefs(w http.ResponseWriter, r *http.Request, repoName string) {
	service := r.URL.Query().Get("service")
	if service != "git-upload-pack" && service != "git-receive-pack" {
		http.Error(w, "service parameter required", http.StatusBadRequest)
		return
	}

	storer := storage.NewStorer(h.os, h.ms, repoName)
	srv := server.NewServer(&repoLoader{storer: storer})
	ep, _ := transport.NewEndpoint("/")

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-advertisement", service))
	w.Header().Set("Cache-Control", "no-cache")

	enc := pktline.NewEncoder(w)
	if err := enc.Encodef("# service=%s\n", service); err != nil {
		slog.Error("failed to encode service header", "err", err)
		return
	}
	if err := enc.Flush(); err != nil {
		slog.Error("failed to flush", "err", err)
		return
	}

	if service == "git-upload-pack" {
		sess, err := srv.NewUploadPackSession(ep, nil)
		if err != nil {
			slog.Error("failed to create upload pack session", "err", err)
			return
		}
		defer sess.Close()

		ar, err := sess.AdvertisedReferences()
		if err != nil {
			slog.Error("failed to get advertised refs", "err", err)
			return
		}
		if err := ar.Encode(w); err != nil {
			slog.Error("failed to encode refs", "err", err)
		}
	} else {
		sess, err := srv.NewReceivePackSession(ep, nil)
		if err != nil {
			slog.Error("failed to create receive pack session", "err", err)
			return
		}
		defer sess.Close()

		ar, err := sess.AdvertisedReferences()
		if err != nil {
			slog.Error("failed to get advertised refs", "err", err)
			return
		}
		if err := ar.Encode(w); err != nil {
			slog.Error("failed to encode refs", "err", err)
		}
	}
}

// UploadPack handles POST /repositories/:id/git-upload-pack
func (h *GitHandler) UploadPack(w http.ResponseWriter, r *http.Request, repoName string) {
	storer := storage.NewStorer(h.os, h.ms, repoName)
	srv := server.NewServer(&repoLoader{storer: storer})
	ep, _ := transport.NewEndpoint("/")

	sess, err := srv.NewUploadPackSession(ep, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sess.Close()

	req := packp.NewUploadPackRequest()
	if err := req.Decode(r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
	w.Header().Set("Cache-Control", "no-cache")

	resp, err := sess.UploadPack(r.Context(), req)
	if err != nil {
		slog.Error("upload pack failed", "err", err)
		// We might need to write error into response stream if session started?
		return
	}

	if err := resp.Encode(w); err != nil {
		slog.Error("failed to encode upload pack response", "err", err)
	}
}

// ReceivePack handles POST /repositories/:id/git-receive-pack
func (h *GitHandler) ReceivePack(w http.ResponseWriter, r *http.Request, repoName string) {
	storer := storage.NewStorer(h.os, h.ms, repoName)
	srv := server.NewServer(&repoLoader{storer: storer})
	ep, _ := transport.NewEndpoint("/")

	sess, err := srv.NewReceivePackSession(ep, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer sess.Close()

	// Read entire body to avoid buffering issues with mixed pktline/packfile content
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	r.Body.Close()

	// Parse pktlines manually to find the split point between commands and packfile
	// this is necessary because packp.Decode uses bufio which over-reads into the packfile data.

	offset := 0
	for offset < len(bodyBytes) {
		if len(bodyBytes)-offset < 4 {
			break // invalid?
		}

		lenStr := string(bodyBytes[offset : offset+4])
		if lenStr == "0000" {
			offset += 4
			break
		}

		length, err := strconv.ParseInt(lenStr, 16, 64)
		if err != nil {
			http.Error(w, "invalid pktline length", http.StatusBadRequest)
			return
		}
		if length == 0 {
			// should likely be "0000" handled above, but just in case
			offset += 4
			break
		}
		if length < 4 {
			// invalid
			http.Error(w, "invalid pktline length", http.StatusBadRequest)
			return
		}

		offset += int(length)
	}

	// Decode commands from the command part
	cmdbuf := bytes.NewReader(bodyBytes[:offset])
	req := packp.NewReferenceUpdateRequest() // Correctly using ReferenceUpdateRequest for ReceivePack
	if err := req.Decode(cmdbuf); err != nil {
		slog.Error("decode reference update request failed", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slog.Info("manual split", "offset", offset, "total", len(bodyBytes))
	if len(bodyBytes) > offset+4 {
		slog.Info("packfile peek", "signature", string(bodyBytes[offset:offset+4]))
	}

	// The rest is the packfile
	req.Packfile = io.NopCloser(bytes.NewReader(bodyBytes[offset:]))

	w.Header().Set("Content-Type", "application/x-git-receive-pack-result")
	w.Header().Set("Cache-Control", "no-cache")

	resp, err := sess.ReceivePack(r.Context(), req)
	if err != nil {
		slog.Error("receive pack failed", "err", err)
		return
	}

	if err := resp.Encode(w); err != nil {
		slog.Error("failed to encode receive pack response", "err", err)
	}
}
