package minimail

import (
	"io"

	au "nksrv/lib/asciiutils"
)

// some of types put in small package so that nntp won't need to pull in whole mail

type FullMsgID []byte // msgid including < and >
type CoreMsgID []byte // msgid excluding < and >
type FullMsgIDStr string
type CoreMsgIDStr string

type ArticleReader interface {
	io.Reader
	ReadByte() (byte, error)
	Discard(n int) (int, error)
	InvalidNL() bool
}

func CutMessageIDStr(id FullMsgIDStr) CoreMsgIDStr {
	return CoreMsgIDStr(id[1 : len(id)-1])
}

func ValidMessageIDStr(id FullMsgIDStr) bool {
	return len(id) >= 3 &&
		id[0] == '<' && id[len(id)-1] == '>' && len(id) <= 250 &&
		au.IsPrintableASCIIStr(string(CutMessageIDStr(id)), '>')
}