

ALTER TABLE core.game_versions
    ADD COLUMN IF NOT EXISTS steam_mod_config TEXT;

UPDATE core.game_versions v
SET steam_mod_config = 'cstrike'
FROM core.games g
WHERE v.game_id = g.id
  AND g.slug = 'cs16'
  AND v.steam_app_id = 90
  AND v.steam_mod_config IS NULL;
