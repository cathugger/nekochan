package psqlib

import (
	"database/sql"
	"encoding/hex"
	"strings"

	xtypes "github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"

	. "nksrv/lib/logx"
	"nksrv/lib/thumbnailer"
)

func (sp *PSQLIB) pickThumbPlan(isReply, isSage bool) thumbnailer.ThumbPlan {
	if !isReply {
		return sp.tplan_thread
	} else if !isSage {
		return sp.tplan_reply
	} else {
		return sp.tplan_sage
	}
}

func mustUnmarshal(x interface{}, j xtypes.JSONText) {
	err := j.Unmarshal(&x)
	if err != nil {
		panic("json unmarshal")
	}
}

func (sp *PSQLIB) registeredMod(
	tx *sql.Tx, pubkeystr string) (
	modid uint64, hascap bool, mcc ModCombinedCaps,
	err error) {

	// mod posts MAY later come back and want more of things in this table (if they eval/GC modposts)
	// at which point we're fucked because moddel posts also will exclusively block files table
	// and then we won't be able to insert into it..
	_, err = tx.Exec("LOCK ib0.modlist IN EXCLUSIVE MODE")
	if err != nil {
		err = sp.sqlError("lock ib0.modlist query", err)
		return
	}

	sp.log.LogPrintf(DEBUG, "REGMOD %s done locking ib0.modlist", pubkeystr)

	st := tx.Stmt(sp.st_prep[st_mod_autoregister_mod])
	x := 0
	for {

		var (
			m_g_cap   sql.NullString
			m_b_cap   map[string]string
			m_b_cap_j xtypes.JSONText

			m_g_caplvl   *[]sql.NullInt32
			m_b_caplvl   map[string]string
			m_b_caplvl_j xtypes.JSONText

			mi_g_cap   sql.NullString
			mi_b_cap   map[string]string
			mi_b_cap_j xtypes.JSONText

			mi_g_caplvl   *[]sql.NullInt32
			mi_b_caplvl   map[string]string
			mi_b_caplvl_j xtypes.JSONText
		)

		err = st.QueryRow(pubkeystr).Scan(
			&modid,

			&m_g_cap,
			&m_b_cap_j,
			pq.Array(&m_g_caplvl),
			&m_b_caplvl_j,

			&mi_g_cap,
			&mi_b_cap_j,
			pq.Array(&mi_g_caplvl),
			&mi_b_caplvl_j)

		if err != nil {

			if err == sql.ErrNoRows && x < 100 {

				x++

				sp.log.LogPrintf(DEBUG, "REGMOD %s retry", pubkeystr)

				continue
			}

			err = sp.sqlError("st_web_autoregister_mod queryrowscan", err)
			return
		}

		mustUnmarshal(&m_b_cap, m_b_cap_j)
		mustUnmarshal(&m_b_caplvl, m_b_caplvl_j)
		mustUnmarshal(&mi_b_cap, mi_b_cap_j)
		mustUnmarshal(&mi_b_caplvl, mi_b_caplvl_j)

		// enough to check only usable flags
		hascap = m_g_cap.Valid || len(m_b_cap) != 0 ||
			m_g_caplvl != nil || len(m_b_caplvl) != 0

		if m_g_cap.Valid {
			mcc.ModCap.Cap = StrToCap(m_g_cap.String)
		}
		if m_g_caplvl != nil {
			mcc.ModCap = processCapLevel(mcc.ModCap, *m_g_caplvl)
		}

		if mi_g_cap.Valid {
			mcc.ModInheritCap.Cap = StrToCap(mi_g_cap.String)
		}
		if mi_g_caplvl != nil {
			mcc.ModInheritCap = processCapLevel(mcc.ModInheritCap, *mi_g_caplvl)
		}

		mcc.ModBoardCap = make(ModBoardCap)
		mcc.ModBoardCap.TakeIn(m_b_cap, m_b_caplvl)

		mcc.ModInheritBoardCap = make(ModBoardCap)
		mcc.ModInheritBoardCap.TakeIn(mi_b_cap, mi_b_caplvl)

		return
	}
}

func makeCapLvlArray(mc ModCap) interface{} {
	var x [caplvlx_num]sql.NullInt32
	for i := range x {
		x[i].Int32 = int32(mc.CapLevel[i])
		x[i].Valid = mc.CapLevel[i] >= 0
	}
	return pq.Array(x)
}

func (sp *PSQLIB) setModCap(
	tx *sql.Tx, pubkeystr, group string, m_cap, mi_cap ModCap) (err error) {

	// do key update
	var dummy int32
	// this probably should lock relevant row.
	// that should block reads of this row I think?
	// which would mean no further new mod posts for this key
	var r *sql.Row

	m_caplvl := makeCapLvlArray(m_cap)
	mi_caplvl := makeCapLvlArray(mi_cap)

	if group == "" {

		ust := tx.Stmt(sp.st_prep[st_mod_set_mod_priv])

		r = ust.QueryRow(

			pubkeystr,

			m_cap.Cap.String(),
			m_caplvl,

			mi_cap.Cap.String(),
			mi_caplvl)

	} else {

		ust := tx.Stmt(sp.st_prep[st_mod_set_mod_priv_group])

		r = ust.QueryRow(

			pubkeystr,
			group,

			m_cap.Cap.String(),
			m_caplvl,

			mi_cap.Cap.String(),
			mi_caplvl)

	}

	err = r.Scan(&dummy)

	if err != nil {
		if err == sql.ErrNoRows {
			// we changed nothing so return now
			sp.log.LogPrintf(DEBUG, "setmodpriv: %s priv unchanged", pubkeystr)
			err = nil
			return
		}
		err = sp.sqlError("st_web_set_mod_priv queryrowscan", err)
		return
	}

	sp.log.LogPrintf(DEBUG,
		"setmodpriv: %s priv changed", pubkeystr)

	return
}

func (sp *PSQLIB) DemoSetModCap(mods []string, modCap ModCap) {
	var err error

	for i, s := range mods {
		if _, err = hex.DecodeString(s); err != nil {
			sp.log.LogPrintf(ERROR, "invalid modid %q", s)
			return
		}
		// we use uppercase (I forgot why)
		mods[i] = strings.ToUpper(s)
	}

	tx, err := sp.db.DB.Begin()
	if err != nil {
		err = sp.sqlError("tx begin", err)
		sp.log.LogPrintf(ERROR, "%v", err)
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var delmsgids delMsgIDState
	defer func() { sp.cleanDeletedMsgIDs(delmsgids) }()

	for _, s := range mods {
		sp.log.LogPrintf(INFO, "setmodpriv %s %s", s, modCap.String())

		err = sp.setModCap(tx, s, "", modCap, noneModCap)
		if err != nil {
			sp.log.LogPrintf(ERROR, "%v", err)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		err = sp.sqlError("tx commit", err)
		sp.log.LogPrintf(ERROR, "%v", err)
		return
	}
}

func checkFiles() {
	//
	//sp.st_prep[st_mod_load_files].
}
