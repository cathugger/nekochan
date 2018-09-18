package psqlib

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"

	//. "nekochan/lib/logx"
	"nekochan/lib/nntp"
)

//var _ nntp.NNTPProvider = (*PSQLIB)(nil)

type groupState struct {
	bname string
	bid   boardID
	pid   postID
}

type (
	Responder    = nntp.Responder
	FullMsgID    = nntp.FullMsgID
	CoreMsgID    = nntp.CoreMsgID
	FullMsgIDStr = nntp.FullMsgIDStr
	CoreMsgIDStr = nntp.CoreMsgIDStr
	ConnState    = nntp.ConnState
)

func artnumInGroup(cs *ConnState, bid boardID, num uint64) uint64 {
	if cg, _ := cs.CurrentGroup.(*groupState); cg != nil && cg.bid == bid {
		return num
	} else {
		return 0
	}
}

func getGroupState(cs *ConnState) *groupState {
	gs, _ := cs.CurrentGroup.(*groupState)
	return gs
}

func isGroupSelected(gs *groupState) bool {
	return gs != nil && gs.bid != 0
}

/*
func (p *PSQLIB) SupportsNewNews() bool {
	return p.SupportNewNews
}

func (p *PSQLIB) SupportsOverByMsgID() bool {
	return p.SupportOverByMsgID
}

func (p *PSQLIB) SupportsHdr() bool {
	return p.SupportHdr
}

func (p *PSQLIB) SupportsIHave() bool {
	return p.SupportIHave
}

func (p *PSQLIB) SupportsPost() bool {
	return p.SupportPost
}

func (p *PSQLIB) SupportsStream() bool {
	return p.SupportStream
}
*/

func unsafeCoreMsgIDToStr(b CoreMsgID) CoreMsgIDStr {
	return CoreMsgIDStr(unsafeBytesToStr(b))
}

var (
	errNotExist        = os.ErrNotExist
	errNoBoardSelected = errors.New("no board selected")
)

func handleNNTPGetError(w Responder, nc nntpCopyer, e error) bool {
	if e == nil {
		// no error - handled successfuly
		return true
	}
	if !nc.IsClosed() {
		// writer wasn't properly closed -- we should reset connection
		w.Abort()
		return true
	}
	// rest of errors are easier to handle
	if e == errNotExist {
		return false // this is pretty convenient
	} else if e == errNoBoardSelected {
		w.ResNoNewsgroupSelected()
	} else {
		w.ResInternalError(e)
	}
	return true
}

func (sp *PSQLIB) getArticleCommonByMsgID(nc nntpCopyer, w Responder, cs *ConnState, msgid CoreMsgID) bool {
	sid := unsafeCoreMsgIDToStr(msgid)
	e := sp.nntpObtainItemByMsgID(nc, cs, sid)
	return handleNNTPGetError(w, nc, e)
}
func (sp *PSQLIB) GetArticleFullByMsgID(w Responder, cs *ConnState, msgid CoreMsgID) bool {
	nc := &fullNNTPCopyer{w: w}
	return sp.getArticleCommonByMsgID(nc, w, cs, msgid)
}
func (sp *PSQLIB) GetArticleHeadByMsgID(w Responder, cs *ConnState, msgid CoreMsgID) bool {
	nc := &headNNTPCopyer{w: w}
	return sp.getArticleCommonByMsgID(nc, w, cs, msgid)
}
func (sp *PSQLIB) GetArticleBodyByMsgID(w Responder, cs *ConnState, msgid CoreMsgID) bool {
	nc := &bodyNNTPCopyer{w: w}
	return sp.getArticleCommonByMsgID(nc, w, cs, msgid)
}
func (sp *PSQLIB) GetArticleStatByMsgID(w Responder, cs *ConnState, msgid CoreMsgID) bool {
	// notice: not reference
	sc := statNNTPCopyer{w: w}
	return sp.getArticleCommonByMsgID(sc, w, cs, msgid)
}

func (sp *PSQLIB) getArticleCommonByNum(nc nntpCopyer, w Responder, cs *ConnState, num uint64) bool {
	e := sp.nntpObtainItemByNum(nc, cs, num)
	return handleNNTPGetError(w, nc, e)
}
func (sp *PSQLIB) GetArticleFullByNum(w Responder, cs *ConnState, num uint64) bool {
	nc := &fullNNTPCopyer{w: w}
	return sp.getArticleCommonByNum(nc, w, cs, num)
}
func (sp *PSQLIB) GetArticleHeadByNum(w Responder, cs *ConnState, num uint64) bool {
	nc := &headNNTPCopyer{w: w}
	return sp.getArticleCommonByNum(nc, w, cs, num)
}
func (sp *PSQLIB) GetArticleBodyByNum(w Responder, cs *ConnState, num uint64) bool {
	nc := &bodyNNTPCopyer{w: w}
	return sp.getArticleCommonByNum(nc, w, cs, num)
}
func (sp *PSQLIB) GetArticleStatByNum(w Responder, cs *ConnState, num uint64) bool {
	// notice: not reference
	sc := statNNTPCopyer{w: w}
	return sp.getArticleCommonByNum(sc, w, cs, num)
}

func (sp *PSQLIB) getArticleCommonByCurr(nc nntpCopyer, w Responder, cs *ConnState) bool {
	e := sp.nntpObtainItemByCurr(nc, cs)
	return handleNNTPGetError(w, nc, e)
}
func (sp *PSQLIB) GetArticleFullByCurr(w Responder, cs *ConnState) bool {
	nc := &fullNNTPCopyer{w: w}
	return sp.getArticleCommonByCurr(nc, w, cs)
}
func (sp *PSQLIB) GetArticleHeadByCurr(w Responder, cs *ConnState) bool {
	nc := &headNNTPCopyer{w: w}
	return sp.getArticleCommonByCurr(nc, w, cs)
}
func (sp *PSQLIB) GetArticleBodyByCurr(w Responder, cs *ConnState) bool {
	nc := &bodyNNTPCopyer{w: w}
	return sp.getArticleCommonByCurr(nc, w, cs)
}
func (sp *PSQLIB) GetArticleStatByCurr(w Responder, cs *ConnState) bool {
	// notice: not reference
	sc := statNNTPCopyer{w: w}
	return sp.getArticleCommonByCurr(sc, w, cs)
}

func (sp *PSQLIB) SelectGroup(w Responder, cs *ConnState, group []byte) bool {
	sgroup := unsafeBytesToStr(group)

	var bid uint32
	var lo, hi uint64
	q := `SELECT xb.bid,MIN(xp.pid),MAX(xp.pid)
	FROM ib0.boards AS xb
	LEFT JOIN ib0.posts AS xp
	USING (bid)
	WHERE xb.bname = $1
	GROUP BY xb.bid`
	err := sp.db.DB.QueryRow(q, sgroup).Scan(&bid, &lo, &hi)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		w.ResInternalError(sp.sqlError("board-posts row query scan", err))
		return true
	}

	gs := getGroupState(cs)
	if gs == nil {
		gs = &groupState{}
		cs.CurrentGroup = gs
	}
	gs.bid = bid
	if gs.bname != sgroup {
		// sgroup is unsafe
		gs.bname = string(group)
	}
	gs.pid = lo

	if lo != 0 {
		if hi < lo {
			hi = lo
		}
		w.ResGroupSuccessfullySelected(hi-lo+1, lo, hi, sgroup)
	} else {
		w.ResGroupSuccessfullySelected(0, 0, 0, sgroup)
	}

	return true
}
func (sp *PSQLIB) SelectAndListGroup(w Responder, cs *ConnState, group []byte, rmin, rmax int64) bool {
	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		if len(group) == 0 {
			w.ResNoNewsgroupSelected()
			return true
		}
		if gs == nil {
			gs = &groupState{}
			cs.CurrentGroup = gs
		}
	}

	var sgroup string
	if len(group) != 0 {
		sgroup = unsafeBytesToStr(group)
	} else {
		sgroup = gs.bname
	}

	// TODO optimise
	q := `SELECT x1.bid,x2.lo,x2.hi,x3.pid
	FROM ib0.boards AS x1
	LEFT JOIN (
		SELECT xb.bid AS bid,MIN(xp.pid) AS lo,MAX(xp.pid) AS hi
		FROM ib0.boards AS xb
		LEFT JOIN ib0.posts AS xp
		USING (bid)
		WHERE xb.bname = $1
		GROUP BY xb.bid
	) x2
	ON x1.bid = x2.bid
	LEFT JOIN (
		SELECT xb.bid AS bid,xp.pid AS pid
		FROM ib0.boards AS xb
		LEFT JOIN ib0.posts AS xp
		USING (bid)
		WHERE xb.bname = $1
			AND xp.pid >= $2 AND ($3 < 0 OR x3.pid <= $3)
		ORDER BY xp.pid ASC
	) x3
	ON x1.bid = x3.bid
	WHERE x1.bname = $1`
	rows, err := sp.db.DB.Query(q, sgroup)
	if err != nil {
		w.ResInternalError(sp.sqlError("board-posts query", err))
		return true
	}
	var dw io.WriteCloser
	for rows.Next() {
		var bid boardID
		var lo, hi, pid postID
		err = rows.Scan(&bid, &lo, &hi, &pid)
		if err != nil {
			rows.Close()
			err = sp.sqlError("board-post query rows scan", err)
			if dw == nil {
				w.ResInternalError(err)
			} else {
				w.Abort()
			}
			return true
		}
		if dw == nil {
			// we have something. do switch, send info about group
			gs.bid = bid
			if gs.bname != sgroup {
				// sgroup is unsafe
				gs.bname = string(group)
			}
			gs.pid = lo

			if lo != 0 {
				if hi < lo {
					hi = lo
				}
				w.ResArticleNumbersFollow(hi-lo+1, lo, hi, sgroup)
			} else {
				w.ResArticleNumbersFollow(0, 0, 0, sgroup)
			}

			dw = w.DotWriter()
		}

		if pid != 0 {
			fmt.Fprintf(dw, "%d\n", pid)
		}
	}

	if dw != nil {
		dw.Close()
		return true
	} else {
		return false
	}
}
func (sp *PSQLIB) SelectNextArticle(w Responder, cs *ConnState) {
	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		w.ResNoNewsgroupSelected()
		return
	}
	x := gs.pid
	if x == 0 {
		w.ResCurrentArticleNumberIsInvalid()
		return
	}
	var msgid CoreMsgIDStr
	var npid postID
	q := "SELECT pid,msgid FROM ib0.posts WHERE bid = $1 AND pid > $2 LIMIT 1"
	err := sp.db.DB.QueryRow(q, gs.bid, x).Scan(&npid, &msgid)
	if err != nil {
		if err == sql.ErrNoRows {
			w.ResNoNextArticleInThisGroup()
			return
		}
		w.ResInternalError(sp.sqlError("posts row query scan", err))
		return
	}
	gs.pid = npid
	w.ResArticleFound(npid, msgid)
	return
}
func (sp *PSQLIB) SelectPrevArticle(w Responder, cs *ConnState) {
	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		w.ResNoNewsgroupSelected()
		return
	}
	x := gs.pid
	if x == 0 {
		w.ResCurrentArticleNumberIsInvalid()
		return
	}
	var msgid CoreMsgIDStr
	var npid postID
	q := "SELECT pid,msgid FROM ib0.posts WHERE bid = $1 AND pid < $2 LIMIT 1"
	err := sp.db.DB.QueryRow(q, gs.bid, x).Scan(&npid, &msgid)
	if err != nil {
		if err == sql.ErrNoRows {
			w.ResNoPrevArticleInThisGroup()
			return
		}
		w.ResInternalError(sp.sqlError("posts row query scan", err))
		return
	}
	gs.pid = npid
	w.ResArticleFound(npid, msgid)
	return
}

/*

func (p *TestSrv) ListNewGroups(w io.Writer, qt time.Time) {
	for _, gn := range s1.groupsSort {
		g := s1.groups[gn]
		if !qt.After(g.created) {
			lo, hi := g.articlesSort[0], g.articlesSort[len(g.articlesSort)-1]
			fmt.Fprintf(w, "%s %d %d %c\n", gn, hi, lo, g.status)
		}
	}
}

func emptyWildmat(w []byte) bool {
	return len(w) == 0 || (len(w) == 1 && w[0] == '*')
}

func (p *TestSrv) ListNewNews(w io.Writer, wildmat []byte, qt time.Time) {
	var chk func(string) bool
	if emptyWildmat(wildmat) {
		chk = func(g string) bool { return true }
	} else {
		if nntp.ValidGroupSlice(wildmat) {
			sw := unsafeBytesToStr(wildmat)
			chk = func(g string) bool { return g == sw }
		} else {
			wm := nntp.CompileWildmat(wildmat)
			chk = func(g string) bool { return wm.CheckString(g) }
		}
	}

	for id, a := range s1.articles {
		if !qt.After(a.posted) && chk(a.group) {
			fmt.Fprintf(w, "<%s>\n", id)
		}
	}
}

func (p *TestSrv) ListActiveGroups(w io.Writer, wildmat []byte) {
	var chk func(string) bool
	if emptyWildmat(wildmat) {
		chk = func(g string) bool { return true }
	} else {
		if nntp.ValidGroupSlice(wildmat) {
			sw := unsafeBytesToStr(wildmat)
			chk = func(g string) bool { return g == sw }
		} else {
			wm := nntp.CompileWildmat(wildmat)
			chk = func(g string) bool { return wm.CheckString(g) }
		}
	}

	for _, gn := range s1.groupsSort {
		if chk(gn) {
			g := s1.groups[gn]
			lo, hi := g.articlesSort[0], g.articlesSort[len(g.articlesSort)-1]
			fmt.Fprintf(w, "%s %d %d %c\n", gn, hi, lo, g.status)
		}
	}
}

func (p *TestSrv) ListNewsgroups(w io.Writer, wildmat []byte) {
	var chk func(string) bool
	if emptyWildmat(wildmat) {
		chk = func(g string) bool { return true }
	} else {
		if nntp.ValidGroupSlice(wildmat) {
			sw := unsafeBytesToStr(wildmat)
			chk = func(g string) bool { return g == sw }
		} else {
			wm := nntp.CompileWildmat(wildmat)
			chk = func(g string) bool { return wm.CheckString(g) }
		}
	}

	for _, gn := range s1.groupsSort {
		if chk(gn) {
			g := s1.groups[gn]
			fmt.Fprintf(w, "%s\t%s\n", gn, g.info)
		}
	}
}

func printOver(w io.Writer, num uint64, a *article) {
	fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\tXref: %s\n", num,
		a.over.subject, a.over.from, a.over.date, a.over.msgid,
		a.over.references, a.over.bytes, a.over.lines, a.over.xref)
}
*/

// + ok: 224{ResOverviewInformationFollows}
// fail:
//   <ByMsgID>      430{ResNoArticleWithThatMsgID[false]}
//   <OverByRange>  412{ResNoNewsgroupSelected} 423{ResNoArticlesInThatRange[false]}
//   <XOverByRange> 412{ResNoNewsgroupSelected} 420{ResXNoArticles[false]}
//   <ByCurr>       412{ResNoNewsgroupSelected} 420{ResCurrentArticleNumberIsInvalid[false]}
/*
func (p *TestSrv) GetOverByMsgID(w Responder, cs *ConnState, msgid CoreMsgID) bool {
	sid := unsafeCoreMsgIDToStr(msgid)
	a := s1.articles[sid]
	if a == nil {
		return false
	}
*/
/*
	The first 8 fields MUST be the following, in order:
		"0" or article number (see below)
		Subject header content
		From header content
		Date header content
		Message-ID header content
		References header content
		:bytes metadata item
		:lines metadata item
*/
/*
	w.ResOverviewInformationFollows()
	dw := w.DotWriter()
	printOver(dw, artnumInGroup(cs, a.group, a.number), a)
	dw.Close()
	return true
}
*/
/*
func (p *TestSrv) GetOverByRange(w Responder, cs *ConnState, rmin, rmax int64) bool {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return true
	}
	var ww io.WriteCloser = nil
	for _, an := range gs.g.articlesSort {
		if an >= uint64(rmin) && (rmax < 0 || an <= uint64(rmax)) {
			if ww == nil {
				w.ResOverviewInformationFollows()
				ww = w.DotWriter()
			}
			a := gs.g.articles[an]
			printOver(ww, a.number, a)
		}
	}
	if ww != nil {
		ww.Close()
		return true
	} else {
		return false
	}
}
func (p *TestSrv) GetXOverByRange(w Responder, cs *ConnState, rmin, rmax int64) bool {
	return p.GetOverByRange(w, cs, rmin, rmax)
}
func (p *TestSrv) GetOverByCurr(w Responder, cs *ConnState) bool {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return true
	}
	a := gs.g.articles[gs.number]
	if a == nil {
		return false
	}
	w.ResOverviewInformationFollows()
	dw := w.DotWriter()
	printOver(dw, a.number, a)
	dw.Close()
	return true
}

func (p *TestSrv) commonGetHdrByMsgID(w Responder, cs *ConnState, hdr []byte, msgid CoreMsgID, rfc bool) bool {
	sid := unsafeCoreMsgIDToStr(msgid)
	a := s1.articles[sid]
	if a == nil {
		return false
	}
	h, supported := a.over.GetByHdr(hdr)
	if !supported {
		w.PrintfLine("503 %q header unsupported", hdr)
		return true
	}
	if rfc {
		w.ResHdrFollow()
		dw := w.DotWriter()
		fmt.Fprintf(dw, "%d %s\n", artnumInGroup(cs, a.group, a.number), h)
		dw.Close()
	} else {
		w.ResXHdrFollow()
		dw := w.DotWriter()
		fmt.Fprintf(dw, "<%s> %s\n", msgid, h)
		dw.Close()
	}
	return true
}
func (p *TestSrv) commonGetHdrByRange(w Responder, cs *ConnState, hdr []byte, rmin, rmax int64, rfc bool) bool {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return true
	}
	var ww io.WriteCloser = nil
	for _, an := range gs.g.articlesSort {
		if an >= uint64(rmin) && (rmax < 0 || an <= uint64(rmax)) {
			a := gs.g.articles[an]
			h, supported := a.over.GetByHdr(hdr)
			if !supported {
				w.PrintfLine("503 %q header unsupported", hdr)
				return true
			}
			if ww == nil {
				if rfc {
					w.ResHdrFollow()
				} else {
					w.ResXHdrFollow()
				}
				ww = w.DotWriter()
			}
			fmt.Fprintf(ww, "%d %s\n", an, h)
		}
	}
	if ww != nil {
		ww.Close()
		return true
	} else {
		return false
	}
}
func (p *TestSrv) commonGetHdrByCurr(w Responder, cs *ConnState, hdr []byte, rfc bool) bool {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return true
	}
	a := gs.g.articles[gs.number]
	if a == nil {
		return false
	}
	h, supported := a.over.GetByHdr(hdr)
	if !supported {
		w.PrintfLine("503 %q header unsupported", hdr)
		return true
	}
	if rfc {
		w.ResHdrFollow()
	} else {
		w.ResXHdrFollow()
	}
	dw := w.DotWriter()
	fmt.Fprintf(dw, "%d %s\n", a.number, h)
	dw.Close()
	return true
}
func (p *TestSrv) GetHdrByMsgID(w Responder, cs *ConnState, hdr []byte, msgid CoreMsgID) bool {
	return p.commonGetHdrByMsgID(w, cs, hdr, msgid, true)
}
func (p *TestSrv) GetHdrByRange(w Responder, cs *ConnState, hdr []byte, rmin, rmax int64) bool {
	return p.commonGetHdrByRange(w, cs, hdr, rmin, rmax, true)
}
func (p *TestSrv) GetHdrByCurr(w Responder, cs *ConnState, hdr []byte) bool {
	return p.commonGetHdrByCurr(w, cs, hdr, true)
}
func (p *TestSrv) GetXHdrByMsgID(w Responder, hdr []byte, msgid CoreMsgID) bool {
	return p.commonGetHdrByMsgID(w, nil, hdr, msgid, false)
}
func (p *TestSrv) GetXHdrByRange(w Responder, cs *ConnState, hdr []byte, rmin, rmax int64) bool {
	return p.commonGetHdrByRange(w, cs, hdr, rmin, rmax, false)
}
func (p *TestSrv) GetXHdrByCurr(w Responder, cs *ConnState, hdr []byte) bool {
	return p.commonGetHdrByCurr(w, cs, hdr, false)
}
*/
