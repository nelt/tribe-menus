package scenarios

import (
	"os"
	"reflect"
	"testing"
	"testing/fstest"
)

const feature = `# language: fr
Fonctionnalité: Essai

  Contexte:
    Étant donné une tribu

  @P1 @ui
  Scénario: Un scénario
    Quand je crée le plat "Chili" avec les ingrédients :
      | ingrédient | quantité |
      | bœuf       | 500      |
    Alors il figure dans la bibliothèque

  @P3
  Plan du scénario: Un plan
    Quand je recherche "<recherche>"
    Alors les résultats sont "<résultats>"

    @ui
    Exemples:
      | recherche | résultats |
      | chili     | Chili     |
      | poulet    |           |
`

func TestRead(t *testing.T) {
	got, err := Read(fstest.MapFS{"essai.feature": {Data: []byte(feature)}, "README.md": {Data: []byte("#")}})
	if err != nil {
		t.Fatal(err)
	}
	want := []Scenario{
		{File: "essai.feature", Title: "Un scénario", Tags: []string{"@P1", "@ui"}, Steps: []Step{
			{Type: "Context", Text: "une tribu"},
			{Type: "Action", Text: `je crée le plat "Chili" avec les ingrédients :`, Table: [][]string{{"ingrédient", "quantité"}, {"bœuf", "500"}}},
			{Type: "Outcome", Text: "il figure dans la bibliothèque"},
		}},
		{File: "essai.feature", Title: "Un plan", Tags: []string{"@P3", "@ui"}, Example: 1, Steps: []Step{
			{Type: "Context", Text: "une tribu"},
			{Type: "Action", Text: `je recherche "chili"`},
			{Type: "Outcome", Text: `les résultats sont "Chili"`},
		}},
		{File: "essai.feature", Title: "Un plan", Tags: []string{"@P3", "@ui"}, Example: 2, Steps: []Step{
			{Type: "Context", Text: "une tribu"},
			{Type: "Action", Text: `je recherche "poulet"`},
			{Type: "Outcome", Text: `les résultats sont ""`},
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
	if got[1].Key() != "essai.feature: Un plan" || !got[1].HasTag(UITag) || got[0].HasTag("@P3") {
		t.Errorf("key %q", got[1].Key())
	}
}

func TestReadSpecs(t *testing.T) {
	all, err := Read(os.DirFS("../../docs/specs/features"))
	if err != nil {
		t.Fatal(err)
	}
	ui := 0
	for _, s := range all {
		if s.HasTag(UITag) {
			ui++
		}
	}
	if len(all) < 100 || ui == 0 {
		t.Errorf("%d scenarios, %d @ui", len(all), ui)
	}
}

func TestReadInvalid(t *testing.T) {
	if _, err := Read(fstest.MapFS{"a.feature": {Data: []byte("# language: fr\nScénario sans fonctionnalité\n  Quand")}}); err == nil {
		t.Error("invalid feature accepted")
	}
}
