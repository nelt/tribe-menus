// Command recetteauto runs the journeys of the acceptance run against the staging instance,
// and removes their videos once the acceptance run is over (plan recette-auto, D10, D11 and
// step 10; ADR 0032, points 12 and 13).
//
// Usage:
//
//	go run ./internal/tools/recetteauto run -version vX.Y.Z [-pr <number>] [-- <arguments of playwright test>]
//	go run ./internal/tools/recetteauto clean [-version vX.Y.Z]
//
// run checks the files of the volume tribe-menus-recette, that the plan of the acceptance run
// docs/recettes/vX.Y.Z.md exists, and that the staging instance serves the archive of the pull
// request (pr-<number>) or else the version, read on its public site; then it runs
// web/playwright.recette.config.ts, writing into a new directory vX.Y.Z/<time>/ of the volume
// of the videos, and shows the path of the control page. It exits in error when a journey
// fails. clean removes the directory of the version, or everything in the volume.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/nelt/tribe-menus/internal/tools/recetteconfig"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	e := env{client: &http.Client{Timeout: 30 * time.Second}, playwright: playwright, now: time.Now}
	os.Exit(e.main(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

// env is what the command needs from outside, replaced by the tests.
type env struct {
	client *http.Client
	// playwright runs playwright test in dir with the extra environment and arguments.
	playwright func(ctx context.Context, dir string, environ, args []string, stdout, stderr io.Writer) error
	now        func() time.Time
}

func playwright(ctx context.Context, dir string, environ, args []string, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, "npx", append([]string{"playwright", "test", "-c", "playwright.recette.config.ts"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), environ...)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	return cmd.Run()
}

var (
	versionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)
	// servedPattern finds the version in the footer of the public site (site/index.html,
	// {{VERSION}} replaced by internal/tools/dist).
	servedPattern = regexp.MustCompile(`Code source de cette version \(([^()<>\s]+)\)`)
)

func defaultVideos() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "recette-videos")
}

func (e env) main(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "recetteauto: want run or clean")
		return 2
	}
	flags := flag.NewFlagSet("recetteauto "+args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	version := flags.String("version", "", "version of the release, vX.Y.Z")
	videos := flags.String("videos", defaultVideos(), "volume of the videos")
	var pr *int
	var configDir, root *string
	switch args[0] {
	case "run":
		pr = flags.Int("pr", 0, "pull request whose archive the staging instance serves")
		configDir = flags.String("config", recetteconfig.DefaultDir(), "directory of "+recetteconfig.ConfigFile)
		root = flags.String("root", ".", "root of the repository")
	case "clean":
	default:
		fmt.Fprintln(stderr, "recetteauto: want run or clean")
		return 2
	}
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if *version != "" && !versionPattern.MatchString(*version) {
		fmt.Fprintln(stderr, "recetteauto: -version: want vX.Y.Z (make recette-auto VERSION=vX.Y.Z)")
		return 2
	}
	var err error
	if args[0] == "clean" {
		if flags.NArg() != 0 {
			fmt.Fprintln(stderr, "recetteauto: unexpected arguments")
			return 2
		}
		err = clean(*videos, *version, stdout)
	} else {
		if *version == "" || *pr < 0 {
			fmt.Fprintln(stderr, "recetteauto: run: want -version vX.Y.Z [-pr <number>]")
			return 2
		}
		err = e.run(ctx, *configDir, *videos, *root, *version, *pr, flags.Args(), stdout, stderr)
	}
	if err != nil {
		fmt.Fprintln(stderr, "recetteauto:", err)
		return 1
	}
	return 0
}

func (e env) run(ctx context.Context, configDir, videos, root, version string, pr int, extra []string, stdout, stderr io.Writer) error {
	cfg, _, err := recetteconfig.Read(configDir)
	if err != nil {
		return fmt.Errorf("volume tribe-menus-recette: %w", err)
	}
	plan, err := filepath.Abs(filepath.Join(root, "docs", "recettes", version+".md"))
	if err != nil {
		return fmt.Errorf("plan of the acceptance run: %w", err)
	}
	if _, err := os.Stat(plan); err != nil {
		return fmt.Errorf("plan of the acceptance run docs/recettes/%s.md needed first (docs/recette.md): %w", version, err)
	}
	want := version
	if pr > 0 {
		want = "pr-" + strconv.Itoa(pr)
	}
	origin, _ := cfg.Origin()
	served, err := e.served(ctx, origin)
	if err != nil {
		return err
	}
	if served != want {
		return fmt.Errorf("the staging instance serves %s, not %s: deploy it first (tribe-menus-deploy --env recette)", served, want)
	}

	out := filepath.Join(videos, version, e.now().Format("20060102-150405"))
	if err := os.MkdirAll(out, 0o700); err != nil {
		return fmt.Errorf("directory of the run: %w", err)
	}
	fmt.Fprintf(stdout, "Staging instance serving %s; run written to %s\n", served, out)
	environ := []string{"RECETTE_CONFIG_DIR=" + configDir, "RECETTE_OUTPUT_DIR=" + out, "RECETTE_PLAN=" + plan}
	runErr := e.playwright(ctx, filepath.Join(root, "web"), environ, extra, stdout, stderr)
	page := filepath.Join(out, "controle.html")
	if _, err := os.Stat(page); err == nil {
		fmt.Fprintf(stdout, "\nControl page: %s\n", page)
	}
	if runErr != nil {
		return fmt.Errorf("playwright: %w", runErr)
	}
	return nil
}

// served reads the version served by the instance on its public site.
func (e env) served(ctx context.Context, origin string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, origin+"/", nil)
	if err != nil {
		return "", fmt.Errorf("served version: %w", err)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("served version: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("served version: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("served version: status %d on the public site", resp.StatusCode)
	}
	m := servedPattern.FindSubmatch(body)
	if m == nil {
		return "", errors.New("served version: not found on the public site")
	}
	return string(m[1]), nil
}

// clean removes the runs of a version, or of all of them; the volume itself stays.
func clean(videos, version string, stdout io.Writer) error {
	targets := []string{filepath.Join(videos, version)}
	if version == "" {
		entries, err := os.ReadDir(videos)
		if err != nil {
			return fmt.Errorf("volume of the videos: %w", err)
		}
		targets = targets[:0]
		for _, e := range entries {
			targets = append(targets, filepath.Join(videos, e.Name()))
		}
	}
	for _, t := range targets {
		if _, err := os.Lstat(t); errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stdout, "Nothing to remove: %s\n", t)
			continue
		}
		if err := os.RemoveAll(t); err != nil {
			return fmt.Errorf("remove: %w", err)
		}
		fmt.Fprintf(stdout, "Removed: %s\n", t)
	}
	return nil
}
