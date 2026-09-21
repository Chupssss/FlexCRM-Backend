CREATE TABLE IF NOT EXISTS activities(
    company_id      UUID NOT NULL,
    actor_id        UUID  NULL,
    entity_type     VARCHAR(50) NOT NULL,
    entity_id       UUID NOT NULL,
    action          VARCHAR(50) NOT NULL,
    metadata        JSONB NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (company_id) REFERENCES companies(id),
    FOREIGN KEY (actor_id) REFERENCES users(id)
);