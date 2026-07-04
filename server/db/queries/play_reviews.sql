-- name: UpsertPlayReview :one
INSERT INTO play_reviews (
    play_id, reviewer_user_id, reviewee_user_id, rating, props, shoutout
) VALUES (
    ?, ?, ?, ?, ?, ?
)
ON CONFLICT (play_id, reviewer_user_id, reviewee_user_id) DO UPDATE SET
    rating = excluded.rating,
    props = excluded.props,
    shoutout = excluded.shoutout,
    updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
RETURNING *;

-- name: ListMyPlayReviews :many
SELECT * FROM play_reviews
WHERE play_id = ? AND reviewer_user_id = ?
ORDER BY reviewee_user_id ASC;

-- name: ListReviewEligibleUsersByPlay :many
-- The single source of truth for who can review and be reviewed on a play:
-- active registered users holding a reserved spot (confirmed or added) plus
-- the play hosts. Guests have no account and are excluded.
SELECT
    u.id,
    u.display_name,
    u.username,
    u.photo_url,
    CAST(EXISTS (
        SELECT 1 FROM play_hosts ph2
        WHERE ph2.play_id = sqlc.arg('play_id') AND ph2.user_id = u.id
    ) AS INTEGER) AS is_host
FROM users u
WHERE u.status = 'active'
  AND u.id IN (
      SELECT pp.user_id FROM play_participants pp
      WHERE pp.play_id = sqlc.arg('play_id')
        AND pp.user_id IS NOT NULL
        AND pp.status IN ('confirmed', 'added')
      UNION
      SELECT ph.user_id FROM play_hosts ph
      WHERE ph.play_id = sqlc.arg('play_id')
  )
ORDER BY u.display_name ASC, u.id ASC;

-- name: GetUserRatingAggregate :one
-- Ratings are anonymous: only the aggregate ever leaves the database.
SELECT
    CAST(COALESCE(AVG(rating), 0) AS REAL) AS average,
    COUNT(rating) AS rating_count
FROM play_reviews
WHERE reviewee_user_id = ? AND rating IS NOT NULL;

-- name: ListUserPropCounts :many
-- Props are sport-linked: each given prop counts toward the sport of the
-- play it was earned in.
SELECT
    p.sport,
    CAST(je.value AS TEXT) AS prop,
    COUNT(*) AS prop_count
FROM play_reviews r
JOIN plays p ON p.id = r.play_id
CROSS JOIN json_each(r.props) je
WHERE r.reviewee_user_id = ?
GROUP BY p.sport, je.value
ORDER BY p.sport ASC, prop_count DESC, prop ASC;

-- name: ListUserShoutouts :many
-- Shoutouts are attributed by design; the review's rating is never selected.
SELECT
    r.shoutout,
    r.created_at,
    u.display_name AS reviewer_display_name,
    u.username AS reviewer_username,
    u.photo_url AS reviewer_photo_url,
    p.id AS play_id,
    p.sport,
    p.name AS play_name,
    p.starts_at
FROM play_reviews r
JOIN users u ON u.id = r.reviewer_user_id AND u.status = 'active'
JOIN plays p ON p.id = r.play_id
WHERE r.reviewee_user_id = ? AND r.shoutout IS NOT NULL
ORDER BY r.created_at DESC, r.id DESC
LIMIT ?;

-- name: ListPlaysNeedingReviewPrompt :many
-- Plays that ended at least an hour ago; the 72h lower bound keeps the scan
-- small and stops plays that ended before this feature shipped from prompting.
SELECT
    p.id,
    p.name,
    COALESCE(v.name, NULLIF(p.venue, ''), 'No venue') AS venue_name,
    p.sport
FROM plays p
LEFT JOIN venues v ON v.id = p.venue_id
WHERE p.cancelled_at IS NULL
  AND p.created_by IS NOT NULL
  AND p.ends_at <= strftime('%Y-%m-%d %H:%M:%S+00:00', 'now', '-1 hour')
  AND p.ends_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now', '-72 hours')
ORDER BY p.ends_at ASC;

-- name: MarkReviewPromptSent :execrows
INSERT OR IGNORE INTO play_review_prompts (play_id, user_id)
VALUES (?, ?);
