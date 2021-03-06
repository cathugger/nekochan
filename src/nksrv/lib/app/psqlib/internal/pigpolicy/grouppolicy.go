package pigpolicy

import (
	"errors"
	"strings"
	"unicode"

	"nksrv/lib/nntp"
)

type NewGroupPolicy struct {
	check      nntp.Wildmat // run thru these wildmat filters
	permissive bool
	allowUTF8  bool
}

func MakeNewGroupPolicy(s string) (ngp NewGroupPolicy, err error) {
	allowall := false
	if s == "" {
		// disallow all, nil policy
		return
	}

	if !nntp.ValidWildmatStr(s) {
		err = errors.New("invalid wildmat")
		return
	}

	ss := strings.Split(s, ",")
	for i := len(ss) - 1; i >= 0; i-- {
		// delet explicit ctl, we will add implicit
		if ss[i] == "ctl" {
			ss = append(ss[:i], ss[i+1:]...)
		}
	}
	if len(ss) == 1 && ss[0] == "*" {
		// we won't need implicit stuff
		allowall = true
	}
	// add implicit ctl
	if !allowall {
		ss = append(ss, "ctl")
	}
	// add implicit future-compat blocker
	ss = append(ss, "!*._*")
	// ok done
	s = strings.Join(ss, ",")

	// put it in
	ngp.check = nntp.CompileWildmatStr(s)
	ngp.allowUTF8 = true

	return
}

// if first or the only component equals
func groupFirstCompEq(grp, comp string) bool {
	return strings.HasPrefix(grp, comp) &&
		(len(grp) == len(comp) || grp[len(comp)] == '.')
}

// if any of components equal
func groupAnyCompEq(grp, comp string) bool {
	off := 0
	for {
		i := strings.Index(grp[off:], comp)
		if i < 0 {
			return false
		}
		i += off
		if (i == 0 || grp[i-1] == '.') &&
			(i+len(comp) == len(grp) || grp[i+len(comp)] == '.') {

			return true
		}
		off = i + 1
	}
}

func validNetNewsGroup(s string, allowUTF8 bool) bool {
	// already assumes valid UTF-8 string
	/*
		{RFC 5536}
		   newsgroup-name  =  component *( "." component )
		   component       =  1*component-char
		   component-char  =  ALPHA / DIGIT / "+" / "-" / "_"
	*/
	for i, c := range s {
		if c == '.' {
			if i == 0 || s[i-1] == '.' || i+1 == len(s) {
				return false
			}
			continue
		}
		if (c >= '0' && c <= '9') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= 'a' && c <= 'z') ||
			c == '+' || c == '-' || c == '_' ||
			(c >= 0x80 && allowUTF8 && unicode.IsPrint(c)) {

			continue
		}
		return false
	}
	/*
	   The following <newsgroup-name>s are reserved and MUST NOT be used as
	   the name of a newsgroup:
	   o  Groups whose first (or only) <component> is "example"
	   o  The group "poster"
	*/
	if groupFirstCompEq(s, "example") || s == "poster" {
		return false
	}
	// "to[.*]" are used for UUCP ihave iirc
	if groupFirstCompEq(s, "to") {
		return false
	}
	// "any" is wildcard in some impls
	if groupAnyCompEq(s, "any") {
		return false
	}

	return true
}

func validFuckyGroup(s string, allowUTF8 bool) bool {
	for _, c := range s {
		if c < 0x80 {
			// it should already be checked against control chars I think
			continue
		}
		if !allowUTF8 || !unicode.IsPrint(c) {
			return false
		}
	}
	return true
}

func (ngp NewGroupPolicy) CheckGroup(group string) bool {
	// NOTE: it is already assumed that s is valid UTF-8 at this point
	if !ngp.check.CheckString(group) {
		return false
	}
	// some additional checks to prevent fucky groups being autoadded
	if !ngp.permissive {
		if !validNetNewsGroup(group, ngp.allowUTF8) {
			return false
		}
	} else {
		if !validFuckyGroup(group, ngp.allowUTF8) {
			return false
		}
	}
	return true
}
