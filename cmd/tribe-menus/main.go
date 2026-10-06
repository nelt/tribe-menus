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
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/nelt/tribe-menus/internal/admin"
	"github.com/nelt/tribe-menus/internal/config"
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
// connection idle, holds a goroutine and a file descriptor without limit (plan
// revue-securite, finding 3). A request body is 4 KB at most, a response a few hundred
// KB. The shutdown does not wait for them to expire: see shutdownTimeout.
type timeouts struct {
	readHeader, read, write, idle time.Duration
}

var defaultTimeouts = timeouts{readHeader: 10 * time.Second, read: 20 * time.Second, write: 30 * time.Second, idle: 60 * time.Second}

// shutdownTimeout is how long the shutdown waits for the requests in progress; the
// connections still open then are closed. A variable for the tests.
var shutdownTimeout = 10 * time.Second

// embeddedWeb returns the front end embedded in the binary: a variable for the tests, run
// before the front end is built.
var embeddedWeb = web.Dist

const (
	// purgeInterval: login codes, sessions and rate limit rows are purged at startup, then
	// at this interval (PT-07). The retention stated in the specs, ADR 0021 and the privacy
	// page (an hour and ten minutes at most for the hashes of addresses and IPs) is
	// tribe.RateLimitRetention plus this interval: change them together.
	purgeInterval = 10 * time.Minute
)

const usage = `Usage: tribe-menus <command> [flags]

Commands:
  serve        run the HTTP server: -config <file> (server mode) or -dev (development mode)
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
		return serve(ctx, args[1:], stdout, stderr)
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
	configPath := flags.String("config", "", "the config file of the server mode, for its data and baseURL keys")
	data := flags.String("data", "data", "directory of the databases, without -config")
	baseURL := flags.String("base-url", "http://localhost:8080", "address of the instance, to show the URL of the tribe, without -config")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "tribe-menus admin %s: unexpected argument %q\n", args[0], flags.Arg(0))
		return 2
	}
	// The script tribe-menus-admin of the server passes the config file of the service
	// (ADR 0015, point 13).
	if *configPath != "" {
		for _, name := range []string{"data", "base-url"} {
			if flagSet(flags, name) {
				fmt.Fprintf(stderr, "tribe-menus admin %s: -%s is refused with -config\n", args[0], name)
				return 2
			}
		}
		cfg, err := config.Load(*configPath)
		if err != nil {
			fmt.Fprintf(stderr, "tribe-menus admin %s: %v\n", args[0], err)
			return 2
		}
		*data, *baseURL = cfg.Data, cfg.BaseURL
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

// devFlags are the flags of the development mode, refused in server mode (plan
// production, D2).
var devFlags = []string{"addr", "data", "root", "mail-file"}

// serve runs the HTTP server until ctx is done, in server mode (-config) or in development
// mode (-dev), never both, never neither (plan production, D2).
func serve(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "", "server mode: the config file")
	dev := flags.Bool("dev", false, "development mode: read the front end and the public site from disk")
	addr := flags.String("addr", "localhost:8080", "development mode: TCP address to listen on")
	root := flags.String("root", ".", "development mode: repository root")
	data := flags.String("data", "data", "development mode: directory of the databases")
	mailFile := flags.String("mail-file", "", "development mode: also append each email to this file, one JSON object per line (end-to-end tests)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	fail := func(msg string) int {
		fmt.Fprintf(stderr, "tribe-menus serve: %s\n", msg)
		return 2
	}
	if flags.NArg() > 0 {
		return fail("unexpected argument " + strconv.Quote(flags.Arg(0)))
	}
	switch {
	case *dev && *configPath != "":
		return fail("-dev and -config exclude each other")
	case !*dev && *configPath == "":
		return fail("want -config <file> (server mode) or -dev (development mode)")
	}

	if *dev {
		// A unit that would run serve -dev: the development mode never runs under systemd
		// (plan revue-securite, finding 8).
		if n, err := systemdSocketCount(os.Getenv, os.Getpid()); n > 0 || err != nil {
			return fail("-dev refused: a socket is passed by systemd")
		}
		logger := slog.New(slog.NewTextHandler(stderr, nil))
		opts := serveOptions{listen: *addr, dev: true, root: *root, data: *data, mailFile: *mailFile}
		return runServer(ctx, opts, logger)
	}

	for _, name := range devFlags {
		if flagSet(flags, name) {
			return fail("-" + name + " is a flag of the development mode, refused with -config")
		}
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return fail(err.Error())
	}
	logger := slog.New(slog.NewJSONHandler(stdout, nil))
	opts := serveOptions{listen: cfg.Listen, configPath: *configPath, data: cfg.Data}
	return runServer(ctx, opts, logger)
}

// flagSet reports whether the flag was given on the command line.
func flagSet(flags *flag.FlagSet, name string) bool {
	set := false
	flags.Visit(func(f *flag.Flag) { set = set || f.Name == name })
	return set
}

func runServer(ctx context.Context, opts serveOptions, logger *slog.Logger) int {
	if err := migrateAndServe(ctx, opts, logger); err != nil {
		logger.Error("server stopped", "error", err)
		return 1
	}
	return 0
}

// serveOptions are the options of the serve command, from its flags in development mode or
// from the config file in server mode.
type serveOptions struct {
	// listen is config.ListenSystemd or a TCP address.
	listen string
	dev    bool
	// configPath is the config file of the server mode.
	configPath string
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
	cfg.BehindProxy = opts.listen == config.ListenSystemd
	handler, err := server.New(cfg)
	if err != nil {
		return err
	}

	listener, err := listen(opts.listen, os.Getenv, os.Getpid())
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	srv := newHTTPServer(handler, slog.NewLogLogger(logger.Handler(), slog.LevelWarn), defaultTimeouts)
	started := []any{"version", version, "commit", commit, "dev", opts.dev}
	if opts.listen == config.ListenSystemd {
		started = append(started, "listen", "systemd", "addr", listener.Addr().String())
	} else {
		started = append(started, "listen", "tcp", "addr", "http://"+listener.Addr().String())
	}
	if opts.configPath != "" {
		started = append(started, "config", opts.configPath)
	}
	logger.Info("server started", started...)

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
		// A client slower than the wait, such as one that stops sending its body, must not
		// make the shutdown, hence a deployment, fail: its connection is closed.
		if !errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("shutdown: %w", err)
		}
		logger.Warn("connections closed at the end of the shutdown wait", "wait", shutdownTimeout)
		if err := srv.Close(); err != nil {
			return fmt.Errorf("shutdown: close connections: %w", err)
		}
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
	dist, err := embeddedWeb()
	if err != nil {
		return server.Config{}, fmt.Errorf("embedded front end: %w", err)
	}
	return server.Config{Web: dist, Logger: logger}, nil
}
