// Package scenarios reads the Gherkin scenarios of docs/specs/features with the parser that
// godog uses (ADR 0028, point 7): the scenarios @ui exported for Playwright, and their keys,
// which godog leaves to Playwright in acceptance/pending.txt. Development only: the binary
// does not import it.
package scenarios

import (
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"

	gherkin "github.com/cucumber/gherkin/go/v42"
	messages "github.com/cucumber/messages/go/v34"
)

// UITag marks the scenarios proved in the browser (ADR 0005, 0028).
const UITag = "@ui"

// Scenario is a scenario, or one example of a scenario outline.
type Scenario struct {
	// File is the name of the .feature file; Title, the title of the scenario, the same for
	// every example of an outline. "<file>: <title>" is its key in pending.txt.
	File  string   `json:"file"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
	// Example is the number of the example in its outline, from 1; 0 for a scenario.
	Example int    `json:"example"`
	Steps   []Step `json:"steps"`
}

// Step is a step, background steps first, with its type (Context, Action or Outcome) and its
// data table, if any, rows of cells, the header first.
type Step struct {
	Type  string     `json:"type"`
	Text  string     `json:"text"`
	Table [][]string `json:"table,omitempty"`
}

// Key is the key of the scenario in pending.txt.
func (s Scenario) Key() string {
	return s.File + ": " + s.Title
}

// HasTag reports whether the scenario carries the tag, "@" included.
func (s Scenario) HasTag(tag string) bool {
	return slices.Contains(s.Tags, tag)
}

// Read reads the .feature files at the root of fsys, in the order of their names, with the
// examples of the outlines unrolled.
func Read(fsys fs.FS) ([]Scenario, error) {
	names, err := fs.Glob(fsys, "*.feature")
	if err != nil {
		return nil, fmt.Errorf("list features: %w", err)
	}
	var all []Scenario
	for _, name := range names {
		read, err := readFile(fsys, name)
		if err != nil {
			return nil, err
		}
		all = append(all, read...)
	}
	return all, nil
}

func readFile(fsys fs.FS, name string) ([]Scenario, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	defer f.Close()
	ids := 0
	newID := func() string {
		ids++
		return strconv.Itoa(ids)
	}
	doc, err := gherkin.ParseGherkinDocument(f, newID)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	var list []Scenario
	examples := map[string]int{} // examples met so far, by title
	for _, p := range gherkin.Pickles(*doc, name, newID) {
		s := Scenario{File: path.Base(name), Title: p.Name, Tags: []string{}}
		for _, t := range p.Tags {
			s.Tags = append(s.Tags, t.Name)
		}
		if len(p.AstNodeIds) > 1 { // the scenario, then the row of its example
			examples[p.Name]++
			s.Example = examples[p.Name]
		}
		for _, st := range p.Steps {
			s.Steps = append(s.Steps, Step{Type: string(st.Type), Text: st.Text, Table: table(st.Argument)})
		}
		list = append(list, s)
	}
	return list, nil
}

func table(arg *messages.PickleStepArgument) [][]string {
	if arg == nil || arg.DataTable == nil {
		return nil
	}
	var rows [][]string
	for _, r := range arg.DataTable.Rows {
		var cells []string
		for _, c := range r.Cells {
			cells = append(cells, c.Value)
		}
		rows = append(rows, cells)
	}
	return rows
}
