package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/nelt/tribe-menus/internal/scenarios"
)

func TestExport(t *testing.T) {
	features := fstest.MapFS{"a.feature": {Data: []byte(`# language: fr
Fonctionnalité: A

  @ui
  Scénario: Dans le navigateur
    Quand je recherche "chili"

  Scénario: Contre l'API
    Quand je crée le plat "Chili"

  @ui @manuel
  Scénario: À la main
    Quand je regarde
`)}}
	out := filepath.Join(t.TempDir(), "sub", "scenarios.json")
	if err := export(features, out); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var got []scenarios.Scenario
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Title != "Dans le navigateur" || got[0].Steps[0].Text != `je recherche "chili"` {
		t.Errorf("exported %+v", got)
	}
}

func TestRunUsage(t *testing.T) {
	var stderr bytes.Buffer
	if code := run(nil, &stderr); code != 2 {
		t.Errorf("exit code %d, stderr %s", code, &stderr)
	}
}
