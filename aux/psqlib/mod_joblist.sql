-- :name mod_joblist_modlist_changes_get
WITH
	x AS (
		SELECT
			j_id,

			mod_id,

			t_date_sent,
			t_g_p_id,
			t_b_id
		FROM
			ib0.modlist_changes
		ORDER BY
			j_id ASC
		LIMIT
			1
		FOR UPDATE
	),
	d AS (
		DELETE FROM
			ib0.modlist_changes AS mc
		USING
			x
		WHERE
			mc.mod_id = x.mod_id AND
				mc.j_id <> x.j_id
		RETURNING
			x.j_id
	),
	ds AS (
		SELECT
			j_id
		FROM
			d
		LIMIT
			1
	),
	u AS (
		UPDATE
			ib0.modlist_changes AS mc
		SET
			t_date_sent = NULL,
			t_g_p_id    = NULL,
			t_b_id      = NULL
		FROM
			ds
		WHERE
			mc.j_id = ds.j_id
	)
SELECT
	x.j_id,

	x.mod_id,

	(CASE ds.j_id WHEN x.j_id THEN x.t_date_sent ELSE NULL END),
	(CASE ds.j_id WHEN x.j_id THEN x.t_g_p_id    ELSE NULL END),
	(CASE ds.j_id WHEN x.j_id THEN x.t_b_id      ELSE NULL END)
FROM
	x
FULL JOIN
	ds
ON
	TRUE

-- :name mod_joblist_modlist_changes_set
UPDATE
	ib0.modlist_changes
SET
	t_date_sent = $2,
	t_g_p_id    = $3,
	t_b_id      = $4
WHERE
	j_id = $1

-- :name mod_joblist_modlist_changes_del
DELETE FROM
	ib0.modlist_changes
WHERE
	j_id = $1



-- :name mod_joblist_refs_recalc_get
SELECT
	j_id,

	p_name,
	b_name,
	msgid,

	b_id,
	b_p_id
FROM
	ib0.refs_recalc
ORDER BY
	j_id ASC
LIMIT
	1
FOR UPDATE
-- :name mod_joblist_refs_recalc_set
UPDATE
	ib0.refs_recalc
SET
	b_id   = $2,
	b_p_id = $3
WHERE
	j_id = $1
-- :name mod_joblist_refs_recalc_del
DELETE FROM
	ib0.refs_recalc
WHERE
	j_id = $1