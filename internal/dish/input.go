package dish

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

// Limits (Q31).
const (
	MaxNameLength           = 60
	MaxIngredientNameLength = 40
	MaxLines                = 40
	MinServings             = 1
	MaxServings             = 50
)

// Input is a dish as entered: created, or saved after a change.
type Input struct {
	Name string
	// Servings are the reference servings; 0 on a creation gives the tribe size (T3).
	Servings int
	Lines    []LineInput
}

// LineInput is a line as entered: the ingredient by its name (Q20), and an optional quantity
// as typed, with its unit code.
type LineInput struct {
	Name     string
	Quantity string
	Unit     string
}

// ErrorCode tells why a dish is refused, in the "error" field of the API.
type ErrorCode string

// Error codes of a dish refused (docs/design/README.md, « Messages »).
const (
	NameRequired           ErrorCode = "name_required"
	NameTooLong            ErrorCode = "name_too_long"
	NameTaken              ErrorCode = "name_taken"
	InvalidServings        ErrorCode = "invalid_servings"
	TooManyLines           ErrorCode = "too_many_lines"
	IngredientNameRequired ErrorCode = "ingredient_name_required"
	IngredientNameTooLong  ErrorCode = "ingredient_name_too_long"
	InvalidUnit            ErrorCode = "invalid_unit"
	UnitRequired           ErrorCode = "unit_required"
	InvalidQuantity        ErrorCode = "invalid_quantity"
	QuantityTooPrecise     ErrorCode = "quantity_too_precise"
	DuplicateIngredient    ErrorCode = "duplicate_ingredient"
)

// InputError is a dish refused: nothing is saved.
type InputError struct {
	Code ErrorCode
	// Line is the index of the line in cause, -1 for the dish itself.
	Line int
	// Ingredient is the name of the ingredient already in the dish, for DuplicateIngredient.
	Ingredient string
}

func (e *InputError) Error() string {
	if e.Line < 0 {
		return fmt.Sprintf("dish refused: %s", e.Code)
	}
	return fmt.Sprintf("dish refused: %s on line %d", e.Code, e.Line)
}

func dishError(code ErrorCode) *InputError { return &InputError{Code: code, Line: -1} }

// validInput is an input that follows the rules of a dish.
type validInput struct {
	name, normalized string
	// servings is 0 when the input leaves them to the tribe size.
	servings int
	lines    []validLine
}

type validLine struct {
	name, normalized string
	quantity         *Quantity
}

// validate checks the rules that depend on the input alone (P1, Q26, Q27, Q31); the name
// taken (Q28) is checked against the library. A unit without quantity is dropped: without
// quantity, no unit (Q26).
func (in Input) validate() (validInput, error) {
	v := validInput{name: trimName(in.Name), servings: in.Servings}
	v.normalized = NormalizeName(v.name)
	switch {
	case v.normalized == "":
		return v, dishError(NameRequired)
	case utf8.RuneCountInString(v.name) > MaxNameLength:
		return v, dishError(NameTooLong)
	case in.Servings != 0 && (in.Servings < MinServings || in.Servings > MaxServings):
		return v, dishError(InvalidServings)
	case len(in.Lines) > MaxLines:
		return v, dishError(TooManyLines)
	}
	seen := map[string]string{} // display name by normalized name
	for i, l := range in.Lines {
		line, code := validateLine(l)
		if code != "" {
			return v, &InputError{Code: code, Line: i}
		}
		if first, ok := seen[line.normalized]; ok {
			return v, &InputError{Code: DuplicateIngredient, Line: i, Ingredient: first}
		}
		seen[line.normalized] = line.name
		v.lines = append(v.lines, line)
	}
	return v, nil
}

func validateLine(l LineInput) (validLine, ErrorCode) {
	line := validLine{name: trimName(l.Name)}
	line.normalized = NormalizeName(line.name)
	switch {
	case line.normalized == "":
		return line, IngredientNameRequired
	case utf8.RuneCountInString(line.name) > MaxIngredientNameLength:
		return line, IngredientNameTooLong
	case trimName(l.Quantity) == "":
		return line, ""
	case l.Unit == "":
		return line, UnitRequired
	}
	unit, err := ParseUnit(l.Unit)
	if err != nil {
		return line, InvalidUnit
	}
	q, err := ParseQuantity(l.Quantity, unit)
	switch {
	case errors.Is(err, ErrQuantityTooPrecise):
		return line, QuantityTooPrecise
	case err != nil:
		return line, InvalidQuantity
	}
	line.quantity = &q
	return line, ""
}
