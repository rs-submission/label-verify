ALTER TABLE applications
    ADD COLUMN IF NOT EXISTS name_address TEXT NOT NULL DEFAULT '';

ALTER TABLE applications
    ADD COLUMN IF NOT EXISTS name_address_norm TEXT NOT NULL DEFAULT '';
