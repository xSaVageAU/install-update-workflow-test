package update

import "golang.org/x/mod/semver"

// IsNewer reports whether latestTag denotes a newer version than current.
// Both are tolerated with or without a leading "v" (GitHub tags use "v1.2.3",
// but a locally built binary's version.Version might be "1.2.3" or "dev").
func IsNewer(current, latestTag string) bool {
	cur, latest := normalize(current), normalize(latestTag)
	if !semver.IsValid(cur) || !semver.IsValid(latest) {
		return false
	}
	return semver.Compare(latest, cur) > 0
}

func normalize(v string) string {
	if v == "" {
		return v
	}
	if v[0] != 'v' {
		return "v" + v
	}
	return v
}
