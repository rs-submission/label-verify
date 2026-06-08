CREATE TABLE IF NOT EXISTS applications (
    application_id TEXT PRIMARY KEY,
    brand TEXT NOT NULL,
    class_type TEXT NOT NULL,
    net_contents TEXT NOT NULL,
    abv TEXT NOT NULL,
    government_warning TEXT NOT NULL,
    name_address TEXT NOT NULL DEFAULT '',
    foreign_blocks JSONB NOT NULL DEFAULT '[]'::jsonb,
    declared_languages TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    brand_norm TEXT NOT NULL,
    class_type_norm TEXT NOT NULL,
    net_contents_norm TEXT NOT NULL,
    abv_norm TEXT NOT NULL,
    government_warning_norm TEXT NOT NULL,
    name_address_norm TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS verification_results (
    id BIGSERIAL PRIMARY KEY,
    application_id TEXT NOT NULL REFERENCES applications(application_id),
    image_ref TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    confidence DOUBLE PRECISION NOT NULL,
    idempotency_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS field_results (
    id BIGSERIAL PRIMARY KEY,
    verification_id BIGINT NOT NULL REFERENCES verification_results(id) ON DELETE CASCADE,
    field_name TEXT NOT NULL,
    expected_value TEXT NOT NULL,
    extracted_value TEXT NOT NULL,
    match_type TEXT NOT NULL,
    score DOUBLE PRECISION NOT NULL,
    pass BOOLEAN NOT NULL,
    diff TEXT NOT NULL
);
