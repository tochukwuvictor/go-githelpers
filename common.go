package githelpers

import (
	"path/filepath"
	"strings"
)

func unpack(s []string, vars ...*string) {
	for i, str := range s {
		*vars[i] = str
	}
}

func splitRepoURL(url string) (vcs, ns, name string) {
	var sansVcs string
	unpack(strings.Split(url, ":"), &vcs, &sansVcs) // This will break if http url. Fix.
	fullPath := strings.TrimSuffix(sansVcs, ".git")
	name = filepath.Base(fullPath)
	ns = strings.TrimSuffix(fullPath, "/"+name)

	return vcs, ns, name
}
