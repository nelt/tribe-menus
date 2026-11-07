package dish

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

// TestNormalizeName runs the table of cases shared with web/src/dishes.ts (D9).
func TestNormalizeName(t *testing.T) {
	data, err := os.ReadFile("testdata/normalized-names.json")
	if err != nil {
		t.Fatal(err)
	}
	var table struct {
		Cases []struct {
			Name       string `json:"name"`
			Normalized string `json:"normalized"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &table); err != nil {
		t.Fatal(err)
	}
	if len(table.Cases) == 0 {
		t.Fatal("no case")
	}
	for _, c := range table.Cases {
		if got := NormalizeName(c.Name); got != c.Normalized {
			t.Errorf("NormalizeName(%q) = %q, want %q", c.Name, got, c.Normalized)
		}
	}
}

func TestParseUnit(t *testing.T) {
	for _, u := range []string{"g", "kg", "ml", "cl", "l", "piece", "tablespoon", "teaspoon", "pinch"} {
		got, err := ParseUnit(u)
		if err != nil || string(got) != u {
			t.Errorf("ParseUnit(%q) = %q, %v", u, got, err)
		}
	}
	for _, u := range []string{"", "G", "pièce", "cup", " g"} {
		if _, err := ParseUnit(u); !errors.Is(err, ErrUnknownUnit) {
			t.Errorf("ParseUnit(%q): %v, want ErrUnknownUnit", u, err)
		}
	}
}

func TestFamilies(t *testing.T) {
	cases := []struct {
		unit   Unit
		family Family
	}{
		{Gram, Mass}, {Kilogram, Mass},
		{Millilitre, Volume}, {Centilitre, Volume}, {Litre, Volume},
		{Piece, PieceFamily}, {Tablespoon, TablespoonFamily}, {Teaspoon, TeaspoonFamily}, {Pinch, PinchFamily},
	}
	for _, c := range cases {
		if got := c.unit.Family(); got != c.family {
			t.Errorf("%s: family %s, want %s", c.unit, got, c.family)
		}
	}
}

// TestParseQuantity runs the table of cases shared with web/src/dishes.ts (D20).
func TestParseQuantity(t *testing.T) {
	data, err := os.ReadFile("testdata/quantities.json")
	if err != nil {
		t.Fatal(err)
	}
	var table struct {
		Cases []struct {
			Text   string `json:"text"`
			Unit   Unit   `json:"unit"`
			Amount int64  `json:"amount"` // in the base unit
			Error  string `json:"error"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &table); err != nil {
		t.Fatal(err)
	}
	if len(table.Cases) == 0 {
		t.Fatal("no case")
	}
	wantErrs := map[string]error{"": nil, string(InvalidQuantity): ErrInvalidQuantity, string(QuantityTooPrecise): ErrQuantityTooPrecise}
	for _, c := range table.Cases {
		wantErr, ok := wantErrs[c.Error]
		if !ok {
			t.Fatalf("unknown error %q", c.Error)
		}
		got, err := ParseQuantity(c.Text, c.Unit)
		if !errors.Is(err, wantErr) {
			t.Errorf("ParseQuantity(%q, %s): error %v, want %v", c.Text, c.Unit, err, wantErr)
			continue
		}
		if err == nil && (got.Amount != c.Amount || got.Unit != c.Unit) {
			t.Errorf("ParseQuantity(%q, %s) = %+v, want %d %s", c.Text, c.Unit, got, c.Amount, c.Unit)
		}
	}
}

func TestQuantityText(t *testing.T) {
	cases := []struct {
		q    Quantity
		want string
	}{
		{Quantity{500_000, Gram}, "500"},
		{Quantity{25, Centilitre}, "2.5"},
		{Quantity{125_000, Kilogram}, "0.125"},
		{Quantity{1500, Piece}, "1.5"},
		{Quantity{7, Millilitre}, "7"},
		{Quantity{1, Gram}, "0.001"},
		{Quantity{1_000_000, Kilogram}, "1"},
	}
	for _, c := range cases {
		if got := c.q.Text(); got != c.want {
			t.Errorf("%+v: %q, want %q", c.q, got, c.want)
		}
		back, err := ParseQuantity(c.q.Text(), c.q.Unit)
		if err != nil || back != c.q {
			t.Errorf("%+v read back as %+v, %v", c.q, back, err)
		}
	}
}

func TestPickColor(t *testing.T) {
	all := func(n int) map[Color]int {
		counts := map[Color]int{}
		for _, c := range Palette {
			counts[c] = n
		}
		return counts
	}
	cases := []struct {
		name   string
		counts map[Color]int
		want   Color
	}{
		{"empty library: the first of the palette", nil, "#C2502E"},
		{"the least used", func() map[Color]int { c := all(1); c["#6E6680"] = 0; return c }(), "#6E6680"},
		{"tie: the first in palette order", func() map[Color]int { c := all(2); c["#B0306A"] = 1; c["#2E8A6E"] = 1; return c }(), "#B0306A"},
		{"all equal", all(3), "#C2502E"},
		{"a color out of the palette is ignored", map[Color]int{"#000000": 0, "#C2502E": 1}, "#3F7F25"},
	}
	for _, c := range cases {
		if got := PickColor(c.counts); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
	if len(Palette) != 9 {
		t.Errorf("%d hues, want the 9 of docs/design/README.md", len(Palette))
	}
}

func TestValidate(t *testing.T) {
	line := func(name, quantity, unit string) LineInput {
		return LineInput{Name: name, Quantity: quantity, Unit: unit}
	}
	lines := func(n int) []LineInput {
		var l []LineInput
		for i := range n {
			l = append(l, line("ingrédient "+string(rune('a'+i%26))+strings.Repeat("x", i/26), "", ""))
		}
		return l
	}
	cases := []struct {
		name     string
		input    Input
		wantCode ErrorCode
		wantLine int
		wantIngr string
	}{
		{name: "valid", input: Input{Name: "Chili", Lines: []LineInput{line("bœuf haché", "500", "g"), line("sel", "", "")}}},
		{name: "no ingredient", input: Input{Name: "Restes"}},
		{name: "name required", input: Input{Name: "   "}, wantCode: NameRequired, wantLine: -1},
		{name: "name of a zero width space", input: Input{Name: "\u200b"}, wantCode: NameRequired, wantLine: -1},
		{name: "name of a combining mark", input: Input{Name: "\u0301"}, wantCode: NameRequired, wantLine: -1},
		{name: "name with a control character", input: Input{Name: "Soupe\a"}, wantCode: NameInvalid, wantLine: -1},
		{name: "name of a control character", input: Input{Name: "\x00"}, wantCode: NameInvalid, wantLine: -1},
		{name: "name with a tab", input: Input{Name: "Soupe\tà l'oignon"}},
		{name: "name of an emoji", input: Input{Name: "🍲"}},
		{name: "name of a composed emoji", input: Input{Name: "👩\u200d🍳"}},
		{name: "name of 60 characters once spaces reduced", input: Input{Name: " " + strings.Repeat("é", 30) + "    " + strings.Repeat("é", 29) + " "}},
		{name: "name of 60 characters", input: Input{Name: strings.Repeat("é", 60)}},
		{name: "name too long", input: Input{Name: strings.Repeat("é", 61)}, wantCode: NameTooLong, wantLine: -1},
		{name: "servings 50", input: Input{Name: "Chili", Servings: 50}},
		{name: "servings 51", input: Input{Name: "Chili", Servings: 51}, wantCode: InvalidServings, wantLine: -1},
		{name: "negative servings", input: Input{Name: "Chili", Servings: -1}, wantCode: InvalidServings, wantLine: -1},
		{name: "40 lines", input: Input{Name: "Chili", Lines: lines(40)}},
		{name: "41 lines", input: Input{Name: "Chili", Lines: lines(41)}, wantCode: TooManyLines, wantLine: -1},
		{name: "ingredient name required", input: Input{Name: "Chili", Lines: []LineInput{line("sel", "", ""), line(" ", "", "")}}, wantCode: IngredientNameRequired, wantLine: 1},
		{name: "ingredient name of invisible characters", input: Input{Name: "Chili", Lines: []LineInput{line("\u200b\u00a0", "", "")}}, wantCode: IngredientNameRequired, wantLine: 0},
		{name: "ingredient name with a control character", input: Input{Name: "Chili", Lines: []LineInput{line("sel", "", ""), line("poivre\u0085", "", "")}}, wantCode: IngredientNameInvalid, wantLine: 1},
		{name: "ingredient name of 40 characters", input: Input{Name: "Chili", Lines: []LineInput{line(strings.Repeat("é", 40), "", "")}}},
		{name: "ingredient name too long", input: Input{Name: "Chili", Lines: []LineInput{line(strings.Repeat("é", 41), "", "")}}, wantCode: IngredientNameTooLong, wantLine: 0},
		{name: "unknown unit", input: Input{Name: "Chili", Lines: []LineInput{line("riz", "1", "cup")}}, wantCode: InvalidUnit, wantLine: 0},
		{name: "quantity without unit", input: Input{Name: "Chili", Lines: []LineInput{line("riz", "1", "")}}, wantCode: UnitRequired, wantLine: 0},
		{name: "invalid quantity", input: Input{Name: "Chili", Lines: []LineInput{line("riz", "0", "g")}}, wantCode: InvalidQuantity, wantLine: 0},
		{name: "quantity too precise", input: Input{Name: "Chili", Lines: []LineInput{line("lait", "2,5", "ml")}}, wantCode: QuantityTooPrecise, wantLine: 0},
		{
			name:     "same ingredient twice",
			input:    Input{Name: "Ratatouille", Lines: []LineInput{line("courgette", "3", "piece"), line("tomate", "", ""), line(" Courgette", "200", "g")}},
			wantCode: DuplicateIngredient, wantLine: 2, wantIngr: "courgette",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := c.input.validate()
			if c.wantCode == "" {
				if err != nil {
					t.Fatalf("error %v", err)
				}
				return
			}
			var inputErr *InputError
			if !errors.As(err, &inputErr) {
				t.Fatalf("error %v, want %s", err, c.wantCode)
			}
			if inputErr.Code != c.wantCode || inputErr.Line != c.wantLine || inputErr.Ingredient != c.wantIngr {
				t.Errorf("error %+v, want %s on line %d (%q)", inputErr, c.wantCode, c.wantLine, c.wantIngr)
			}
		})
	}
}

func TestDisplayName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Ratatouille", "Ratatouille"},
		{"  Gratin   de  courgettes ", "Gratin de courgettes"},
		{"Soupe\tà\nl’oignon", "Soupe à l’oignon"},
		{"crème\u00a0\u202ffraîche", "crème fraîche"},
		{"Œufs  Brouillés", "Œufs Brouillés"},
		{"thé\u2003vert", "thé\u2003vert"}, // not a space of the normalized name
		{"👩\u200d🍳", "👩\u200d🍳"},
	}
	for _, c := range cases {
		if got := displayName(c.in); got != c.want {
			t.Errorf("displayName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCleanName(t *testing.T) {
	cases := []struct {
		in, want string
		max      int
		fault    NameFault
	}{
		{"  Marché   du samedi ", "Marché du samedi", 40, NameOK},
		{"éééé", "éééé", 4, NameOK},
		{"ééééé", "ééééé", 4, NameLong},
		{" \u200b ", "\u200b", 4, NameEmpty},
		{"", "", 4, NameEmpty},
		// A control character comes first, even in a name too long.
		{"sel\a et poivre", "sel\a et poivre", 4, NameControl},
	}
	for _, c := range cases {
		if got, fault := CleanName(c.in, c.max); got != c.want || fault != c.fault {
			t.Errorf("CleanName(%q, %d) = %q, %d; want %q, %d", c.in, c.max, got, fault, c.want, c.fault)
		}
	}
}

func TestValidateResult(t *testing.T) {
	in := Input{Name: "  Pâtes  au\tbeurre ", Servings: 3, Lines: []LineInput{
		{Name: " pâtes ", Quantity: "500", Unit: "g"},
		{Name: "sel", Unit: "g"}, // a unit without quantity is dropped (Q26)
	}}
	v, err := in.validate()
	if err != nil {
		t.Fatal(err)
	}
	if v.name != "Pâtes au beurre" || v.normalized != "pates au beurre" || v.servings != 3 {
		t.Errorf("dish %+v", v)
	}
	if len(v.lines) != 2 {
		t.Fatalf("lines %+v", v.lines)
	}
	if l := v.lines[0]; l.Name != "pâtes" || l.Normalized != "pates" || l.Quantity == nil || *l.Quantity != (Quantity{500_000, Gram}) {
		t.Errorf("line 0 %+v", l)
	}
	if l := v.lines[1]; l.Name != "sel" || l.Quantity != nil {
		t.Errorf("line 1 %+v", l)
	}
}

// TestNameRules runs the table of cases of a displayed name shared with web/src/dishes.ts
// (D30): the screen refuses before sending what the server refuses.
func TestNameRules(t *testing.T) {
	data, err := os.ReadFile("testdata/names.json")
	if err != nil {
		t.Fatal(err)
	}
	var table struct {
		Cases []struct {
			Text    string `json:"text"`
			Control bool   `json:"control"`
			Visible bool   `json:"visible"`
			Trimmed string `json:"trimmed"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &table); err != nil {
		t.Fatal(err)
	}
	if len(table.Cases) == 0 {
		t.Fatal("no case")
	}
	for _, c := range table.Cases {
		if got := hasControl(displayName(c.Text)); got != c.Control {
			t.Errorf("hasControl(displayName(%q)) = %v, want %v", c.Text, got, c.Control)
		}
		if got := hasVisible(c.Text); got != c.Visible {
			t.Errorf("hasVisible(%q) = %v, want %v", c.Text, got, c.Visible)
		}
		if got := trimName(c.Text); got != c.Trimmed {
			t.Errorf("trimName(%q) = %q, want %q", c.Text, got, c.Trimmed)
		}
	}
}
