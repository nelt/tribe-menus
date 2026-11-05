// Package mealplan holds the meal plan of a tribe (R1 to R5): pure rules, and the Plan behind
// a Store interface (ADR 0002). The server knows no "today": it accepts any real date of the
// bounds, past or future (D4).
package mealplan

import (
	"errors"
	"time"

	"github.com/nelt/tribe-menus/internal/dish"
)

// Bounds of a meal (D4, Q35), and of the servings of a served dish: those of a dish.
const (
	FirstDate   Date = "2000-01-01"
	LastDate    Date = "2099-12-31"
	MinServings      = dish.MinServings
	MaxServings      = dish.MaxServings
)

// dateLayout is the format of a date of the API and the database: AAAA-MM-JJ, without
// time zone.
const dateLayout = "2006-01-02"

// Date is a day of the meal plan, as AAAA-MM-JJ.
type Date string

// Mealtime is the moment of a meal (Q1).
type Mealtime string

// Mealtimes.
const (
	Lunch  Mealtime = "lunch"
	Dinner Mealtime = "dinner"
)

var (
	// ErrInvalidDate is returned for a text that is not a real date of the bounds.
	ErrInvalidDate = errors.New("invalid date")
	// ErrInvalidMealtime is returned for a mealtime other than lunch or dinner.
	ErrInvalidMealtime = errors.New("invalid mealtime")
	// ErrInvalidServings is returned for servings out of 1 to 50.
	ErrInvalidServings = errors.New("invalid servings")
)

// ParseDate reads a date AAAA-MM-JJ: a real date (no 30 February), from 2000-01-01 to
// 2099-12-31 (D4).
func ParseDate(text string) (Date, error) {
	t, err := time.Parse(dateLayout, text)
	if err != nil || t.Format(dateLayout) != text {
		return "", ErrInvalidDate
	}
	d := Date(text)
	if d < FirstDate || d > LastDate {
		return "", ErrInvalidDate
	}
	return d, nil
}

// ParseMealtime reads a mealtime: lunch or dinner.
func ParseMealtime(text string) (Mealtime, error) {
	switch m := Mealtime(text); m {
	case Lunch, Dinner:
		return m, nil
	}
	return "", ErrInvalidMealtime
}

// CheckServings checks the servings of a served dish: from 1 to 50 (Q35).
func CheckServings(servings int) error {
	if servings < MinServings || servings > MaxServings {
		return ErrInvalidServings
	}
	return nil
}
