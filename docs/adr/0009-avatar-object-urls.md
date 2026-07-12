# 0009: Persist avatar object URLs with stable object keys

Status: Accepted

## Context

`users.photo_url` is the displayed photo and may contain either an OAuth
provider URL or a custom avatar URL. Custom avatars also retain their stable R2
object identity in `users.avatar_key`.

## Decision

When a custom avatar is saved, persist both its provider-independent object key
and the public URL derived from `OBJECT_STORE_PUBLIC_BASE_URL`. This keeps user
reads simple and preserves the existing `photo_url` contract.

## Consequences

Changing `OBJECT_STORE_PUBLIC_BASE_URL` does not rewrite existing rows. Keep the
old domain available during a migration, then backfill custom-avatar URLs:

```sql
UPDATE users
SET photo_url = 'https://new-avatar-domain.example/' || avatar_key
WHERE avatar_key IS NOT NULL;
```

No objects need to be copied or renamed. `avatar_key`, rather than `photo_url`,
is the stable object identity.
