package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/internal/repos"
)

func handleWebhook(next http.Handler, sigCh chan<- struct{}) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/webhook" {
			if req.Method != http.MethodPost {
				res.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			if req.Header.Get("X-GitHub-Event") != "push" {
				http.Error(res, "Only push events are accepted by this bot", http.StatusBadRequest)
				return
			}

			err := verifySecure(req)
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}

			sigCh <- struct{}{}
			return
		}

		next.ServeHTTP(res, req)
	})
}

func verifySecure(req *http.Request) error {
	sigStr := req.Header.Get("X-Hub-Signature-256")
	sig, err := hex.DecodeString(strings.TrimPrefix(sigStr, "sha256="))
	if err != nil {
		return err
	}

	secretStr, ok := os.LookupEnv("LURE_API_GITHUB_SECRET")
	if !ok {
		return errors.New("LURE_API_GITHUB_SECRET must be set to the secret used for setting up the github webhook")
	}
	secret := []byte(secretStr)

	h := hmac.New(sha256.New, secret)
	_, err = io.Copy(h, req.Body)
	if err != nil {
		return err
	}

	if !hmac.Equal(h.Sum(nil), sig) {
		log.Warn("Insecure webhook request").
			Str("from", req.RemoteAddr).
			Bytes("sig", sig).
			Bytes("hmac", h.Sum(nil)).
			Send()

		return errors.New("webhook signature mismatch")
	}

	return nil
}

func repoPullWorker(ctx context.Context, sigCh <-chan struct{}) {
	for {
		select {
		case <-sigCh:
			err := repos.Pull(ctx, gdb, cfg.Repos)
			if err != nil {
				log.Warn("Error while pulling repositories").Err(err).Send()
			}
		case <-ctx.Done():
			return
		}
	}
}
