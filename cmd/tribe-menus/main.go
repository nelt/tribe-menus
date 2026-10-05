// Command tribe-menus runs the Melting Tribe application.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nelt/tribe-menus/internal/admin"
	"github.com/nelt/tribe-menus/internal/mail"
	"github.com/nelt/tribe-menus/internal/server"
	"github.com/nelt/tribe-menus/internal/storage"
	"github.com/nelt/tribe-menus/internal/tribe"
	"github.com/nelt/tribe-menus/web"
)

// Set at build time with -ldflags -X (ADR 0012).
var (
	version = "dev"
	commit  = "unknown"
)

// timeouts of the HTTP server: without them, a client that stops sending, or keeps a
// connection idle, holds a goroutine and a file descriptor without limit, and makes the
// shutdown fail (plan revue-securite, finding 3). A request body is 4 KB at most, a
// response a few hundred KB.
type timeouts struct {
	readHeader, read, write, idle time.Duration
}

var defaultTimeouts = timeouts{readHeader: 10 * time.Second, read: 20 * time.Second, write: 30 * time.Second, idle: 60 * time.Second}

const (
	shutdownTimeout = 10 * time.Second
	// purgeInterval: login codes, sessions and rate limit rows are purged at startup, then
	// at this interval (PT-07). The retention stated in the specs, ADR 0021 and the privacy
	// page (an hour and ten minutes at most for the hashes of addresses and IPs) is
	// tribe.RateLimitRetention plus this interval: change them together.
	purgeInterval = 10 * time.Minute
)

const usage = `Usage: tribe-menus <command> [flags]

Commands:
  serve        run the HTTP server
  admin init   create a tribe, interactively (EF-08)
  admin seed   create the demonstration tribe "demo", if missing
  version      print the version
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run executes the command line and returns the process exit code.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "version":
		fmt.Fprintf(stdout, "tribe-menus %s (%s)\n", version, commit)
		return 0
	case "serve":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return serve(ctx, args[1:], stderr)
	case "admin":
		// Signals are not caught: reading an answer does not watch a context, and the default
		// SIGINT stops the process. Nothing is written before the last answer.
		return runAdmin(context.Background(), args[1:], stdin, stdout, stderr)
	case "help", "-h", "-help", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "tribe-menus: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

// runAdmin runs a subcommand of the admin command on the databases of the data directory.
func runAdmin(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 || (args[0] != "init" && args[0] != "seed") {
		fmt.Fprint(stderr, "Usage: tribe-menus admin init|seed [flags]\n")
		return 2
	}
	flags := flag.NewFlagSet("admin "+args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	data := flags.String("data", "data", "directory of the databases")
	baseURL := flags.String("base-url", "http://localhost:8080", "address of the instance, to show the URL of the tribe")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}

	err := withAdminCommand(ctx, *data, *baseURL, stdin, stdout, func(cmd *admin.Command) error {
		if args[0] == "seed" {
			return cmd.Seed(ctx)
		}
		return cmd.Init(ctx)
	})
	if err != nil {
		fmt.Fprintf(stderr, "tribe-menus admin %s: %v\n", args[0], err)
		return 1
	}
	return 0
}

// withAdminCommand runs do with an admin command on the databases of the data directory.
func withAdminCommand(ctx context.Context, data, baseURL string, stdin io.Reader, stdout io.Writer, do func(*admin.Command) error) (err error) {
	migrations, err := storage.EmbeddedMigrations()
	if err != nil {
		return err
	}
	store, err := storage.Open(ctx, data, migrations)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close databases: %w", closeErr)
		}
	}()
	return do(&admin.Command{Store: store, In: stdin, Out: stdout, BaseURL: baseURL, Now: time.Now})
}

// serve runs the HTTP server until ctx is done.
func serve(ctx context.Context, args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	addr := flags.String("addr", "localhost:8080", "TCP address to listen on")
	dev := flags.Bool("dev", false, "development mode: read the front end and the public site from disk")
	root := flags.String("root", ".", "repository root, used in development mode")
	data := flags.String("data", "data", "directory of the databases")
	mailFile := flags.String("mail-file", "", "development mode: also append each email to this file, one JSON object per line (end-to-end tests)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *mailFile != "" && !*dev {
		fmt.Fprintln(stderr, "tribe-menus serve: -mail-file requires -dev")
		return 2
	}

	logger := slog.New(slog.NewTextHandler(stderr, nil))
	opts := serveOptions{addr: *addr, dev: *dev, root: *root, data: *data, mailFile: *mailFile}
	if err := migrateAndServe(ctx, opts, logger); err != nil {
		logger.Error("server stopped", "error", err)
		return 1
	}
	return 0
}

// serveOptions are the flags of the serve command.
type serveOptions struct {
	addr string
	dev  bool
	// root is the repository root, read in development mode.
	root string
	data string
	// mailFile, in development mode only, receives a copy of each email (D6).
	mailFile string
}

// migrateAndServe opens and migrates every database before listening, so that the server,
// and its health check, never answer on a database left unmigrated (ADR 0003, point 6).
func migrateAndServe(ctx context.Context, opts serveOptions, logger *slog.Logger) (err error) {
	migrations, err := storage.EmbeddedMigrations()
	if err != nil {
		return err
	}
	store, err := storage.Open(ctx, opts.data, migrations)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close databases: %w", closeErr)
		}
	}()
	logger.Info("databases migrated", "data", opts.data)

	// Sending email over SMTP comes with the deployment (ADR 0014): until then, the codes
	// are written to the logs in development, and not sent otherwise.
	var mailer mail.Mailer = mail.Unconfigured{}
	if opts.dev {
		mailer = mail.LogMailer{Logger: logger}
	}
	if opts.dev && opts.mailFile != "" {
		mailer = mail.Mailers{mailer, &mail.FileMailer{Path: opts.mailFile}}
	}
	outbox := mail.NewOutbox(mailer, tribe.LoginCodeValidity, logger)
	// Deferred after store.Close, hence run before: the sendings in progress end first (D11).
	defer outbox.Wait()
	login := &tribe.Login{RateLimit: tribe.NewRateLimitDB(store.RateLimit()), Mailer: outbox, Now: time.Now}

	purgeCtx, stopPurge := context.WithCancel(ctx)
	purged := make(chan struct{})
	go func() {
		defer close(purged)
		purgePeriodically(purgeCtx, login, store, logger)
	}()
	defer func() {
		stopPurge()
		<-purged
	}()

	return listenAndServe(ctx, opts, logger, store, login)
}

// purgePeriodically purges at once, then at each interval until ctx is done.
func purgePeriodically(ctx context.Context, login *tribe.Login, dbs tribe.Databases, logger *slog.Logger) {
	ticker := time.NewTicker(purgeInterval)
	defer ticker.Stop()
	for {
		if err := login.Purge(ctx, dbs); err != nil && ctx.Err() == nil {
			logger.Error("purge failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func listenAndServe(ctx context.Context, opts serveOptions, logger *slog.Logger, store *storage.Store, login *tribe.Login) error {
	cfg, err := serverConfig(opts.dev, opts.root, logger)
	if err != nil {
		return err
	}
	cfg.Tribes, cfg.Login = store, login
	cfg.Version, cfg.Commit = version, commit
	handler, err := server.New(cfg)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", opts.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	srv := newHTTPServer(handler, slog.NewLogLogger(logger.Handler(), slog.LevelWarn), defaultTimeouts)
	logger.Info("server started", "version", version, "commit", commit, "addr", "http://"+listener.Addr().String(), "dev", opts.dev)

	served := make(chan error, 1)
	go func() { served <- srv.Serve(listener) }()

	select {
	case err := <-served:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-served; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// newHTTPServer returns the HTTP server of the handler, with its timeouts.
func newHTTPServer(handler http.Handler, errorLog *log.Logger, t timeouts) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: t.readHeader,
		ReadTimeout:       t.read,
		WriteTimeout:      t.write,
		IdleTimeout:       t.idle,
		ErrorLog:          errorLog,
	}
}

func serverConfig(dev bool, root string, logger *slog.Logger) (server.Config, error) {
	if dev {
		return server.Config{
			Web:    os.DirFS(filepath.Join(root, "web", "dist")),
			Site:   os.DirFS(filepath.Join(root, "site")),
			Dev:    true,
			Logger: logger,
		}, nil
	}
	dist, err := web.Dist()
	if err != nil {
		return server.Config{}, fmt.Errorf("embedded front end: %w", err)
	}
	return server.Config{Web: dist, Logger: logger}, nil
}
