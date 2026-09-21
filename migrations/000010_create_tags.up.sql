CREATE TABLE IF NOT EXISTS tags(
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      UUID  NOT NULL,
    name            VARCHAR(100) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (company_id) REFERENCES companies(id)
);