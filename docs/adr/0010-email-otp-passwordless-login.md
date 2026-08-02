# 0010 - Email OTP passwordless login

- Status: Accepted
- Date: 2026-08-01

## Context

We need a low-friction passwordless login without passwords or a callback page. A short code keeps
the user in the login screen and feels familiar to users of GitHub and similar products.

## Decision

Email OTP is the passwordless email-login flow. The API exposes
`POST /api/auth/email-otp/request` and `POST /api/auth/email-otp/verify`. Codes are six digits,
expire after 15 minutes, are single-use, and are stored only as HMAC-SHA-256 verifiers keyed by a
server secret. Five incorrect attempts invalidate a challenge. The provider-neutral email sender,
exact domain allowlist, and
process-local request limiter are reused.

The UI uses direct, conversational copy: “Check your inbox”, “We sent a 6-digit code”, and “Sign in”.

## Consequences

- Login stays on one page and does not depend on URL fragments or callback JavaScript.
- Codes require one copy/paste step and are easy to use across devices.
- OTP challenges add a small database table and an attempt counter.
- The email provider can still be replaced without changing auth behavior.

## Revisit triggers

- If code entry causes material friction, revisit the login UX using funnel and support data.
- Before running multiple API instances, move the request limiter to shared state.
