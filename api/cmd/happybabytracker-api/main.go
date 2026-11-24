// Command happybabytracker-api serves the baby tracker's HTTP API over SQLite.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/config"
	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/growth"
	"github.com/davcs86/happybabytracker/api/internal/httpapi"
	"github.com/davcs86/happybabytracker/api/internal/ntfy"
	"github.com/davcs86/happybabytracker/api/internal/push"
	"github.com/davcs86/happybabytracker/api/internal/reminders"
	"github.com/davcs86/happybabytracker/api/internal/store"
	"github.com/davcs86/happybabytracker/api/internal/vaccines"
)

const fingerprintKey = "admin_fingerprint"

var errAdminCollision = errors.New("ADMIN_USERNAME collides with an existing parent account")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	// Restoring default handling after the first signal lets a second one kill the process.
	go func() { <-ctx.Done(); stop() }()
	if err := run(ctx, os.Getenv, os.Stdout); err != nil {
		slog.New(slog.NewJSONHandler(os.Stdout, nil)).Error("happybabytracker-api failed", "error", err.Error())
		os.Exit(1)
	}
}

// app is what start prepares; the caller runs worker and closes store.
type app struct {
	cfg     config.Config
	log     *slog.Logger
	store   *store.Store
	handler http.Handler
	worker  *reminders.Worker
}

// start does everything before listening, so tests can call it without binding cfg.Addr.
func start(ctx context.Context, getenv func(string) string, stdout io.Writer) (*app, error) {
	cfg, err := config.Load(getenv)
	if err != nil {
		return nil, err
	}
	log := slog.New(slog.NewJSONHandler(stdout, nil))
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	if err := prepare(ctx, st, cfg); err != nil {
		return nil, errors.Join(err, st.Close())
	}
	pushState := "off"
	if cfg.VAPIDKey != nil {
		pushState = "on"
	}
	ntfyState := "off"
	if cfg.NtfyEnabled {
		ntfyState = "on"
	}
	log.Info("channels", "push", pushState, "ntfy", ntfyState)

	deps := httpapi.Deps{
		Store: st, Cfg: cfg, Limiter: auth.NewLimiter(constants.LoginAttemptsPerMinute, time.Now), Now: time.Now, Log: log,
		Resolver: net.DefaultResolver,
	}
	if err := wirePush(&deps, cfg); err != nil {
		return nil, errors.Join(err, st.Close())
	}
	var publisher ntfy.Publisher
	if cfg.NtfyEnabled {
		publisher = ntfy.NewPublisher(cfg.Ntfy)
	}
	// Before the first tick: a row left sending may have been delivered, so it is never resent.
	if err := st.MarkUnconfirmed(ctx); err != nil {
		return nil, errors.Join(err, st.Close())
	}
	worker := reminders.New(reminders.Deps{Store: st, Push: deps.Push, Ntfy: publisher, Origin: cfg.PublicOrigin, Now: time.Now, Log: log})
	deps.LastTick = worker.LastTick
	return &app{cfg: cfg, log: log, store: st, handler: httpapi.NewHandler(deps), worker: worker}, nil
}

// wirePush sets the sender and the public key from the one VAPID key, so the key is served only with a sender.
func wirePush(d *httpapi.Deps, cfg config.Config) error {
	if cfg.VAPIDKey == nil {
		return nil
	}
	if cfg.AllowFakePush {
		d.PushRecorder = &push.Recorder{}
		d.Push = d.PushRecorder
	} else {
		sig, err := push.NewSigner(cfg.VAPIDKey, cfg.VAPIDSubject)
		if err != nil {
			return err
		}
		d.Push = push.NewSender(sig, time.Now)
	}
	d.VAPIDPublicKey = push.PublicKey(cfg.VAPIDKey)
	return nil
}

func prepare(ctx context.Context, st *store.Store, cfg config.Config) error {
	if err := checkAdminUsername(ctx, st, cfg); err != nil {
		return err
	}
	if err := syncAdminFingerprint(ctx, st, cfg); err != nil {
		return err
	}
	if err := growth.Load(); err != nil {
		return err
	}
	return vaccines.Load()
}

func run(ctx context.Context, getenv func(string) string, stdout io.Writer) error {
	a, err := start(ctx, getenv, stdout)
	if err != nil {
		return err
	}
	defer a.store.Close()
	cfg, log, handler := a.cfg, a.log, a.handler

	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.Addr, err)
	}
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: constants.ReadHeaderTimeout,
		ReadTimeout:       constants.ReadTimeout,
		WriteTimeout:      constants.WriteTimeout,
	}
	log.Info("listening", "addr", ln.Addr().String())

	wctx, stopWorker := context.WithCancel(ctx)
	workerDone := make(chan struct{})
	go func() { a.worker.Run(wctx); close(workerDone) }()
	// The store closes after the worker has stopped.
	defer func() { stopWorker(); <-workerDone }()

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}
	sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), constants.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		log.Warn("http shutdown timed out; closing open connections", "timeout_ms", constants.ShutdownTimeout.Milliseconds())
		return errors.Join(err, srv.Close())
	}
	return nil
}

// syncAdminFingerprint ends every admin session when the admin credentials differ from the last start.
func syncAdminFingerprint(ctx context.Context, st *store.Store, cfg config.Config) error {
	want := auth.AdminFingerprint(cfg.InternalAPIToken, cfg.AdminUsername, cfg.AdminPassword)
	have, ok, err := st.MetaGet(ctx, fingerprintKey)
	if err != nil {
		return err
	}
	if ok && have == want {
		return nil
	}
	if err := st.DeleteAdminSessions(ctx); err != nil {
		return err
	}
	return st.MetaSet(ctx, fingerprintKey, want)
}

// checkAdminUsername refuses to start when a parent already owns the admin's login name; the error omits the name.
func checkAdminUsername(ctx context.Context, st *store.Store, cfg config.Config) error {
	name, ok := auth.NormalizeUsername(cfg.AdminUsername)
	if !ok {
		return errors.New("ADMIN_USERNAME is not a valid username")
	}
	_, _, err := st.ParentByUsername(ctx, name)
	if err == nil {
		return errAdminCollision
	}
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	return err
}
