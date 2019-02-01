package fileutil

import (
	"strings"

	au "centpd/lib/asciiutils"
)

// SafeExt cuts out safe extension from file name.
// If file name contains no extension, or extension contains
// undesired characters, SafeExt returns empty string.
func SafeExt(oname string) (ext string) {
	i := strings.LastIndexByte(oname, '.')
	if i >= 0 && i+1 < len(oname) {
		s := oname[i+1:]

		if /* 32 chars should be long enough */ len(s) <= 32 &&
			// this blacklist is sorta opinionated
			strings.IndexAny(s, "/\\&'`,;=*\"?<>|") < 0 &&
			// probably no one really use anything other than US-ASCII for exts
			au.IsPrintableASCIIStr(s, 0) {

			// twitter' :orig and similar shit, shouldn't change content type
			if j := strings.IndexByte(s, ':'); j >= 0 {
				s = s[:j]
			}

			ext = s
		}
	}
	return
}
