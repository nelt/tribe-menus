// Command tribe-menus runs the Melting Tribe application.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nelt/tribe-menus/internal/admin"
	"github.com/nelt/tribe-menus/internal/server"
	"github.com/nelt/tribe-menus/internal/storage"
	"github.com/nelt/tribe-menus/web"
)

// Set at build time with -ldflags -X (ADR 0012).
var (
	version = "dev"
	commit  = "unknown"
)

const (
	readHeaderTimeout = 10 * time.Second
	shutdownTimeout   = 10 * time.Second
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
	if err := flags.Parse(args); err != nil {
		return 2
	}

	logger := slog.New(slog.NewTextHandler(stderr, nil))
	if err := migrateAndServe(ctx, *data, *addr, *dev, *root, logger); err != nil {
		logger.Error("server stopped", "error", err)
		return 1
	}
	return 0
}

// migrateAndServe opens and migrates every database before listening, so that the server,
// and its health check, never answer on a database left unmigrated (ADR 0003, point 6).
func migrateAndServe(ctx context.Context, data, addr string, dev bool, root string, logger *slog.Logger) (err error) {
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
	logger.Info("databases migrated", "data", data)
	return listenAndServe(ctx, addr, dev, root, logger)
}

func listenAndServe(ctx context.Context, addr string, dev bool, root string, logger *slog.Logger) error {
	cfg, err := serverConfig(dev, root, logger)
	if err != nil {
		return err
	}
	handler, err := server.New(cfg)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}
	logger.Info("server started", "version", version, "commit", commit, "addr", "http://"+listener.Addr().String(), "dev", dev)

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
