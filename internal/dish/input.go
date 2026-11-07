package dish

import (
	"errors"
	"fmt"
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
	NameInvalid            ErrorCode = "name_invalid"
	NameTaken              ErrorCode = "name_taken"
	InvalidServings        ErrorCode = "invalid_servings"
	TooManyLines           ErrorCode = "too_many_lines"
	IngredientNameRequired ErrorCode = "ingredient_name_required"
	IngredientNameTooLong  ErrorCode = "ingredient_name_too_long"
	IngredientNameInvalid  ErrorCode = "ingredient_name_invalid"
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

// Error codes of the faults of a name of a dish and of an ingredient.
var (
	dishNameErrors       = map[NameFault]ErrorCode{NameControl: NameInvalid, NameEmpty: NameRequired, NameLong: NameTooLong}
	ingredientNameErrors = map[NameFault]ErrorCode{NameControl: IngredientNameInvalid, NameEmpty: IngredientNameRequired, NameLong: IngredientNameTooLong}
)

// validInput is an input that follows the rules of a dish.
type validInput struct {
	name, normalized string
	// servings is 0 when the input leaves them to the tribe size.
	servings int
	lines    []ParsedLine
}

// ParsedLine is a line as entered, checked alone: the name of its ingredient with its spaces
// reduced (D30), its normalized name (Q29), and its quantity, if any.
type ParsedLine struct {
	Name, Normalized string
	Quantity         *Quantity
}

// validate checks the rules that depend on the input alone (P1, Q26, Q27, Q31); the name
// taken (Q28) is checked against the library. Names are kept with their spaces reduced
// (D30).
func (in Input) validate() (validInput, error) {
	v := validInput{servings: in.Servings}
	var fault NameFault
	v.name, fault = CleanName(in.Name, MaxNameLength)
	v.normalized = NormalizeName(v.name)
	switch {
	case fault != NameOK:
		return v, dishError(dishNameErrors[fault])
	case in.Servings != 0 && (in.Servings < MinServings || in.Servings > MaxServings):
		return v, dishError(InvalidServings)
	case len(in.Lines) > MaxLines:
		return v, dishError(TooManyLines)
	}
	seen := map[string]string{} // display name by normalized name
	for i, l := range in.Lines {
		line, code := ParseLine(l)
		if code != "" {
			return v, &InputError{Code: code, Line: i}
		}
		if first, ok := seen[line.Normalized]; ok {
			return v, &InputError{Code: DuplicateIngredient, Line: i, Ingredient: first}
		}
		seen[line.Normalized] = line.Name
		v.lines = append(v.lines, line)
	}
	return v, nil
}

// ParseLine checks a line alone (Q26, Q31): a line of a dish, or an item added by hand to a
// shopping list, which follows the same rules (Q13). A unit without quantity is dropped:
// without quantity, no unit (Q26).
func ParseLine(l LineInput) (ParsedLine, ErrorCode) {
	var line ParsedLine
	var fault NameFault
	line.Name, fault = CleanName(l.Name, MaxIngredientNameLength)
	line.Normalized = NormalizeName(line.Name)
	switch {
	case fault != NameOK:
		return line, ingredientNameErrors[fault]
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
	line.Quantity = &q
	return line, ""
}
