UPDATE core.game_versions gv
SET steam_app_id = fix.app_id
FROM core.games g,
     (VALUES ('css', 232330::bigint), ('gmod', 4020), ('squad', 403240)) AS fix(slug, app_id)
WHERE g.id = gv.game_id
  AND g.slug = fix.slug
  AND gv.steam_app_id IS DISTINCT FROM fix.app_id
  AND gv.steam_app_id IN (240, 4000, 393380);

UPDATE core.game_versions
SET sort_order = 100
WHERE version = 'latest'
  AND COALESCE(meta->>'managed_by', '') <> 'gamecatalog'
  AND sort_order = 0
  AND NOT EXISTS (
      SELECT 1 FROM core.servers s WHERE s.game_version_id = core.game_versions.id
  );

UPDATE core.games
SET active = false
WHERE slug IN (
    '7d2d', 'arksa', 'arkse', 'arma3', 'armaref', 'bbermuda', 'conan',
    'daydrag', 'dayz', 'dayzdev', 'draconia', 'empyrion', 'enshroud',
    'hytale', 'icarus', 'isleevr', 'lifeyo', 'mordhau', 'nosurv', 'palworld',
    'pathtit', 'rs2vn', 'rust', 'satisfac', 'scum', 'sotf', 'soulmask',
    'spaceeng', 'squad', 'starrupt', 'theisle', 'ttworlds', 'vrising'
)
  AND active = true
  AND NOT EXISTS (
      SELECT 1 FROM core.servers s
      WHERE s.tenant_id = core.games.tenant_id AND s.game_id = core.games.slug
  );

UPDATE core.games SET query_protocol = 'a2s' WHERE slug = 'rust' AND query_protocol <> 'a2s';
UPDATE core.games SET query_protocol = 'samp' WHERE slug = 'crmp' AND query_protocol <> 'samp';
UPDATE core.games SET query_protocol = 'none' WHERE slug IN ('mta', 'factorio', 'hytale') AND query_protocol <> 'none';
