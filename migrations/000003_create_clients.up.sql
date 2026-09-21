CREATE TABLE IF NOT EXISTS clients(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL,
    responsible_user_id UUID,
    created_by UUID NOT NULL,
    client_type VARCHAR(20) NOT NULL,
    first_name VARCHAR(100) NULL,
    last_name VARCHAR(100) NULL,
    middle_name VARCHAR(100) NULL,
    company_name VARCHAR(255) NULL,
    email VARCHAR(255) NULL,
    phone VARCHAR(50) NULL,
    description TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (company_id) REFERENCES companies(id),
    FOREIGN KEY (responsible_user_id) REFERENCES users(id),
    FOREIGN KEY (created_by) REFERENCES users(id),
    CHECK (client_type IN ('PERSON', 'COMPANY')),

CHECK (
    (
        client_type = 'PERSON'
        AND first_name IS NOT NULL
        AND last_name IS NOT NULL
        AND company_name IS NULL
    )
    OR
    (
        client_type = 'COMPANY'
        AND company_name IS NOT NULL
    )
)
);