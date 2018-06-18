package nntp

import (
	"io"
	"time"
)

type FullMsgID []byte // msgid with < and >
type CutMsgID []byte  // msgid without < and >

type ArticleReader interface {
	io.Reader
	ReadByte() (byte, error)
	Discard(n int) (int, error)
}

type ReaderOpener interface {
	OpenReader() ArticleReader
}

type NNTPProvider interface {
	SupportsNewNews() bool
	SupportsOverByMsgID() bool
	SupportsIHave() bool
	SupportsPost() bool
	SupportsStream() bool

	// ARTICLE, HEAD, BODY, STAT x 3 forms for each
	// ok: ARTICLE - 220, HEAD - 221, BODY - 222, STAT - 223
	// fail:
	//   1st_form: 430 (not found by msgid)
	//   2nd_form: 412 (no group selected), 423 (not found by num)
	//   3rd_form: 412 (no group selected), 420 (not found by curr)
	GetArticleFullByMsgID(w Responder, msgid CutMsgID) bool
	GetArticleHeadByMsgID(w Responder, msgid CutMsgID) bool
	GetArticleBodyByMsgID(w Responder, msgid CutMsgID) bool
	GetArticleStatByMsgID(w Responder, msgid CutMsgID) bool
	GetArticleFullByNum(w Responder, cs *ConnState, num uint64) bool
	GetArticleHeadByNum(w Responder, cs *ConnState, num uint64) bool
	GetArticleBodyByNum(w Responder, cs *ConnState, num uint64) bool
	GetArticleStatByNum(w Responder, cs *ConnState, num uint64) bool
	GetArticleFullByCurr(w Responder, cs *ConnState) bool
	GetArticleHeadByCurr(w Responder, cs *ConnState) bool
	GetArticleBodyByCurr(w Responder, cs *ConnState) bool
	GetArticleStatByCurr(w Responder, cs *ConnState) bool

	SelectGroup(w Responder, cs *ConnState, group []byte) bool
	SelectAndListGroup(w Responder, cs *ConnState, group []byte, rmin, rmax int64) bool
	SelectNextArticle(w Responder, cs *ConnState)
	SelectPrevArticle(w Responder, cs *ConnState)

	ListNewGroups(w io.Writer, qt time.Time)
	ListNewNews(w io.Writer, wildmat []byte, qt time.Time) // SupportsNewNews()
	ListActiveGroups(w io.Writer, wildmat []byte)
	ListNewsgroups(w io.Writer, wildmat []byte)

	GetOverByMsgID(w Responder, msgid CutMsgID) bool // SupportsOverByMsgID()
	GetOverByRange(w Responder, cs *ConnState, rmin, rmax int64) bool
	GetOverByCurr(w Responder, cs *ConnState) bool

	// implementers MUST drain readers or bad things will happen
	HandlePost(w Responder, cs *ConnState, ro ReaderOpener) bool                  // SupportsIHave()
	HandleIHave(w Responder, cs *ConnState, ro ReaderOpener, msgid CutMsgID) bool // SupportsPost()
	HandleCheck(w Responder, cs *ConnState, msgid CutMsgID) bool                  // SupportsStream()
	HandleTakeThis(w Responder, cs *ConnState, r ArticleReader, msgid CutMsgID)   // SupportsStream()
}
