UPDATE core.game_versions v
SET active = false
WHERE v.active = true
  AND COALESCE(v.meta->>'managed_by', '') <> 'gamecatalog'
  AND COALESCE(v.archive_url, '') = ''
  AND COALESCE(v.steam_app_id, 0) = 0
  AND EXISTS (
      SELECT 1 FROM core.game_versions good
      WHERE good.game_id = v.game_id
        AND good.tenant_id = v.tenant_id
        AND good.id <> v.id
        AND good.active = true
        AND good.meta->>'managed_by' = 'gamecatalog'
        AND (COALESCE(good.archive_url, '') <> '' OR COALESCE(good.steam_app_id, 0) > 0)
  );
