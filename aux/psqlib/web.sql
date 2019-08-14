-- :name web_listboards
SELECT
	b_name,bdesc,attrib,t_count,p_count
FROM
	ib0.boards
ORDER BY
	b_name COLLATE "und-x-icu"

-- :name web_thread_list_page
-- input: {b_name} {page num}
SELECT
	xb.b_id,
	xb.bdesc,
	xb.attrib,
	xb.threads_per_page,
	xb.t_count,
	xt.t_id,
	xt.t_name,
	xt.p_count,
	xt.f_count AS xt_f_count,
	xbp.b_p_id,
	xbp.p_name,
	xp.msgid,
	xp.pdate,
	xp.sage,
	xp.f_count AS xp_f_count,
	xp.author,
	xp.trip,
	xp.title,
	xp.message,
	xp.attrib,
	xp.headers,
	xf.f_id,
	xf.fname,
	xf.ftype,
	xf.fsize,
	xf.thumb,
	xf.oname,
	xf.filecfg,
	xf.thumbcfg
FROM
	ib0.boards xb
LEFT JOIN
	LATERAL (
		SELECT
			zt.b_id,
			zt.t_id,
			zt.t_name,
			zt.bump,
			zt.p_count,
			zt.f_count
		FROM
			ib0.threads AS zt
		WHERE
			zt.b_id = xb.b_id
		ORDER BY
			zt.bump DESC,
			zt.t_id ASC
		LIMIT
			(CASE
				WHEN
					xb.threads_per_page > 0
				THEN
					xb.threads_per_page
				WHEN
					$2 = 0
				THEN
					NULL
				ELSE
					0
				END
			)
		OFFSET
			(CASE
				WHEN
					xb.threads_per_page > 0
				THEN
					$2 * xb.threads_per_page
				ELSE
					0
				END
			)
	) AS xt
ON
	TRUE
LEFT JOIN
	LATERAL (
		SELECT
			b_id,
			t_id,
			b_p_id,
			g_p_id,
			p_name,
			pdate
		FROM
			ib0.bposts AS op
		WHERE
			op.b_id = xb.b_id AND op.b_p_id = xt.t_id
		UNION ALL
		SELECT
			b_id,
			t_id,
			b_p_id,
			g_p_id,
			p_name,
			pdate
		FROM (
			SELECT
				*
			FROM
				(
					SELECT
						rp.b_id,
						rp.t_id,
						rp.b_p_id,
						rp.g_p_id,
						rp.p_name,
						rp.pdate
					FROM
						ib0.bposts AS rp
					WHERE
						rp.b_id = xb.b_id AND
							rp.t_id = xt.t_id AND
							rp.b_p_id != xt.t_id
					ORDER BY
						rp.pdate DESC,
						rp.b_p_id DESC
					LIMIT 5
				) AS tt
			ORDER BY
				pdate ASC,
				b_p_id ASC
		) AS ttt
	) AS xbp
ON
	TRUE
LEFT JOIN
	ib0.posts AS xp
ON
	xbp.g_p_id = xp.g_p_id
LEFT JOIN LATERAL
	(
		SELECT
			*
		FROM
			ib0.files zf
		WHERE
			xp.g_p_id = zf.g_p_id
		ORDER BY
			zf.f_id
	) AS xf
ON
	TRUE
WHERE
	xb.b_name = $1

-- :name web_overboard_page
-- input: {page num} {threads_per_page}
SELECT
	xt.b_id,
	xt.b_name,
	xt.t_id,
	xt.t_name,
	xt.p_count,
	xt.f_count AS xt_f_count,
	xbp.b_p_id,
	xbp.p_name,
	xp.msgid,
	xp.pdate,
	xp.sage,
	xp.f_count AS xp_f_count,
	xp.author,
	xp.trip,
	xp.title,
	xp.message,
	xp.attrib,
	xp.headers,
	xf.f_id,
	xf.fname,
	xf.ftype,
	xf.fsize,
	xf.thumb,
	xf.oname,
	xf.filecfg,
	xf.thumbcfg
FROM
	(
		SELECT
			zt.b_id,
			zb.b_name,
			zt.t_id,
			zt.t_name,
			zt.bump,
			zt.p_count,
			zt.f_count
		FROM
			ib0.threads AS zt
		JOIN
			ib0.boards AS zb
		ON
			zt.b_id = zb.b_id
		WHERE
			zt.skip_over IS NOT TRUE
		ORDER BY
			zt.bump DESC,
			zt.g_t_id ASC,
			zt.b_id ASC
		LIMIT
			(CASE
				WHEN
					$2 > 0
				THEN
					$2
				WHEN
					$1 = 0
				THEN
					NULL
				ELSE
					0
				END
			)
		OFFSET
			(CASE
				WHEN
					$2 > 0
				THEN
					$1 * $2
				ELSE
					0
				END
			)
	) AS xt
LEFT JOIN
	LATERAL (
		SELECT
			b_id,
			t_id,
			b_p_id,
			g_p_id,
			p_name,
			pdate
		FROM
			ib0.bposts AS op
		WHERE
			op.b_id = xt.b_id AND op.b_p_id = xt.t_id
		UNION ALL
		SELECT
			b_id,
			t_id,
			b_p_id,
			g_p_id,
			p_name,
			pdate
		FROM (
			SELECT
				*
			FROM
				(
					SELECT
						rp.b_id,
						rp.t_id,
						rp.b_p_id,
						rp.g_p_id,
						rp.p_name,
						rp.pdate
					FROM
						ib0.bposts AS rp
					WHERE
						rp.b_id = xt.b_id AND
							rp.t_id = xt.t_id AND
							rp.b_p_id != xt.t_id
					ORDER BY
						rp.pdate DESC,
						rp.b_p_id DESC
					LIMIT 5
				) AS tt
			ORDER BY
				pdate ASC,
				b_p_id ASC
		) AS ttt
	) AS xbp
ON
	TRUE
-- XXX possibly misorder join, too annoy to move inside
LEFT JOIN
	ib0.posts AS xp
ON
	xbp.g_p_id = xp.g_p_id
LEFT JOIN LATERAL
	(
		SELECT
			*
		FROM
			ib0.files zf
		WHERE
			xp.g_p_id = zf.g_p_id
		ORDER BY
			zf.f_id ASC
	) AS xf
ON
	TRUE

-- :name web_thread_catalog
-- input: {b_name}
SELECT
	xb.b_id,
	xb.bdesc,
	xb.attrib,
	xt.t_id,
	xt.t_name,
	xt.p_count,
	xt.f_count AS xt_f_count,
	xt.bump,
	xbp.b_p_id,
	xp.pdate,
	xp.f_count AS xp_f_count,
	xp.author,
	xp.trip,
	xp.title,
	xp.message,
	xf.f_id,
	xf.fname,
	xf.ftype,
	xf.thumb,
	xf.thumbcfg
FROM
	ib0.boards xb
LEFT JOIN LATERAL
	(
		SELECT
			*
		FROM
			ib0.threads zt
		WHERE
			xb.b_id = zt.b_id
		ORDER BY
			zt.bump DESC,
			zt.t_id ASC
	) AS xt
ON
	TRUE
LEFT JOIN
	ib0.bposts xbp
ON
	xt.b_id = xbp.b_id AND xt.t_id = xbp.b_p_id
LEFT JOIN
	ib0.posts xp
ON
	xbp.g_p_id = xp.g_p_id
LEFT JOIN
	LATERAL (
		SELECT
			zf.f_id,
			zf.fname,
			zf.ftype,
			zf.thumb,
			zf.thumbcfg
		FROM
			ib0.files AS zf
		WHERE
			xp.g_p_id = zf.g_p_id AND zf.ftype != 'msg'
		ORDER BY
			zf.f_id
		LIMIT
			1
	) AS xf
ON
	TRUE
WHERE
	xb.b_name = $1

-- :name web_overboard_catalog
-- input: {thread_count}
SELECT
	xt.b_id,
	xt.b_name,
	xt.t_id,
	xt.t_name,
	xt.p_count,
	xt.f_count AS xt_f_count,
	xt.bump,
	xbp.b_p_id,
	xp.pdate,
	xp.f_count AS xp_f_count,
	xp.author,
	xp.trip,
	xp.title,
	xp.message,
	xf.f_id,
	xf.fname,
	xf.ftype,
	xf.thumb,
	xf.thumbcfg
FROM
	(
		SELECT
			zt.b_id,
			zb.b_name,
			zt.t_id,
			zt.t_name,
			zt.bump,
			zt.p_count,
			zt.f_count
		FROM
			ib0.threads AS zt
		JOIN
			ib0.boards AS zb
		ON
			zt.b_id = zb.b_id
		WHERE
			zt.skip_over IS NOT TRUE
		ORDER BY
			zt.bump DESC,
			zt.g_t_id ASC,
			zt.b_id ASC
		LIMIT
			$1
	) AS xt
LEFT JOIN
	ib0.bposts xbp
ON
	xt.b_id = xbp.b_id AND xt.t_id = xbp.b_p_id
LEFT JOIN
	ib0.posts xp
ON
	xbp.g_p_id = xp.g_p_id
LEFT JOIN
	LATERAL (
		SELECT
			zf.f_id,
			zf.fname,
			zf.ftype,
			zf.thumb,
			zf.thumbcfg
		FROM
			ib0.files AS zf
		WHERE
			xp.g_p_id = zf.g_p_id AND zf.ftype != 'msg'
		ORDER BY
			zf.f_id
		LIMIT
			1
	) AS xf
ON
	TRUE

-- :name web_thread
-- input: {b_name} {t_name}
SELECT
	xb.b_id,
	xb.bdesc,
	xb.attrib,
	xb.threads_per_page,
	xb.t_count,
	xt.t_id,
	xt.t_name,
	xt.p_count,
	xt.f_count AS xt_f_count,
	xto.t_pos,
	xbp.b_p_id,
	xbp.p_name,
	xp.msgid,
	xp.pdate,
	xp.sage,
	xp.f_count AS xp_f_count,
	xp.author,
	xp.trip,
	xp.title,
	xp.message,
	xp.attrib,
	xp.headers,
	xf.f_id,
	xf.fname,
	xf.ftype,
	xf.fsize,
	xf.thumb,
	xf.oname,
	xf.filecfg,
	xf.thumbcfg
FROM
	ib0.boards AS xb
LEFT JOIN LATERAL
	(
		SELECT
			*
		FROM
			ib0.threads zt
		WHERE
			zt.b_id = xb.b_id AND zt.t_name = $2
		LIMIT
			1
	) AS xt
ON
	TRUE
LEFT JOIN
	LATERAL (
		SELECT
			*
		FROM
			(
				SELECT
					b_id,
					t_id,
					row_number() OVER (
						ORDER BY
							bump DESC,
							t_id ASC
					) AS t_pos
				FROM
					ib0.threads qt
				WHERE
					qt.b_id = xt.b_id
			) AS zt
		WHERE
			xt.b_id = zt.b_id AND xt.t_id = zt.t_id
		LIMIT
			1
	) AS xto
ON
	TRUE
LEFT JOIN LATERAL
	(
		SELECT
			*
		FROM
			ib0.bposts zbp
		WHERE
			xt.b_id = zbp.b_id AND xt.t_id = zbp.t_id
		ORDER BY
			zbp.pdate ASC,zbp.b_p_id ASC
	) AS xbp
ON
	TRUE
LEFT JOIN
	ib0.posts AS xp
ON
	xbp.g_p_id = xp.g_p_id
LEFT JOIN LATERAL
	(
		SELECT
			*
		FROM
			ib0.files zf
		WHERE
			xp.g_p_id = zf.g_p_id
		ORDER BY
			zf.f_id ASC
	) AS xf
ON
	TRUE
WHERE
	xb.b_name=$1



-- TODO common bucket

-- :name web_failref_write
WITH
	delold AS (
		DELETE FROM
			ib0.failrefs
		WHERE
			g_p_id = $1
	)
INSERT INTO
	ib0.failrefs (
		g_p_id,
		p_name,
		b_name,
		msgid
	)
SELECT
	$1,
	unnest($2::text[]) AS p_name,
	unnest($3::text[]) AS b_name,
	unnest($4::text[]) AS msgid

-- :name web_failref_find
-- args: offset,p_name,board,msgid
WITH
	msgs AS (
		SELECT
			*
		FROM
			(
				SELECT
					g_p_id
				FROM
					ib0.failrefs
				WHERE
					-- index-search by first 8 bytes, then narrow
					(p_name LIKE substring($2 for 8) || '%') AND
						($2 LIKE p_name || '%') AND
						(b_name IS NULL OR b_name = $3)
				UNION
				SELECT
					g_p_id
				FROM
					ib0.failrefs
				WHERE
					msgid = $4
				ORDER BY
					g_p_id
			) AS x
		WHERE
			g_p_id > $1
		LIMIT
			8192
	)
SELECT
	msgs.g_p_id,
	xp.message,
	xp.headers -> 'In-Reply-To' ->> 0,
	xp.attrib,
	xbp.b_id,
	xbp.t_id
FROM
	msgs
JOIN
	ib0.posts AS xp
ON
	xp.g_p_id = msgs.g_p_id
JOIN
	LATERAL (
		SELECT
			zbp.b_id,
			zbp.t_id
		FROM
			ib0.bposts AS zbp
		JOIN
			ib0.boards AS zb
		ON
			zbp.b_id = zb.b_id
		WHERE
			zbp.g_p_id = xp.g_p_id
		ORDER BY
			zb.b_name
		LIMIT
			1
	) AS xbp
ON
	TRUE

-- :name update_post_attrs
UPDATE
	ib0.posts
SET
	attrib = $2
WHERE
	g_p_id = $1


-- :name autoregister_mod
INSERT INTO
	ib0.modlist AS ml (
		mod_pubkey,
		automanage
	)
VALUES (
	$1,
	TRUE
)
ON CONFLICT (mod_pubkey) DO UPDATE -- DO NOTHING returns nothing so we update something irrelevant as hack
	SET automanage = ml.automanage
RETURNING
	mod_id, mod_priv


-- :name delete_by_msgid
/*
IMPORTANT:
https://www.postgresql.org/docs/9.6/queries-with.html
All the statements are executed with the same snapshot (see Chapter 13),
so they cannot "see" one another's effects on the target tables.
This alleviates the effects of the unpredictability of the actual order
of row updates, and means that RETURNING data is the only way to
communicate changes between different WITH sub-statements and the main query.
*/
WITH
	delgp AS (
		-- delete global post
		DELETE FROM
			ib0.posts
		WHERE
			msgid = $1 AND padded IS NOT NULL
		RETURNING
			g_p_id,f_count,msgid
	),
	delbp AS (
		-- delete all board posts of that
		DELETE FROM
			ib0.bposts xbp
		USING
			delgp
		WHERE
			xbp.g_p_id = delgp.g_p_id
		RETURNING
			xbp.b_id,xbp.t_id,xbp.b_p_id,xbp.mod_id,delgp.f_count
	),
	delbt AS (
		-- delete thread(s) incase we nuked OP(s)
		DELETE FROM
			ib0.threads xt
		USING
			delbp
		WHERE
			xt.b_id = delbp.b_id AND xt.t_id = delbp.b_p_id
		RETURNING
			xt.b_id,xt.t_id
	),
	updbt AS (
		-- update thread(s) counters incase we haven't deleted thread(s) earlier
		-- un-bump is done adhoc
		UPDATE
			ib0.threads xt
		SET
			p_count = xt.p_count - 1,
			f_count = xt.f_count - delbp.f_count
		FROM
			delbp
		WHERE
			delbp.b_id = xt.b_id AND delbp.t_id = xt.t_id
	),
	delbcp AS (
		-- delete board child posts incase we nuked thread(s)
		DELETE FROM
			ib0.bposts xbp
		USING
			delbt
		WHERE
			xbp.b_id = delbt.b_id AND xbp.t_id = delbt.t_id
		RETURNING
			xbp.b_id,xbp.b_p_id,xbp.g_p_id,xbp.mod_id
	),
	delgcp AS (
		-- delete global child posts (from above)
		DELETE FROM
			ib0.posts xp
		USING
			(
				-- XXX is it even possible to have this false?
				SELECT
					delbcp.g_p_id,COUNT(xbp.g_p_id) > 1 AS hasrefs
				FROM
					delbcp
				LEFT JOIN
					ib0.bposts xbp
				ON
					delbcp.g_p_id = xbp.g_p_id
				GROUP BY
					delbcp.g_p_id
			) AS rcnts
		WHERE
			rcnts.hasrefs = FALSE AND rcnts.g_p_id = xp.g_p_id
		RETURNING
			xp.g_p_id,xp.msgid
	),
	clean_mods AS (
		-- garbage collect moderator list (maybe we nuked mod post(s))
		DELETE FROM
			ib0.modlist mods
		USING
			(
				SELECT
					delmod.mod_id,COUNT(xbp.mod_id) > 1 AS hasrefs
				FROM
					(
						SELECT mod_id,b_id,b_p_id FROM delbp
						UNION ALL
						SELECT mod_id,b_id,b_p_id FROM delbcp
					) AS delmod
				LEFT JOIN
					ib0.bposts xbp
				ON
					delmod.mod_id = xbp.mod_id
				WHERE
					delmod.mod_id IS NOT NULL
				GROUP BY
					delmod.mod_id
			) AS rcnts
		WHERE
			rcnts.hasrefs = FALSE AND rcnts.mod_id = mods.mod_id AND mods.automanage = TRUE
	),
	updb AS (
		-- update boards post and thread counts
		UPDATE
			ib0.boards xb
		SET
			p_count = xb.p_count - xtp.p_count,
			t_count = xb.t_count - xtp.t_count
		FROM
			(
				SELECT
					xx.b_id,
					SUM(xx.p_count) AS p_count,
					COUNT(delbt.b_id) AS t_count
				FROM
					(
						SELECT
							delbpx.b_id,
							COUNT(delbpx.b_id) AS p_count
						FROM
							(
								SELECT b_id FROM delbp
								UNION ALL
								SELECT b_id FROM delbcp
							) AS delbpx
						GROUP BY
							delbpx.b_id
					) AS xx
				LEFT JOIN
					delbt
				ON
					xx.b_id = delbt.b_id
				GROUP BY
					xx.b_id
			) AS xtp
		WHERE
			xb.b_id = xtp.b_id
	),
	delf AS (
		-- delete relevant files
		DELETE FROM
			ib0.files xf
		USING
			(
				SELECT g_p_id FROM delgp
				UNION ALL
				SELECT g_p_id FROM delgcp
			) AS xgpids
		WHERE
			xgpids.g_p_id = xf.g_p_id
		RETURNING
			xf.f_id,xf.fname,xf.thumb
	)
SELECT
	leftf.fname,leftf.fnum,leftt.thumb,leftt.tnum,NULL,NULL,NULL
FROM
	(
		-- minus 1 because snapshot isolation
		SELECT
			delf.fname,COUNT(xf.fname) - 1 AS fnum
		FROM
			delf
		LEFT JOIN
			ib0.files xf
		ON
			delf.fname = xf.fname
		GROUP BY
			delf.fname
	) AS leftf
FULL JOIN
	(
		-- minus 1 because snapshot isolation
		SELECT
			delf.fname,delf.thumb,COUNT(xf.thumb) - 1 AS tnum
		FROM
			delf
		LEFT JOIN
			ib0.files xf
		ON
			delf.fname = xf.fname AND delf.thumb = xf.thumb
		GROUP BY
			delf.fname,delf.thumb
	) AS leftt
ON
	leftf.fname = leftt.fname
UNION ALL
SELECT
	'',0,'',0,b_id,t_id,NULL
FROM
	delbp
WHERE
	t_id != b_p_id
UNION ALL
SELECT
	'',0,'',0,NULL,NULL,msgid
FROM
	delgp
UNION ALL
SELECT
	'',0,'',0,NULL,NULL,msgid
FROM
	delgcp


-- :name ban_by_msgid
-- TODO deduplicate
WITH
	insban AS (
		INSERT INTO
			ib0.banlist (
				msgid,
				b_id,
				b_p_id,
				ban_info
			)
		VALUES
			(
				$1,
				$2,
				$3,
				$4
			)
	),
	delgp AS (
		-- replace global post with ban post
		INSERT INTO
			ib0.posts (
				msgid,
				pdate,
				padded,
				sage,
				f_count,
				author,
				trip,
				title,
				message,
				headers,
				attrib,
				layout,
				extras
			)
		VALUES
			(
				$1,
				NULL,
				NULL,
				FALSE,
				0,
				'',
				'',
				'',
				'',
				NULL,
				NULL,
				NULL,
				NULL
			)
		ON CONFLICT (msgid) DO UPDATE
			SET
				pdate   = excluded.pdate,
				padded  = excluded.padded,
				sage    = excluded.sage,
				f_count = excluded.f_count,
				author  = excluded.author,
				trip    = excluded.trip,
				title   = excluded.title,
				message = excluded.message,
				headers = excluded.headers,
				attrib  = excluded.attrib,
				layout  = excluded.layout,
				extras  = excluded.extras
		RETURNING
			g_p_id,f_count,msgid
	),
	delbp AS (
		-- delete all board posts of that
		DELETE FROM
			ib0.bposts xbp
		USING
			delgp
		WHERE
			xbp.g_p_id = delgp.g_p_id
		RETURNING
			xbp.b_id,xbp.t_id,xbp.b_p_id,xbp.mod_id,delgp.f_count
	),
	delbt AS (
		-- delete thread(s) incase we nuked OP(s)
		DELETE FROM
			ib0.threads xt
		USING
			delbp
		WHERE
			xt.b_id = delbp.b_id AND xt.t_id = delbp.b_p_id
		RETURNING
			xt.b_id,xt.t_id
	),
	updbt AS (
		-- update thread(s) counters incase we haven't deleted thread(s) earlier
		-- un-bump is done adhoc
		UPDATE
			ib0.threads xt
		SET
			p_count = xt.p_count - 1,
			f_count = xt.f_count - delbp.f_count
		FROM
			delbp
		WHERE
			delbp.b_id = xt.b_id AND delbp.t_id = xt.t_id
	),
	delbcp AS (
		-- delete board child posts incase we nuked thread(s)
		DELETE FROM
			ib0.bposts xbp
		USING
			delbt
		WHERE
			xbp.b_id = delbt.b_id AND xbp.t_id = delbt.t_id
		RETURNING
			xbp.b_id,xbp.b_p_id,xbp.g_p_id,xbp.mod_id
	),
	delgcp AS (
		-- delete global child posts (from above)
		DELETE FROM
			ib0.posts xp
		USING
			(
				-- XXX is it even possible to have this false?
				SELECT
					delbcp.g_p_id,COUNT(xbp.g_p_id) > 1 AS hasrefs
				FROM
					delbcp
				LEFT JOIN
					ib0.bposts xbp
				ON
					delbcp.g_p_id = xbp.g_p_id
				GROUP BY
					delbcp.g_p_id
			) AS rcnts
		WHERE
			rcnts.hasrefs = FALSE AND rcnts.g_p_id = xp.g_p_id
		RETURNING
			xp.g_p_id,xp.msgid
	),
	clean_mods AS (
		-- garbage collect moderator list (maybe we nuked mod post(s))
		DELETE FROM
			ib0.modlist mods
		USING
			(
				SELECT
					delmod.mod_id,COUNT(xbp.mod_id) > 1 AS hasrefs
				FROM
					(
						SELECT mod_id,b_id,b_p_id FROM delbp
						UNION ALL
						SELECT mod_id,b_id,b_p_id FROM delbcp
					) AS delmod
				LEFT JOIN
					ib0.bposts xbp
				ON
					delmod.mod_id = xbp.mod_id
				WHERE
					delmod.mod_id IS NOT NULL
				GROUP BY
					delmod.mod_id
			) AS rcnts
		WHERE
			rcnts.hasrefs = FALSE AND rcnts.mod_id = mods.mod_id AND mods.automanage = TRUE
	),
	updb AS (
		-- update boards post and thread counts
		UPDATE
			ib0.boards xb
		SET
			p_count = xb.p_count - xtp.p_count,
			t_count = xb.t_count - xtp.t_count
		FROM
			(
				SELECT
					xx.b_id,
					SUM(xx.p_count) AS p_count,
					COUNT(delbt.b_id) AS t_count
				FROM
					(
						SELECT
							delbpx.b_id,
							COUNT(delbpx.b_id) AS p_count
						FROM
							(
								SELECT b_id FROM delbp
								UNION ALL
								SELECT b_id FROM delbcp
							) AS delbpx
						GROUP BY
							delbpx.b_id
					) AS xx
				LEFT JOIN
					delbt
				ON
					xx.b_id = delbt.b_id
				GROUP BY
					xx.b_id
			) AS xtp
		WHERE
			xb.b_id = xtp.b_id
	),
	delf AS (
		-- delete relevant files
		DELETE FROM
			ib0.files xf
		USING
			(
				SELECT g_p_id FROM delgp
				UNION ALL
				SELECT g_p_id FROM delgcp
			) AS xgpids
		WHERE
			xgpids.g_p_id = xf.g_p_id
		RETURNING
			xf.f_id,xf.fname,xf.thumb
	)
SELECT
	leftf.fname,leftf.fnum,leftt.thumb,leftt.tnum,NULL,NULL,NULL
FROM
	(
		-- minus 1 because snapshot isolation
		SELECT
			delf.fname,COUNT(xf.fname) - 1 AS fnum
		FROM
			delf
		LEFT JOIN
			ib0.files xf
		ON
			delf.fname = xf.fname
		GROUP BY
			delf.fname
	) AS leftf
FULL JOIN
	(
		-- minus 1 because snapshot isolation
		SELECT
			delf.fname,delf.thumb,COUNT(xf.thumb) - 1 AS tnum
		FROM
			delf
		LEFT JOIN
			ib0.files xf
		ON
			delf.fname = xf.fname AND delf.thumb = xf.thumb
		GROUP BY
			delf.fname,delf.thumb
	) AS leftt
ON
	leftf.fname = leftt.fname
UNION ALL
SELECT
	'',0,'',0,b_id,t_id,NULL
FROM
	delbp
WHERE
	t_id != b_p_id
UNION ALL
SELECT
	'',0,'',0,NULL,NULL,msgid
FROM
	delgp
UNION ALL
SELECT
	'',0,'',0,NULL,NULL,msgid
FROM
	delgcp


-- :name bname_topts_by_tid
SELECT
	xb.b_name,xb.thread_opts,xt.thread_opts
FROM
	ib0.boards xb
JOIN
	ib0.threads xt
ON
	xb.b_id = xt.b_id
WHERE
	xb.b_id = $1 AND xt.t_id = $2

-- :name refresh_bump_by_tid
UPDATE
	ib0.threads
SET
	bump = pdate
FROM
	(
		SELECT
			pdate
		FROM
			(
				SELECT
					pdate,
					b_p_id,
					sage
				FROM
					ib0.bposts
				WHERE
					-- count sages against bump limit.
					-- because others do it like that :<
					b_id = $1 AND t_id = $2
				ORDER BY
					pdate ASC,
					b_p_id ASC
				LIMIT
					$3
				-- take bump posts, sorted by original date,
				-- only upto bump limit
			) AS tt
		WHERE
			sage != TRUE
		ORDER BY
			pdate DESC,b_p_id DESC
		LIMIT
			1
		-- and pick latest one
	) as xbump
WHERE
	b_id = $1 AND t_id = $2


-- :name set_mod_priv
-- args: <pubkey> <newpriv>
INSERT INTO
	ib0.modlist AS ml (
		mod_pubkey,
		automanage,
		mod_priv
	)
VALUES
	(
		$1,
		FALSE,
		$2
	)
ON CONFLICT (mod_pubkey) DO UPDATE
	SET
		automanage = FALSE,
		mod_priv = $2
	WHERE
		ml.mod_priv <> $2 OR ml.automanage <> FALSE
RETURNING -- inserted or modified
	mod_id

-- :name unset_mod
-- args: <pubkey>
WITH
	-- do update there
	upd_mod AS (
		UPDATE
			ib0.modlist
		SET
			mod_priv = 'none', -- don't see point having anything else there yet
			automanage = TRUE
		WHERE
			mod_pubkey = $1 AND
			(mod_priv <> 'none' OR automanage <> TRUE)
		RETURNING
			mod_id
	)
-- garbage collect moderator list
DELETE FROM
	ib0.modlist mods
USING
	(
		SELECT
			delmod.mod_id,COUNT(xbp.mod_id) > 0 AS hasrefs
		FROM
			upd_mod AS delmod
		LEFT JOIN
			ib0.bposts xbp
		ON
			delmod.mod_id = xbp.mod_id
		GROUP BY
			delmod.mod_id
	) AS rcnts
WHERE
	rcnts.hasrefs = FALSE AND rcnts.mod_id = mods.mod_id

-- :name fetch_and_clear_mod_msgs
-- args: <modid> <offset>
-- fetches all messages of mod, and also clears all their actions
WITH
	zbp AS (
		SELECT
			b_id,
			b_p_id,
			t_id,
			g_p_id
		FROM
			ib0.bposts
		WHERE
			mod_id = $1
		ORDER BY
			b_id,b_p_id
		OFFSET
			$2
		LIMIT
			4096
	),
	zd AS (
		DELETE FROM
			ib0.banlist bl
		USING
			zbp
		WHERE
			bl.b_id = zbp.b_id AND bl.b_p_id = zbp.b_p_id
	)
SELECT
	zbp.g_p_id,
	zbp.b_id,
	zbp.b_p_id,
	yb.b_name,
	yp.msgid,
	ypp.msgid,
	yp.title,
	yp.pdate,
	yp.message,
	yp.extras -> 'text_attach',
	yf.fname
FROM
	zbp
-- board
JOIN
	ib0.boards yb
ON
	zbp.b_id = yb.b_id
-- global post
JOIN
	ib0.posts yp
ON
	zbp.g_p_id = yp.g_p_id
-- files of global post
LEFT JOIN LATERAL
	(
		SELECT
			xf.fname
		FROM
			ib0.files xf
		WHERE
			yp.g_p_id = xf.g_p_id
		ORDER BY
			xf.f_id -- important
	) AS yf
ON
	TRUE
-- parent board post
LEFT JOIN
	ib0.bposts ypbp
ON
	zbp.b_id = ypbp.b_id AND zbp.t_id = ypbp.b_p_id AND zbp.t_id != zbp.b_p_id
-- parent global post
LEFT JOIN
	ib0.posts ypp
ON
	ypbp.g_p_id = ypp.g_p_id
