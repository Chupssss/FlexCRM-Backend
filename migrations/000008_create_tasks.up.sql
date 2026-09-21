CREATE TABLE IF NOT EXISTS tasks(
    company_id      UUID  NOT NULL,
    created_by      UUID  NOT NULL,
    assignee_id     UUID NULL,
    client_id       UUID  NULL,
    deal_id         UUID  NULL,
    title           VARCHAR(255) NOT NULL,
    description     TEXT NULL,
    status          VARCHAR(30) NOT NULL,
    priority        VARCHAR(30) NOT NULL,
    due_at          TIMESTAMPTZ NULL,
    completed_at    TIMESTAMPTZ NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (company_id) REFERENCES companies(id),
    FOREIGN KEY (created_by) REFERENCES users(id),
    FOREIGN KEY (client_id) REFERENCES clients(id),
    FOREIGN KEY (deal_id) REFERENCES deals(id)
);