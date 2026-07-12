-- +goose Up

-- Keep the identity-provider photo separate from an OpenPlays-managed avatar.
-- avatar_key is provider-neutral: it is an object key, never a bucket name or URL.
ALTER TABLE users ADD COLUMN oauth_photo_url TEXT;
ALTER TABLE users ADD COLUMN avatar_key TEXT;

UPDATE users SET oauth_photo_url = photo_url;

-- +goose Down

ALTER TABLE users DROP COLUMN avatar_key;
ALTER TABLE users DROP COLUMN oauth_photo_url;
