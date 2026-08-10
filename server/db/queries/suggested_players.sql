-- name: ListSuggestedPlayers :many
-- Active registered users who shared an ended play with the viewer. The
-- roster shape mirrors ListReviewEligibleUsersByPlay: reserved participants
-- and hosts are eligible, while guests are excluded.
WITH eligible_members AS (
    SELECT pp.play_id, pp.user_id
    FROM play_participants pp
    JOIN plays p ON p.id = pp.play_id
    WHERE p.cancelled_at IS NULL
      AND p.ends_at <= strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
      AND pp.user_id IS NOT NULL
      AND pp.status IN ('confirmed', 'added')

    UNION

    SELECT ph.play_id, ph.user_id
    FROM play_hosts ph
    JOIN plays p ON p.id = ph.play_id
    WHERE p.cancelled_at IS NULL
      AND p.ends_at <= strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
),
viewer_plays AS (
    SELECT DISTINCT play_id
    FROM eligible_members
    WHERE user_id = sqlc.arg('user_id')
)
SELECT
    u.id,
    u.display_name,
    u.username,
    u.photo_url,
    u.sports_profile,
    COUNT(DISTINCT em.play_id) AS play_count
FROM eligible_members em
JOIN viewer_plays vp ON vp.play_id = em.play_id
JOIN users u ON u.id = em.user_id
WHERE u.status = 'active'
  AND u.id <> sqlc.arg('user_id')
  AND NOT EXISTS (
      SELECT 1
      FROM user_blocks ub
      WHERE (ub.blocker_id = sqlc.arg('user_id') AND ub.blocked_id = u.id)
         OR (ub.blocker_id = u.id AND ub.blocked_id = sqlc.arg('user_id'))
  )
  AND (
    sqlc.arg('exclude_play_id') = ''
    OR NOT EXISTS (
        SELECT 1
        FROM play_participants pp
        WHERE pp.play_id = sqlc.arg('exclude_play_id')
          AND pp.user_id = u.id
    )
  )
GROUP BY u.id, u.display_name, u.username, u.photo_url, u.sports_profile
ORDER BY play_count DESC, u.display_name ASC, u.id ASC
LIMIT sqlc.arg('limit');
