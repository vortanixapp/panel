UPDATE core.games SET meta = meta || '{"steam_updatable": true}'::jsonb
WHERE slug IN ('cs2', 'counter-strike-2', 'rust', 'gmod', 'garrys-mod')
  AND NOT (meta ? 'steam_updatable');

UPDATE core.games SET meta = meta || '{"steam_updatable": false}'::jsonb
WHERE slug IN ('minecraft', 'samp', 'san-andreas-mp')
  AND NOT (meta ? 'steam_updatable');

COMMENT ON TABLE core.game_versions IS 'Docker/Steam game runtime definitions (E2 top-5 eggs)';
