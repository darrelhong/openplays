-- +goose Up

CREATE UNIQUE INDEX idx_user_notifications_chat_tag
    ON user_notifications(user_id, tag)
    WHERE tag IS NOT NULL AND kind = 'chat.message';

-- +goose Down

DROP INDEX IF EXISTS idx_user_notifications_chat_tag;
