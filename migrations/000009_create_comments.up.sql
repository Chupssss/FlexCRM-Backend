CREATE TABLE IF NOT EXISTS comments(
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      UUID  NOT NULL,
    author_id       UUID  NOT NULL,
    client_id       UUID  NULL,
    deal_id         UUID NULL,
    task_id         UUID  NULL,
    text            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (company_id) REFERENCES companies(id),
    FOREIGN KEY (author_id) REFERENCES users(id)
);