package acceptance

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cucumber/godog"
	"github.com/nelt/tribe-menus/internal/admin"
	"github.com/nelt/tribe-menus/internal/storage"
	"github.com/nelt/tribe-menus/internal/tribe"
)

// questions maps the words of the scenarios to the questions of the admin command.
var questions = map[string]string{
	"nom de la tribu":          admin.AskTribeName,
	"identifiant d'URL":        admin.AskSlug,
	"e-mail du premier membre": admin.AskEmail,
}

// auditOperations maps the audit operations of the scenarios to the glossary.
var auditOperations = map[string]tribe.AuditOperation{
	"initialisation de la tribu": tribe.TribeInitialized,
}

// registerAdministrationSteps: administration.feature (EF-08 to EF-11).
func (w *world) registerAdministrationSteps(sc *godog.ScenarioContext) {
	sc.Step(`^je lance le script d'initialisation$`, w.startInit)
	sc.Step(`^je réponds "([^"]*)" comme (nom de la tribu|identifiant d'URL|e-mail du premier membre)$`, w.answer)
	sc.Step(`^le script refuse cet identifiant et m'en demande un autre$`, w.slugRefused)
	sc.Step(`^la tribu "([^"]*)" d'identifiant "([^"]*)" est créée$`, w.tribeCreated)
	sc.Step(`^"([^"]*)" en est membre actif$`, w.activeMemberOfLastTribe)
	sc.Step(`^le script affiche l'URL de la tribu "([^"]*)"$`, w.urlShown)
	sc.Step(`^aucun e-mail n'est envoyé à "([^"]*)"$`, w.noMailSent)
	sc.Step(`^le journal d'audit de la tribu "([^"]*)" contient une entrée "([^"]*)" avec "script d'administration" comme auteur$`, w.auditEntryByAdminCommand)

	sc.Step(`^la tribu d'identifiant "([^"]*)" existe$`, w.tribeExists)
	sc.Step(`^j'initialise la tribu "([^"]*)" d'identifiant "([^"]*)"$`, func(ctx context.Context, name, slug string) error {
		return w.initTribe(ctx, name, slug, defaultAnswers[admin.AskEmail])
	})
	sc.Step(`^j'initialise la tribu "([^"]*)" d'identifiant "([^"]*)" avec le premier membre "([^"]*)"$`, w.initTribe)
	sc.Step(`^les tribus "([^"]*)" et "([^"]*)" existent toutes les deux$`, w.bothTribesExist)
	sc.Step(`^"([^"]*)" est membre actif de la tribu "([^"]*)"$`, w.givenActiveMember)
	sc.Step(`^"([^"]*)" est membre actif des tribus "([^"]*)" et "([^"]*)"$`, w.activeMemberOfBoth)
}

func (w *world) startInit(ctx context.Context) error {
	cmd, err := w.adminCommand(ctx)
	if err != nil {
		return err
	}
	w.admin = startInit(ctx, cmd)
	return nil
}

func (w *world) answer(value, question string) error {
	if w.admin == nil {
		return errors.New("no admin command started")
	}
	return w.admin.answer(questions[question], value)
}

func (w *world) slugRefused() error {
	question, shown, err := w.admin.next()
	if err != nil {
		return err
	}
	if !strings.Contains(shown, "Identifiant refusé") || question != admin.AskSlug {
		return fmt.Errorf("want a refusal then the slug question again, got %q then %q", shown, question)
	}
	return nil
}

func (w *world) tribeCreated(ctx context.Context, name, slug string) error {
	if err := w.admin.finish(); err != nil {
		return fmt.Errorf("admin init: %w", err)
	}
	t, err := w.tribeStore(ctx, slug)
	if err != nil {
		return err
	}
	got, err := t.Name(ctx)
	if err != nil {
		return err
	}
	if got != name {
		return fmt.Errorf("tribe %q is named %q, want %q", slug, got, name)
	}
	w.tribe = tribe.Slug(slug)
	return nil
}

func (w *world) activeMemberOfLastTribe(ctx context.Context, email string) error {
	return w.isActiveMember(ctx, email, string(w.tribe))
}

func (w *world) isActiveMember(ctx context.Context, email, slug string) error {
	t, err := w.tribeStore(ctx, slug)
	if err != nil {
		return err
	}
	member, err := t.Member(ctx, tribe.Email(email))
	if err != nil {
		return fmt.Errorf("%s in tribe %q: %w", email, slug, err)
	}
	if member.Status != tribe.Active {
		return fmt.Errorf("%s in tribe %q is %s, want active", email, slug, member.Status)
	}
	return nil
}

func (w *world) urlShown(slug string) error {
	want := baseURL + "/tribes/" + slug + "/"
	if output := w.admin.out.String(); !strings.Contains(output, want) {
		return fmt.Errorf("output %q does not show %s", output, want)
	}
	return nil
}

func (w *world) noMailSent(email string) error {
	for _, m := range w.mails {
		if m.to == tribe.Email(email) {
			return fmt.Errorf("a message was sent to %s", email)
		}
	}
	return nil
}

func (w *world) auditEntryByAdminCommand(ctx context.Context, slug, operation string) error {
	op, ok := auditOperations[operation]
	if !ok {
		return fmt.Errorf("%w: audit operation %q", godog.ErrUndefined, operation)
	}
	t, err := w.tribeStore(ctx, slug)
	if err != nil {
		return err
	}
	entries, err := t.AuditLog(ctx)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Operation == op && e.ByAdminCommand() {
			return nil
		}
	}
	return fmt.Errorf("no %s entry by the admin command in %+v", op, entries)
}

// initTribe runs admin init with every answer at once, as a setup or a single action.
func (w *world) initTribe(ctx context.Context, name, slug, email string) error {
	cmd, err := w.adminCommand(ctx)
	if err != nil {
		return err
	}
	cmd.In = strings.NewReader(strings.Join([]string{name, slug, email, ""}, "\n") + "\n")
	var out strings.Builder
	cmd.Out = &out
	if err := cmd.Init(ctx); err != nil {
		return fmt.Errorf("admin init %s: %w (output: %s)", slug, err, out.String())
	}
	w.tribe = tribe.Slug(slug)
	return nil
}

func (w *world) tribeExists(ctx context.Context, slug string) error {
	return w.initTribe(ctx, "Tribu "+slug, slug, defaultAnswers[admin.AskEmail])
}

func (w *world) bothTribesExist(ctx context.Context, first, second string) error {
	for _, slug := range []string{first, second} {
		if _, err := w.tribeStore(ctx, slug); err != nil {
			return err
		}
	}
	return nil
}

// givenActiveMember creates the tribe with this first member. Adding a member to an
// existing tribe comes with EF-01.
func (w *world) givenActiveMember(ctx context.Context, email, slug string) error {
	store, err := w.openStore(ctx)
	if err != nil {
		return err
	}
	if _, err := store.Tribe(ctx, slug); !errors.Is(err, storage.ErrUnknownTribe) {
		if err != nil {
			return err
		}
		return fmt.Errorf("%w: adding a member to an existing tribe (EF-01)", godog.ErrPending)
	}
	return w.initTribe(ctx, "Tribu "+slug, slug, email)
}

func (w *world) activeMemberOfBoth(ctx context.Context, email, first, second string) error {
	for _, slug := range []string{first, second} {
		if err := w.isActiveMember(ctx, email, slug); err != nil {
			return err
		}
	}
	return nil
}
