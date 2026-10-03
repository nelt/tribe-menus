// Package admin implements the admin command (EF-08 to EF-11): interactive operations
// run on the server, outside the application, traced with the admin command as author.
package admin

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/nelt/tribe-menus/internal/storage"
	"github.com/nelt/tribe-menus/internal/tribe"
)

// ErrInputClosed is returned when the input ends before every question is answered.
var ErrInputClosed = errors.New("input closed before the end of the questions")

// Questions asked by Init, each ending the output until it is answered.
const (
	AskTribeName   = "Nom de la tribu : "
	AskSlug        = "Identifiant d'URL : "
	AskEmail       = "E-mail du premier membre : "
	AskDisplayName = "Nom d'affichage du premier membre (facultatif) : "
)

// Refusal messages, each followed by the same question.
const (
	refusedEmptyName = "Le nom de la tribu ne peut pas être vide."
	refusedSlugForm  = "Identifiant refusé : de 3 à 40 caractères, lettres minuscules sans accent, chiffres et tirets, " +
		"une lettre en premier, ni tiret final ni tirets consécutifs."
	refusedSlugTaken = "Identifiant refusé : une tribu l'utilise déjà."
	refusedEmail     = "Adresse refusée : elle doit être de la forme nom@domaine."
)

// Command is the admin command, reading answers from In and writing to Out.
type Command struct {
	Store *storage.Store
	In    io.Reader
	Out   io.Writer
	// BaseURL is the address of the instance, used to show the URL of a tribe.
	BaseURL string
	Now     func() time.Time
}

// Init creates a tribe (EF-08): it asks for its name, its slug, asked again while invalid
// or taken, the address of the first member and an optional display name. No email is sent.
func (c *Command) Init(ctx context.Context) error {
	p := &prompter{in: bufio.NewScanner(c.In), out: c.Out}

	name, err := ask(p, AskTribeName, func(answer string) (string, string, error) {
		if answer == "" {
			return "", refusedEmptyName, nil
		}
		return answer, "", nil
	})
	if err != nil {
		return err
	}

	slug, err := ask(p, AskSlug, func(answer string) (tribe.Slug, string, error) {
		slug, err := tribe.ParseSlug(answer)
		if err != nil {
			return "", refusedSlugForm, nil
		}
		taken, err := c.Store.SlugTaken(ctx, string(slug))
		if err != nil {
			return "", "", err
		}
		if taken {
			return "", refusedSlugTaken, nil
		}
		return slug, "", nil
	})
	if err != nil {
		return err
	}

	email, err := ask(p, AskEmail, func(answer string) (tribe.Email, string, error) {
		email, err := tribe.ParseEmail(answer)
		if err != nil {
			return "", refusedEmail, nil
		}
		return email, "", nil
	})
	if err != nil {
		return err
	}

	displayName, err := ask(p, AskDisplayName, func(answer string) (string, string, error) { return answer, "", nil })
	if err != nil {
		return err
	}

	err = c.Store.CreateTribe(ctx, string(slug), name, func(ctx context.Context, db *sql.DB) error {
		return tribe.NewStore(db).Initialize(ctx, name, email, displayName, c.Now())
	})
	if err != nil {
		return fmt.Errorf("create tribe %s: %w", slug, err)
	}
	fmt.Fprintf(c.Out, "\nTribu « %s » créée.\nURL à transmettre au premier membre : %s\n", name, c.TribeURL(slug))
	return nil
}

// TribeURL returns the URL of the tribe on this instance.
func (c *Command) TribeURL(slug tribe.Slug) string {
	return strings.TrimRight(c.BaseURL, "/") + "/tribes/" + string(slug) + "/"
}

type prompter struct {
	in  *bufio.Scanner
	out io.Writer
}

// ask asks the question until check accepts the answer, that is returns neither a refusal
// message, shown before asking again, nor an error, which stops the questions.
func ask[T any](p *prompter, question string, check func(answer string) (T, string, error)) (T, error) {
	for {
		fmt.Fprint(p.out, question)
		if !p.in.Scan() {
			var zero T
			if err := p.in.Err(); err != nil {
				return zero, fmt.Errorf("read answer: %w", err)
			}
			return zero, ErrInputClosed
		}
		value, refusal, err := check(strings.TrimSpace(p.in.Text()))
		if err != nil || refusal == "" {
			return value, err
		}
		fmt.Fprintln(p.out, refusal)
	}
}
