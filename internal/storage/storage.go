// Package storage opens and migrates the SQLite databases: the registry of the tribes,
// one database per tribe (ADR 0003) and the rate limit database (ADR 0021).
package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/nelt/tribe-menus/internal/storage/registrydb"
	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

const (
	registryFile  = "registry.db"
	rateLimitFile = "ratelimit.db"
	tribesDir    = "tribes"
	tribeFileExt = ".db"

	busyTimeoutMillis = 5000
)

var (
	// ErrUnknownTribe is returned for a slug absent from the registry.
	ErrUnknownTribe = errors.New("unknown tribe")
	// ErrSlugTaken is returned when creating a tribe whose slug is already in the registry.
	ErrSlugTaken = errors.New("slug already taken")
)

//go:embed migrations
var embedded embed.FS

// Migrations holds one series of migration files per kind of database.
type Migrations struct {
	Registry  fs.FS
	Tribe     fs.FS
	RateLimit fs.FS
}

// EmbeddedMigrations returns the migrations built into the binary.
func EmbeddedMigrations() (Migrations, error) {
	registry, err := fs.Sub(embedded, "migrations/registry")
	if err != nil {
		return Migrations{}, fmt.Errorf("storage: registry migrations: %w", err)
	}
	tribe, err := fs.Sub(embedded, "migrations/tribe")
	if err != nil {
		return Migrations{}, fmt.Errorf("storage: tribe migrations: %w", err)
	}
	rateLimit, err := fs.Sub(embedded, "migrations/ratelimit")
	if err != nil {
		return Migrations{}, fmt.Errorf("storage: rate limit migrations: %w", err)
	}
	return Migrations{Registry: registry, Tribe: tribe, RateLimit: rateLimit}, nil
}

// Store gives access to the registry and to the database of each tribe.
// Each database has a single connection, so that its writes are serialized (ADR 0003, point 5).
type Store struct {
	dir        string
	migrations Migrations
	registry   *sql.DB
	queries    *registrydb.Queries
	rateLimit  *sql.DB

	mu     sync.Mutex
	tribes map[string]*sql.DB // by file name
}

// Open opens the databases of the data directory, creating it if needed, and migrates
// the registry, the rate limit database and then every tribe database (ADR 0003, point 6).
func Open(ctx context.Context, dir string, migrations Migrations) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dir, tribesDir), 0o700); err != nil {
		return nil, fmt.Errorf("storage: create data directory: %w", err)
	}
	registry, err := openDB(ctx, filepath.Join(dir, registryFile), true)
	if err != nil {
		return nil, fmt.Errorf("storage: registry: %w", err)
	}
	s := &Store{dir: dir, migrations: migrations, registry: registry, queries: registrydb.New(registry), tribes: map[string]*sql.DB{}}
	if err := migrate(ctx, registry, migrations.Registry); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("storage: registry: %w", err)
	}
	rateLimit, err := openDB(ctx, filepath.Join(dir, rateLimitFile), true)
	if err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("storage: rate limit database: %w", err)
	}
	s.rateLimit = rateLimit
	if err := migrate(ctx, rateLimit, migrations.RateLimit); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("storage: rate limit database: %w", err)
	}
	if err := s.openAllTribes(ctx); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

// Registry returns the registry database.
func (s *Store) Registry() *sql.DB {
	return s.registry
}

// RateLimit returns the rate limit database (ADR 0021).
func (s *Store) RateLimit() *sql.DB {
	return s.rateLimit
}

// Tribes returns the slugs of the tribes of the registry, in order.
func (s *Store) Tribes(ctx context.Context) ([]string, error) {
	slugs, err := s.queries.TribeSlugs(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage: list tribes: %w", err)
	}
	return slugs, nil
}

// Tribe returns the database of the tribe with this slug, or ErrUnknownTribe.
// A tribe created by another process since Open is opened and migrated on first use.
func (s *Store) Tribe(ctx context.Context, slug string) (*sql.DB, error) {
	file, err := s.queries.TribeFile(ctx, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUnknownTribe
	}
	if err != nil {
		return nil, fmt.Errorf("storage: look up tribe: %w", err)
	}
	return s.tribeDB(ctx, file)
}

// CreateTribe creates the database of a new tribe, migrates it, lets setup fill it,
// then records the tribe in the registry. On failure, neither the file nor the registry
// entry remains.
func (s *Store) CreateTribe(ctx context.Context, slug, name string, setup func(context.Context, *sql.DB) error) (err error) {
	taken, err := s.SlugTaken(ctx, slug)
	if err != nil {
		return err
	}
	if taken {
		return ErrSlugTaken
	}

	file, err := newTribeFileName()
	if err != nil {
		return err
	}
	path := s.tribePath(file)
	db, err := openDB(ctx, path, true)
	if err != nil {
		return fmt.Errorf("storage: create tribe database: %w", err)
	}
	defer func() {
		if err != nil {
			_ = db.Close()
			removeDatabase(path)
		}
	}()
	if err := migrate(ctx, db, s.migrations.Tribe); err != nil {
		return fmt.Errorf("storage: tribe %s: %w", file, err)
	}
	if err := setup(ctx, db); err != nil {
		return err
	}
	if err := s.register(ctx, slug, name, file); err != nil {
		return err
	}

	s.mu.Lock()
	s.tribes[file] = db
	s.mu.Unlock()
	return nil
}

// Close closes every database.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	errs := []error{s.registry.Close()}
	if s.rateLimit != nil {
		errs = append(errs, s.rateLimit.Close())
	}
	for file, db := range s.tribes {
		errs = append(errs, db.Close())
		delete(s.tribes, file)
	}
	return errors.Join(errs...)
}

// SlugTaken reports whether a tribe of the registry has this slug.
func (s *Store) SlugTaken(ctx context.Context, slug string) (bool, error) {
	n, err := s.queries.CountTribesWithSlug(ctx, slug)
	if err != nil {
		return false, fmt.Errorf("storage: look up slug: %w", err)
	}
	return n > 0, nil
}

func (s *Store) register(ctx context.Context, slug, name, file string) error {
	err := s.queries.InsertTribe(ctx, registrydb.InsertTribeParams{Slug: slug, Name: name, File: file})
	if err != nil {
		// Another process may have taken the slug since slugTaken.
		if taken, lookupErr := s.SlugTaken(ctx, slug); lookupErr == nil && taken {
			return ErrSlugTaken
		}
		return fmt.Errorf("storage: register tribe: %w", err)
	}
	return nil
}

func (s *Store) openAllTribes(ctx context.Context) error {
	files, err := s.queries.TribeFiles(ctx)
	if err != nil {
		return fmt.Errorf("storage: list tribes: %w", err)
	}
	for _, file := range files {
		if _, err := s.tribeDB(ctx, file); err != nil {
			return err
		}
	}
	return nil
}

// tribeDB returns the open database of a tribe file, opening and migrating it if needed.
func (s *Store) tribeDB(ctx context.Context, file string) (*sql.DB, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if db, ok := s.tribes[file]; ok {
		return db, nil
	}
	db, err := openDB(ctx, s.tribePath(file), false)
	if err != nil {
		return nil, fmt.Errorf("storage: tribe %s: %w", file, err)
	}
	if err := migrate(ctx, db, s.migrations.Tribe); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("storage: tribe %s: %w", file, err)
	}
	s.tribes[file] = db
	return db, nil
}

func (s *Store) tribePath(file string) string {
	return filepath.Join(s.dir, tribesDir, file)
}

// newTribeFileName returns a random file name, never derived from the slug.
func newTribeFileName() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("storage: random file name: %w", err)
	}
	return fmt.Sprintf("%x%s", b, tribeFileExt), nil
}

// openDB opens a database with a single connection, in WAL mode, with foreign keys
// enforced and a busy timeout. Without create, a missing file is an error.
func openDB(ctx context.Context, path string, create bool) (*sql.DB, error) {
	mode := "rw"
	if create {
		mode = "rwc"
	}
	query := url.Values{
		"mode":          {mode},
		"_journal_mode": {"WAL"},
		"_foreign_keys": {"1"},
		"_busy_timeout": {fmt.Sprint(busyTimeoutMillis)},
		"_txlock":       {"immediate"},
	}
	dsn := "file:" + (&url.URL{Path: path}).EscapedPath() + "?" + query.Encode()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxIdleTime(0)
	db.SetConnMaxLifetime(0)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return db, nil
}

// removeDatabase removes a database file and its WAL companions.
func removeDatabase(path string) {
	for _, suffix := range []string{"", "-wal", "-shm"} {
		_ = os.Remove(path + suffix)
	}
}
