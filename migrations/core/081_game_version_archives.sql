ALTER TABLE core.game_versions ADD COLUMN IF NOT EXISTS archive_path TEXT;
ALTER TABLE core.game_versions ADD COLUMN IF NOT EXISTS archive_name TEXT;
ALTER TABLE core.game_versions ADD COLUMN IF NOT EXISTS archive_size BIGINT;
ALTER TABLE core.game_versions ADD COLUMN IF NOT EXISTS archive_token TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_game_versions_archive_token
    ON core.game_versions (archive_token) WHERE archive_token IS NOT NULL;

ALTER TABLE core.game_versions DROP CONSTRAINT IF EXISTS game_versions_source_type_check;
ALTER TABLE core.game_versions
    ADD CONSTRAINT game_versions_source_type_check
    CHECK (source_type IN ('archive', 'steam', 'docker', 'buildtools'));
