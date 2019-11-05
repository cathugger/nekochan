package psqlib

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lib/pq"

	. "nksrv/lib/logx"
	"nksrv/lib/mailib"
)

const postTQMsgArgCount = 16
const postTQFileArgCount = 8

func (sp *PSQLIB) getNTStmt(n int) (s *sql.Stmt, err error) {
	sp.ntMutex.RLock()
	s = sp.ntStmts[n]
	sp.ntMutex.RUnlock()

	if s != nil {
		return
	}

	sp.ntMutex.Lock()
	defer sp.ntMutex.Unlock()

	// there couldve been race so re-examine situation
	s = sp.ntStmts[n]
	if s != nil {
		return
	}
	// confirmed no statement is there yet.
	// create new
	var st string
	// header
	sth := `WITH
	ugp AS (
		INSERT INTO
			ib0.gposts
			(
				date_sent,     -- 1
				date_recv,     -- NOW()
				sage,          -- FALSE
				f_count,       -- 2
				msgid,         -- 3
				title,         -- 4
				author,        -- 5
				trip,          -- 6
				message,       -- 7
				headers,       -- 8
				attrib,        -- 9
				layout,        -- 10
				extras         -- 11
			)
		VALUES
			(
				$1,        -- date_sent
				NOW(),     -- date_recv
				FALSE,     -- sage
				$2,        -- f_count
				$3,        -- msgid
				$4,        -- title
				$5,        -- author
				$6,        -- trip
				$7,        -- message
				$8,        -- headers
				$9,        -- attrib
				$10,       -- layout
				$11        -- extras
			)
		RETURNING
			g_p_id,
			date_sent,
			date_recv
	),
	ub AS (
		UPDATE
			ib0.boards
		SET
			last_id = last_id + 1,
			t_count = t_count + 1,
			p_count = p_count + 1
		WHERE
			b_id = $12
		RETURNING
			last_id
	),
	ut AS (
		INSERT INTO
			ib0.threads (
				b_id,
				b_t_id,
				g_t_id,
				b_t_name,
				bump,
				p_count,
				f_count,
				fr_count,
				skip_over
			)
		SELECT
			$12,        -- b_id
			ub.last_id, -- b_t_id
			ugp.g_p_id, -- g_t_id
			$13,        -- b_t_name
			$1,         -- date_sent
			1,          -- p_count
			$2,         -- f_count
			0,          -- fr_count
			$14         -- skip_over
		FROM
			ub
		CROSS JOIN
			ugp
	),
	ubp AS (
		INSERT INTO
			ib0.bposts (
				b_id,
				b_t_id,
				b_p_id,
				p_name,
				g_p_id,
				msgid,
				date_sent,
				date_recv,
				sage,
				mod_id,
				attrib
			)
		SELECT
			$12,           -- b_id
			ub.last_id,    -- b_t_id
			ub.last_id,    -- b_p_id
			$13,           -- p_name
			ugp.g_p_id,    -- g_p_id
			$3,            -- msgid
			ugp.date_sent, -- date_sent
			ugp.date_recv, -- date_recv
			FALSE,         -- sage
			$15,           -- mod_id
			$16            -- attrib
		FROM
			ub
		CROSS JOIN
			ugp
		RETURNING
			g_p_id,b_p_id
	)`
	// footer
	stf := `
SELECT
	g_p_id,b_p_id
FROM
	ubp`

	if n == 0 {
		st = sth + stf
	} else {
		// dynamically make statement with required places for files
		var b strings.Builder

		st1 := sth + `,
	uf AS (
		INSERT INTO
			ib0.files (
				g_p_id,
				ftype,
				fsize,
				fname,
				thumb,
				oname,
				filecfg,
				thumbcfg,
				extras
			)
		SELECT *
		FROM (
			SELECT g_p_id
			FROM ugp
		) AS q0
		CROSS JOIN (
			VALUES `
		b.WriteString(st1)

		x := postTQMsgArgCount + 1 // counting from 1
		for i := 0; i < n; i++ {
			if i != 0 {
				b.WriteString(", ")
			}
			xq := "($%d::ftype_t,$%d::BIGINT,$%d,$%d,$%d,$%d::jsonb,$%d::jsonb,$%d::jsonb)"
			fmt.Fprintf(&b, xq,
				x+0, x+1, x+2, x+3, x+4, x+5, x+6, x+7)
			x += postTQFileArgCount
		}

		st2 := `
		) AS q1
	)` + stf
		b.WriteString(st2)

		st = b.String()
	}

	//sp.log.LogPrintf(DEBUG, "will prepare newthread(%d) statement:\n%s\n", n, st)
	sp.log.LogPrintf(DEBUG, "will prepare newthread(%d) statement", n)
	s, err = sp.db.DB.Prepare(st)
	if err != nil {
		return nil, sp.sqlError("newthread statement preparation", err)
	}
	sp.log.LogPrintf(DEBUG, "newthread(%d) statement prepared successfully", n)

	sp.ntStmts[n] = s
	return
}

func (sp *PSQLIB) insertNewThread(
	tx *sql.Tx, gstmt *sql.Stmt,
	bid boardID, pInfo mailib.PostInfo, skipover bool, modid uint64) (
	gpid postID, bpid postID, duplicate bool, err error) {

	if len(pInfo.H) == 0 {
		panic("post should have header filled")
	}

	stmt := tx.Stmt(gstmt)

	Hjson, err := json.Marshal(pInfo.H)
	if err != nil {
		panic(err)
	}
	GAjson, err := json.Marshal(pInfo.GA)
	if err != nil {
		panic(err)
	}
	Ljson, err := json.Marshal(&pInfo.L)
	if err != nil {
		panic(err)
	}
	Ejson, err := json.Marshal(&pInfo.E)
	if err != nil {
		panic(err)
	}
	BAjson, err := json.Marshal(pInfo.BA)
	if err != nil {
		panic(err)
	}

	smodid := sql.NullInt64{Int64: int64(modid), Valid: modid != 0}

	sp.log.LogPrintf(DEBUG, "NEWTHREAD %s start", pInfo.ID)

	var r *sql.Row
	if len(pInfo.FI) == 0 {
		r = stmt.QueryRow(
			pInfo.Date,
			pInfo.FC,
			pInfo.MessageID,
			pInfo.MI.Title,
			pInfo.MI.Author,
			pInfo.MI.Trip,
			pInfo.MI.Message,
			Hjson,
			GAjson,
			Ljson,
			Ejson,

			bid,
			pInfo.ID,
			skipover,
			smodid,
			BAjson)
	} else {

		x := postTQMsgArgCount
		xf := postTQFileArgCount
		args := make([]interface{}, x+(len(pInfo.FI)*xf))

		args[0] = pInfo.Date
		args[1] = pInfo.FC
		args[2] = pInfo.MessageID
		args[3] = pInfo.MI.Title
		args[4] = pInfo.MI.Author
		args[5] = pInfo.MI.Trip
		args[6] = pInfo.MI.Message
		args[7] = Hjson
		args[8] = GAjson
		args[9] = Ljson
		args[10] = Ejson

		args[11] = bid
		args[12] = pInfo.ID
		args[13] = skipover
		args[14] = smodid
		args[15] = BAjson

		for i := range pInfo.FI {

			FFjson, err := json.Marshal(pInfo.FI[i].FileAttrib)
			if err != nil {
				panic(err)
			}
			FTjson, err := json.Marshal(pInfo.FI[i].ThumbAttrib)
			if err != nil {
				panic(err)
			}
			FEjson, err := json.Marshal(pInfo.FI[i].Extras)
			if err != nil {
				panic(err)
			}

			args[x+0] = pInfo.FI[i].Type.String()
			args[x+1] = pInfo.FI[i].Size
			args[x+2] = pInfo.FI[i].ID
			args[x+3] = pInfo.FI[i].Thumb
			args[x+4] = pInfo.FI[i].Original
			args[x+5] = FFjson
			args[x+6] = FTjson
			args[x+7] = FEjson

			x += xf
		}
		r = stmt.QueryRow(args...)
	}

	sp.log.LogPrintf(DEBUG, "NEWTHREAD %s process", pInfo.ID)

	err = r.Scan(&gpid, &bpid)
	if err != nil {
		if pqerr, ok := err.(*pq.Error); ok && pqerr.Code == "23505" {
			// duplicate
			return 0, 0, true, nil
		}
		err = sp.sqlError("newthread insert query scan", err)
		return
	}

	sp.log.LogPrintf(DEBUG, "NEWTHREAD %s done", pInfo.ID)

	// done
	return
}
