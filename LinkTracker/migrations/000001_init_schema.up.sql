CREATE TABLE chats (
    id BIGINT PRIMARY KEY
);

CREATE TABLE links (
    id BIGSERIAL PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    last_updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE link_chat (
    chat_id BIGINT NOT NULL,
    link_id BIGINT NOT NULL,
    PRIMARY KEY (chat_id, link_id),
    FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE,
    FOREIGN KEY (link_id) REFERENCES links(id) ON DELETE CASCADE
);

CREATE TABLE link_tag (
    chat_id BIGINT NOT NULL,
    link_id BIGINT NOT NULL,
    tag TEXT NOT NULL,
    PRIMARY KEY (chat_id, link_id, tag),
    FOREIGN KEY (chat_id, link_id)
        REFERENCES link_chat(chat_id, link_id)
        ON DELETE CASCADE
);

CREATE INDEX idx_link_chat_link_id ON link_chat(link_id);