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

func TestParseQuantity(t *testing.T) {
	cases := []struct {
		text    string
		unit    Unit
		amount  int64 // in the base unit
		wantErr error
	}{
		{"500", Gram, 500_000, nil},
		{"2,5", Centilitre, 25, nil},
		{"2.5", Centilitre, 25, nil},
		{"0,125", Kilogram, 125_000, nil},
		{"1,5", Piece, 1500, nil},
		{"1,5", Litre, 1500, nil},
		{"0,001", Gram, 1, nil},
		{"0,001", Pinch, 1, nil},
		{"2", Tablespoon, 2000, nil},
		{"2,50", Centilitre, 25, nil},
		{" 600 ", Gram, 600_000, nil},
		{"007", Millilitre, 7, nil},
		{"999999", Kilogram, 999_999_000_000, nil},
		{"2,5", Millilitre, 0, ErrQuantityTooPrecise},
		{"2,55", Centilitre, 0, ErrQuantityTooPrecise},
		{"0,1255", Kilogram, 0, ErrQuantityTooPrecise},
		{"1,0001", Piece, 0, ErrQuantityTooPrecise},
		{"0", Gram, 0, ErrInvalidQuantity},
		{"0,000", Gram, 0, ErrInvalidQuantity},
		{"-1", Gram, 0, ErrInvalidQuantity},
		{"", Gram, 0, ErrInvalidQuantity},
		{"abc", Gram, 0, ErrInvalidQuantity},
		{"1,", Gram, 0, ErrInvalidQuantity},
		{",5", Gram, 0, ErrInvalidQuantity},
		{"1,2,3", Gram, 0, ErrInvalidQuantity},
		{"1 000", Gram, 0, ErrInvalidQuantity},
		{"1e3", Gram, 0, ErrInvalidQuantity},
		{"1000000", Gram, 0, ErrInvalidQuantity},
		{"١٢", Gram, 0, ErrInvalidQuantity},
	}
	for _, c := range cases {
		got, err := ParseQuantity(c.text, c.unit)
		if !errors.Is(err, c.wantErr) {
			t.Errorf("ParseQuantity(%q, %s): error %v, want %v", c.text, c.unit, err, c.wantErr)
			continue
		}
		if err == nil && (got.Amount != c.amount || got.Unit != c.unit) {
			t.Errorf("ParseQuantity(%q, %s) = %+v, want %d %s", c.text, c.unit, got, c.amount, c.unit)
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
	line := func(name, quantity, unit string) LineInput { return LineInput{Name: name, Quantity: quantity, Unit: unit} }
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
		{name: "name of 60 characters", input: Input{Name: strings.Repeat("é", 60)}},
		{name: "name too long", input: Input{Name: strings.Repeat("é", 61)}, wantCode: NameTooLong, wantLine: -1},
		{name: "servings 50", input: Input{Name: "Chili", Servings: 50}},
		{name: "servings 51", input: Input{Name: "Chili", Servings: 51}, wantCode: InvalidServings, wantLine: -1},
		{name: "negative servings", input: Input{Name: "Chili", Servings: -1}, wantCode: InvalidServings, wantLine: -1},
		{name: "40 lines", input: Input{Name: "Chili", Lines: lines(40)}},
		{name: "41 lines", input: Input{Name: "Chili", Lines: lines(41)}, wantCode: TooManyLines, wantLine: -1},
		{name: "ingredient name required", input: Input{Name: "Chili", Lines: []LineInput{line("sel", "", ""), line(" ", "", "")}}, wantCode: IngredientNameRequired, wantLine: 1},
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

func TestValidateResult(t *testing.T) {
	in := Input{Name: "  Pâtes au beurre ", Servings: 3, Lines: []LineInput{
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
	if l := v.lines[0]; l.name != "pâtes" || l.normalized != "pates" || l.quantity == nil || *l.quantity != (Quantity{500_000, Gram}) {
		t.Errorf("line 0 %+v", l)
	}
	if l := v.lines[1]; l.name != "sel" || l.quantity != nil {
		t.Errorf("line 1 %+v", l)
	}
}
