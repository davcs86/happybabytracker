package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/config"
	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

var errAdminCollision = errors.New("ADMIN_USERNAME collides with an existing parent account")

var usernameRE = regexp.MustCompile(constants.UsernamePattern)

// importAdmin copies the env admin only while no admin is stored; errors name the variable, never its value.
func importAdmin(ctx context.Context, st *store.Store, cfg config.Config, log *slog.Logger, now time.Time) error {
	ignored := func() error {
		if cfg.AdminUsername != "" || cfg.AdminPassword != "" {
			log.Warn("admin credentials in the environment are ignored")
		}
		return nil
	}
	ok, err := st.HasAdmin(ctx)
	if err != nil {
		return err
	}
	if ok {
		return ignored()
	}
	if !usernameRE.MatchString(cfg.AdminUsername) {
		return fmt.Errorf("ADMIN_USERNAME must match %s", constants.UsernamePattern)
	}
	hash, err := auth.HashAdminPassword(cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("ADMIN_PASSWORD must be %d to %d characters", constants.MinAdminPasswordLen, constants.MaxPasswordLen)
	}
	_, err = st.CreateFirstAdmin(ctx, cfg.AdminUsername, hash, now)
	switch {
	case errors.Is(err, store.ErrConflict):
		return errAdminCollision
	case errors.Is(err, store.ErrAdminExists):
		return ignored()
	}
	return err
}
