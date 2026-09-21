CREATE TABLE IF NOT EXISTS pipeline_stages(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id UUID NOT NULL,
    name  VARCHAR(150) NOT NULL,
    position INTEGER NOT NULL,
    stage_type  VARCHAR(20) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (pipeline_id) REFERENCES pipelines(id)
);