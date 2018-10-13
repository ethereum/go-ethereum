package validation

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func makeSuggestion(prefix string, options []string, input string) string {
	var selected []string
	distances := make(map[string]int)
	for _, opt := range options {
		distance := levenshteinDistance(input, opt)
		threshold := max(len(input)/2, max(len(opt)/2, 1))
		if distance < threshold {
			selected = append(selected, opt)
			distances[opt] = distance
		}
	}

	if len(selected) == 0 {
		return ""
	}
	sort.Slice(selected, func(i, j int) bool {
		return distances[selected[i]] < distances[selected[j]]
	})

	parts := make([]string, len(selected))
	for i, opt := range selected {
		parts[i] = strconv.Quote(opt)
	}
	if len(parts) > 1 {
		parts[len(parts)-1] = "or " + parts[len(parts)-1]
	}
	return fmt.Sprintf(" %s %s?", prefix, strings.Join(parts, ", "))
}

func levenshteinDistance(s1, s2 string) int {
	column := make([]int, len(s1)+1)
	for y := range s1 {
		column[y+1] = y + 1
	}
	for x, rx := range s2 {
		column[0] = x + 1
		lastdiag := x
		for y, ry := range s1 {
			olddiag := column[y+1]
			if rx != ry {
				lastdiag++
			}
			column[y+1] = min(column[y+1]+1, min(column[y]+1, lastdiag))
			lastdiag = olddiag
		}
	}
	return column[len(s1)]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
