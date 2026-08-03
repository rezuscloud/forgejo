package forgejo

import "strings"

// GiteaAPIVersion extracts the Gitea REST API compatibility version from a
// Forgejo version string of the form "16.0.0-rc.1+gitea-1.22.0" (as returned by
// the live server /version endpoint or recorded in a swagger info.version).
//
// It returns the segment after the "+gitea-" suffix (here "1.22.0"), or the
// empty string when the suffix is absent. This is the API "major" level that
// kubectl-style clients compare to detect client/server skew.
func GiteaAPIVersion(v string) string {
	if i := strings.LastIndex(v, "+gitea-"); i >= 0 {
		return v[i+len("+gitea-"):]
	}
	return ""
}
