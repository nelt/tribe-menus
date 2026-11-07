// Package dish holds the dish library and the ingredient catalog of a tribe (P1 to P4):
// pure rules, and the Library behind a Store interface (ADR 0002).
package dish

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// NormalizeName returns the normalized name (Q29, ADR 0027): the identity of an ingredient
// and of the name of a dish. Same steps as normalizeName in web/src/dishes.ts, kept in
// agreement by testdata/normalized-names.json:
//  1. lower case first, so that the next steps only know lower case;
//  2. ligatures œ and æ unfolded;
//  3. NFD decomposition, then only the combining marks U+0300 to U+036F removed;
//  4. typographic apostrophes (’ ‘ ʼ) brought back to ';
//  5. spaces (ASCII spaces, U+00A0, U+202F) trimmed, and runs of them reduced to one space.
func NormalizeName(name string) string {
	s := strings.ToLower(name)
	s = ligatures.Replace(s)
	s = norm.NFD.String(s)
	var b strings.Builder
	space := false
	for _, r := range s {
		switch {
		case r >= 0x0300 && r <= 0x036F:
			continue
		case isSpace(r):
			space = b.Len() > 0
			continue
		case r == '’' || r == '‘' || r == 'ʼ':
			r = '\''
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		b.WriteRune(r)
	}
	return b.String()
}

var ligatures = strings.NewReplacer("œ", "oe", "æ", "ae")

// isSpace: the spaces of the normalized name, the same set in TypeScript.
func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r', ' ', ' ':
		return true
	}
	return false
}

// trimName removes the spaces of the normalized name around a text entered.
func trimName(name string) string {
	return strings.TrimFunc(name, isSpace)
}

// displayName is the name kept as entered, its spaces reduced as in the normalized name
// (step 5), case, accents and apostrophes kept (Q31, plan bibliotheque, D30).
func displayName(name string) string {
	var b strings.Builder
	space := false
	for _, r := range name {
		if isSpace(r) {
			space = b.Len() > 0
			continue
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		b.WriteRune(r)
	}
	return b.String()
}

// hasControl tells whether a displayed name holds a control character (category Cc), its
// spaces being already reduced; format characters (Cf), such as the joiner of composed
// emojis, are allowed (D30).
func hasControl(name string) bool {
	return strings.IndexFunc(name, func(r rune) bool { return unicode.Is(unicode.Cc, r) }) >= 0
}

// hasVisible tells whether a name holds a visible character: a letter, a digit, a
// punctuation mark or a symbol (D30). A name without any is refused as an empty one.
func hasVisible(name string) bool {
	return strings.IndexFunc(name, func(r rune) bool {
		return unicode.In(r, unicode.L, unicode.N, unicode.P, unicode.S)
	}) >= 0
}

// NameFault tells why a name entered breaks the rules of the names (D30).
type NameFault int

// Faults of a name.
const (
	NameOK NameFault = iota
	// NameEmpty: no visible character.
	NameEmpty
	// NameControl: a control character.
	NameControl
	// NameLong: more characters than allowed.
	NameLong
)

// CleanName returns a name entered with its spaces reduced, and why it breaks the rules of
// the names (D30), if it does: the names of the dishes and of the ingredients, and of the
// shopping lists (Q38). Max is the number of characters allowed.
func CleanName(name string, max int) (string, NameFault) {
	name = displayName(name)
	switch {
	case hasControl(name):
		return name, NameControl
	case !hasVisible(name):
		return name, NameEmpty
	case utf8.RuneCountInString(name) > max:
		return name, NameLong
	}
	return name, NameOK
}

// Unit is a unit of the closed list (glossary), by its code in the API and the database.
type Unit string

// Units.
const (
	Gram       Unit = "g"
	Kilogram   Unit = "kg"
	Millilitre Unit = "ml"
	Centilitre Unit = "cl"
	Litre      Unit = "l"
	Piece      Unit = "piece"
	Tablespoon Unit = "tablespoon"
	Teaspoon   Unit = "teaspoon"
	Pinch      Unit = "pinch"
)

// Family is a family of units: quantities convert within a family only.
type Family string

// Families of units.
const (
	Mass             Family = "mass"
	Volume           Family = "volume"
	PieceFamily      Family = "piece"
	TablespoonFamily Family = "tablespoon"
	TeaspoonFamily   Family = "teaspoon"
	PinchFamily      Family = "pinch"
)

// ErrUnknownUnit is returned for a code out of the closed list.
var ErrUnknownUnit = errors.New("unknown unit")

type unitInfo struct {
	family Family
	// factor is the number of base units in one unit: the base unit is the milligram for
	// masses, the millilitre for volumes, the thousandth otherwise (ADR 0003, point 7).
	factor int64
	// decimals is the number of decimals a quantity entered in this unit may have (Q31).
	decimals int
}

var units = map[Unit]unitInfo{
	Gram:       {Mass, 1000, 3},
	Kilogram:   {Mass, 1_000_000, 3},
	Millilitre: {Volume, 1, 0},
	Centilitre: {Volume, 10, 1},
	Litre:      {Volume, 1000, 3},
	Piece:      {PieceFamily, 1000, 3},
	Tablespoon: {TablespoonFamily, 1000, 3},
	Teaspoon:   {TeaspoonFamily, 1000, 3},
	Pinch:      {PinchFamily, 1000, 3},
}

// ParseUnit returns the unit with this code, or ErrUnknownUnit.
func ParseUnit(code string) (Unit, error) {
	if _, ok := units[Unit(code)]; !ok {
		return "", ErrUnknownUnit
	}
	return Unit(code), nil
}

// Family returns the family of the unit.
func (u Unit) Family() Family {
	return units[u].family
}

// Quantity is an amount in the base unit of the family of its unit, and the unit chosen for
// the line, in which it is displayed.
type Quantity struct {
	Amount int64
	Unit   Unit
}

var (
	// ErrInvalidQuantity is returned for a quantity unreadable, zero or negative.
	ErrInvalidQuantity = errors.New("invalid quantity")
	// ErrQuantityTooPrecise is returned for a quantity with more decimals than its unit
	// allows (Q31): refused rather than rounded without a word.
	ErrQuantityTooPrecise = errors.New("quantity too precise for its unit")
)

// quantityPattern: digits, then a comma or a dot and digits. At most 6 digits before the
// decimal separator, leading zeros aside, so that the amount fits in an int64.
var quantityPattern = regexp.MustCompile(`^([0-9]+)(?:[.,]([0-9]+))?$`)

const maxIntegerDigits = 6

// ParseQuantity reads a quantity entered in a unit, with a comma or a dot (Q31).
func ParseQuantity(text string, u Unit) (Quantity, error) {
	info, ok := units[u]
	if !ok {
		return Quantity{}, ErrUnknownUnit
	}
	m := quantityPattern.FindStringSubmatch(strings.TrimFunc(text, isSpace))
	if m == nil {
		return Quantity{}, ErrInvalidQuantity
	}
	whole, fraction := strings.TrimLeft(m[1], "0"), strings.TrimRight(m[2], "0")
	if len(whole) > maxIntegerDigits {
		return Quantity{}, ErrInvalidQuantity
	}
	if len(fraction) > info.decimals {
		return Quantity{}, ErrQuantityTooPrecise
	}
	amount, _ := strconv.ParseInt("0"+whole, 10, 64) // at most 6 digits
	amount *= info.factor
	if fraction != "" {
		f, _ := strconv.ParseInt(fraction, 10, 64) // at most 3 digits
		scale := info.factor
		for range fraction {
			scale /= 10
		}
		amount += f * scale
	}
	if amount <= 0 {
		return Quantity{}, ErrInvalidQuantity
	}
	return Quantity{Amount: amount, Unit: u}, nil
}

// Text returns the quantity in its unit, with a dot and without useless zeros ("2.5",
// "0.125"): ParseQuantity reads it back. Displaying it is up to the front end.
func (q Quantity) Text() string {
	factor := units[q.Unit].factor
	text := strconv.FormatInt(q.Amount/factor, 10)
	rest := q.Amount % factor
	if rest == 0 {
		return text
	}
	digits := len(strconv.FormatInt(factor, 10)) - 1
	fraction := strconv.FormatInt(rest, 10)
	fraction = strings.Repeat("0", digits-len(fraction)) + fraction
	return text + "." + strings.TrimRight(fraction, "0")
}

// Color is a hue of the palette.
type Color string

// Palette holds the hues of the dishes, in the order of docs/design/README.md (Q19).
var Palette = []Color{"#C2502E", "#3F7F25", "#2F6FB0", "#B0306A", "#B36A0B", "#8A45B3", "#2E8A6E", "#9C7400", "#6E6680"}

// PickColor returns the hue of the palette least used by the dishes counted, the first in
// palette order on a tie (Q19).
func PickColor(counts map[Color]int) Color {
	best := Palette[0]
	for _, c := range Palette[1:] {
		if counts[c] < counts[best] {
			best = c
		}
	}
	return best
}
