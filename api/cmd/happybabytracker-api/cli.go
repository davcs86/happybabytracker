package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

// Replaced in tests to prove a refused password never reaches the KDF.
var hashAdminPassword = auth.HashAdminPassword

const usageMsg = "usage: happybabytracker-api admin create|reset-password|unlink-sso <username>; create and reset-password read the password from stdin"

var errPasswordLength = fmt.Errorf("the password must be %d to %d characters", constants.MinAdminPasswordLen, constants.MaxPasswordLen)

// runAdmin reads only DATA_DIR; its output never carries the username or the password.
func runAdmin(ctx context.Context, args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer, now time.Time) int {
	out := slog.New(slog.NewJSONHandler(stdout, nil))
	errLog := slog.New(slog.NewJSONHandler(stderr, nil))
	if len(args) != 2 || !usernameRE.MatchString(args[1]) {
		errLog.Error(usageMsg, "outcome", "usage")
		return 2
	}
	verb, username := args[0], args[1]
	switch verb {
	case "create", "reset-password", "unlink-sso":
	default:
		errLog.Error(usageMsg, "outcome", "usage")
		return 2
	}
	refuse := func(msg string) int {
		errLog.Error(msg, "verb", verb, "outcome", "refused")
		return 1
	}

	dir := getenv("DATA_DIR")
	if dir == "" {
		dir = constants.DefaultDataDir
	}
	st, err := store.Open(dir)
	if err != nil {
		errLog.Error("admin command failed", "verb", verb, "outcome", "error", "error", err.Error())
		return 1
	}
	defer st.Close()

	var hash string
	if verb != "unlink-sso" {
		if hash, err = readPassword(stdin); errors.Is(err, errPasswordLength) {
			return refuse(errPasswordLength.Error())
		} else if err != nil {
			errLog.Error("admin command failed", "verb", verb, "outcome", "error", "error", err.Error())
			return 1
		}
	}

	var msg string
	switch verb {
	case "create":
		_, err = st.CreateFirstAdmin(ctx, username, hash, now)
		msg = "admin created"
	case "reset-password":
		err = st.SetAdminPassword(ctx, username, hash, now)
		msg = "admin password reset; its sessions were ended"
	case "unlink-sso":
		err = st.UnlinkAdminSSO(ctx, username)
		msg = "admin single sign-on unlinked; its sessions were ended"
	}
	switch {
	case errors.Is(err, store.ErrAdminExists):
		return refuse("an admin already exists")
	case errors.Is(err, store.ErrConflict):
		return refuse("that username belongs to a parent")
	case errors.Is(err, store.ErrNotFound):
		return refuse("no admin with that username")
	case err != nil:
		errLog.Error("admin command failed", "verb", verb, "outcome", "error", "error", err.Error())
		return 1
	}
	out.Info(msg, "verb", verb, "outcome", "ok")
	return 0
}

// readPassword strips exactly one trailing "\n" or "\r\n"; over CLIPasswordMaxBytes is refused unhashed.
func readPassword(stdin io.Reader) (string, error) {
	b, err := io.ReadAll(io.LimitReader(stdin, constants.CLIPasswordMaxBytes+1))
	if err != nil {
		return "", err
	}
	if len(b) > constants.CLIPasswordMaxBytes {
		return "", errPasswordLength
	}
	if p, ok := bytes.CutSuffix(b, []byte("\r\n")); ok {
		b = p
	} else {
		b = bytes.TrimSuffix(b, []byte("\n"))
	}
	hash, err := hashAdminPassword(string(b))
	if errors.Is(err, auth.ErrPasswordLength) {
		return "", errPasswordLength
	}
	return hash, err
}
