package main

import (
	"slices"
	"time"
)

// stats are the median and quartiles of a series of times.
type stats struct {
	n              int
	q1, median, q3 time.Duration
}

func summarize(d []time.Duration) stats {
	if len(d) == 0 {
		return stats{}
	}
	s := slices.Sorted(slices.Values(d))
	return stats{n: len(s), q1: quantile(s, 0.25), median: quantile(s, 0.5), q3: quantile(s, 0.75)}
}

// quantile interpolates between the two closest values of a sorted series.
func quantile(sorted []time.Duration, q float64) time.Duration {
	pos := q * float64(len(sorted)-1)
	i := int(pos)
	if i+1 >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	frac := pos - float64(i)
	return sorted[i] + time.Duration(frac*float64(sorted[i+1]-sorted[i]))
}

// superiority is the probability that a time of x exceeds a time of y, ties counting half
// (the statistic of the Mann-Whitney test, divided by the number of pairs): 0.5 when the
// times do not tell the cases apart, 0 or 1 when they always do.
func superiority(x, y []time.Duration) float64 {
	if len(x) == 0 || len(y) == 0 {
		return 0.5
	}
	var wins float64
	for _, a := range x {
		for _, b := range y {
			switch {
			case a > b:
				wins++
			case a == b:
				wins += 0.5
			}
		}
	}
	return wins / float64(len(x)*len(y))
}
