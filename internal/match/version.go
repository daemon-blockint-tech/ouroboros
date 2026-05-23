// Package match implements version comparison helpers for exposure catalogs.
package match

import (
	"strings"

	"golang.org/x/mod/semver"
)

// VersionMatches reports whether installed version satisfies any entry in wants.
// Each want is either an exact version or a semver constraint prefix:
//   - ">=1.2.3", "<=2.0.0", ">1.0.0", "<3.0.0", "=1.2.3"
//
// Non-semver versions (editor extensions, empty) only match on exact equality.
func VersionMatches(wants []string, installed string) (matched string, ok bool) {
	installed = strings.TrimSpace(installed)
	if installed == "" {
		return "", false
	}
	instV := semver.Canonical("v" + strings.TrimPrefix(installed, "v"))

	for _, w := range wants {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		if w == installed {
			return w, true
		}
		if len(w) >= 2 && (w[0] == '>' || w[0] == '<' || w[0] == '=') {
			if instV == "" || !semver.IsValid(instV) {
				continue
			}
			if constraintSatisfied(w, instV) {
				return w, true
			}
		}
	}
	return "", false
}

func constraintSatisfied(constraint, instV string) bool {
	// Single comparator only (v0.2); compound ranges can be added later.
	op, rest := splitComparator(constraint)
	if rest == "" {
		return false
	}
	wantV := semver.Canonical("v" + strings.TrimPrefix(rest, "v"))
	if !semver.IsValid(wantV) {
		return false
	}
	cmp := semver.Compare(instV, wantV)
	switch op {
	case ">=":
		return cmp >= 0
	case "<=":
		return cmp <= 0
	case ">":
		return cmp > 0
	case "<":
		return cmp < 0
	case "=":
		return cmp == 0
	default:
		return false
	}
}

func splitComparator(s string) (op, ver string) {
	switch {
	case strings.HasPrefix(s, ">="):
		return ">=", strings.TrimSpace(s[2:])
	case strings.HasPrefix(s, "<="):
		return "<=", strings.TrimSpace(s[2:])
	case strings.HasPrefix(s, ">"):
		return ">", strings.TrimSpace(s[1:])
	case strings.HasPrefix(s, "<"):
		return "<", strings.TrimSpace(s[1:])
	case strings.HasPrefix(s, "="):
		return "=", strings.TrimSpace(s[1:])
	default:
		return "", s
	}
}
