-- Reflection fields on Moment; captions/markers on Cut (Life On Film slice).

ALTER TABLE moments
    ADD COLUMN IF NOT EXISTS reflection_asset_id UUID,
    ADD COLUMN IF NOT EXISTS reflection_question TEXT;

ALTER TABLE cuts
    ADD COLUMN IF NOT EXISTS captions JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS markers JSONB NOT NULL DEFAULT '[]'::jsonb;
