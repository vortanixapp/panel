

WITH catalog(slug, name, description, code, query_protocol, min_port, max_port, image_url) AS (
    VALUES
        ('7d2d', '7 Days to Die', '7 Days to Die dedicated server.', '7d2d', 'a2s', 26900, 27000, '/games/7d2d.svg'),
        ('arksa', 'Ark Survival Ascended', 'ARK: Survival Ascended dedicated server.', 'arksa', 'a2s', 7777, 28000, '/games/arksa.svg'),
        ('arkse', 'ARK: Survival Evolved', 'ARK: Survival Evolved dedicated server.', 'arkse', 'a2s', 7777, 28000, '/games/arkse.svg'),
        ('arma3', 'Arma 3', 'Arma 3 dedicated server.', 'arma3', 'a2s', 2302, 2400, '/games/arma3.svg'),
        ('armaref', 'Arma Reforger', 'Arma Reforger dedicated server.', 'armaref', 'a2s', 2001, 2100, '/games/armaref.svg'),
        ('bbermuda', 'Beasts of Bermuda', 'Beasts of Bermuda dedicated server.', 'bbermuda', 'a2s', 7777, 7900, '/games/bbermuda.svg'),
        ('conan', 'Conan Exiles', 'Conan Exiles dedicated server.', 'conan', 'a2s', 7777, 7900, '/games/conan.svg'),
        ('crmp', 'CRMP', 'CRMP server (based on SA-MP).', 'crmp', 'none', 7777, 7877, '/games/crmp.svg'),
        ('cs16', 'Counter-Strike 1.6', 'Classic Counter-Strike 1.6 multiplayer server', 'cstrike', 'a2s', 27015, 27030, '/games/cs16.svg'),
        ('cs2', 'Counter-Strike 2', 'Counter-Strike 2 dedicated server.', 'cs2', 'a2s', 27015, 27030, '/games/cs2.svg'),
        ('css', 'Counter-Strike: Source', 'Counter-Strike: Source dedicated server.', 'css', 'a2s', 27015, 27030, '/games/css.svg'),
        ('daydrag', 'Day of Dragons', 'Day of Dragons dedicated server.', 'daydrag', 'a2s', 7777, 7900, '/games/daydrag.svg'),
        ('dayz', 'DayZ Standalone', 'DayZ Standalone dedicated server.', 'dayz', 'a2s', 2302, 2400, '/games/dayz.svg'),
        ('dayzdev', 'DayZ Standalone DEV', 'DayZ Standalone DEV dedicated server.', 'dayzdev', 'a2s', 2302, 2400, '/games/dayzdev.svg'),
        ('draconia', 'Draconia', 'Draconia dedicated server.', 'draconia', 'a2s', 27015, 27100, '/games/draconia.svg'),
        ('empyrion', 'Empyrion - Galactic Survival', 'Empyrion dedicated server.', 'empyrion', 'a2s', 30000, 30100, '/games/empyrion.svg'),
        ('enshroud', 'Enshrouded', 'Enshrouded dedicated server.', 'enshroud', 'a2s', 15637, 15700, '/games/enshroud.svg'),
        ('factorio', 'Factorio', 'Factorio dedicated server.', 'factorio', 'a2s', 34197, 34220, '/games/factorio.svg'),
        ('gmod', 'Garry''s Mod', 'Garry''s Mod dedicated server (Source)', 'gmod', 'a2s', 27015, 27060, '/games/gmod.svg'),
        ('hytale', 'Hytale', 'Hytale server profile.', 'hytale', 'a2s', 25565, 25650, '/games/hytale.svg'),
        ('icarus', 'Icarus', 'Icarus dedicated server.', 'icarus', 'a2s', 17777, 17850, '/games/icarus.svg'),
        ('isleevr', 'The Isle EVRIMA', 'The Isle EVRIMA dedicated server.', 'isleevr', 'a2s', 7777, 7900, '/games/isleevr.svg'),
        ('lifeyo', 'Life is Feudal: Your Own', 'Life is Feudal: Your Own dedicated server.', 'lifeyo', 'a2s', 28000, 28100, '/games/lifeyo.svg'),
        ('mcbedrock', 'Minecraft Bedrock', 'Minecraft Bedrock Edition server', 'bedrock', 'mc', 19132, 19160, '/games/mcbedrock.svg'),
        ('mcfabric', 'Minecraft Java (Fabric)', 'Minecraft Java Edition (Fabric) server', 'mcfabric', 'mc', 25565, 25600, '/games/mcfabric.svg'),
        ('mcforge', 'Minecraft Java (Forge)', 'Minecraft Java Edition (Forge) server', 'mcforge', 'mc', 25565, 25600, '/games/mcforge.svg'),
        ('mcjava', 'Minecraft Java (Vanilla)', 'Minecraft Java Edition (Vanilla) server', 'mcjava', 'mc', 25565, 25600, '/games/mcjava.svg'),
        ('mcpaper', 'Minecraft Java (Paper)', 'Minecraft Java Edition (Paper) server', 'mcpaper', 'mc', 25565, 25600, '/games/mcpaper.svg'),
        ('mcspigot', 'Minecraft Java (Spigot)', 'Minecraft Java Edition (Spigot) server', 'mcspigot', 'mc', 25565, 25600, '/games/mcspigot.svg'),
        ('mordhau', 'Mordhau', 'Mordhau dedicated server.', 'mordhau', 'a2s', 7777, 7900, '/games/mordhau.svg'),
        ('mta', 'MTA:SA', 'Multi Theft Auto: San Andreas dedicated server.', 'mta', 'none', 22003, 22100, '/games/mta.svg'),
        ('nosurv', 'No One Survived', 'No One Survived dedicated server.', 'nosurv', 'a2s', 7777, 7900, '/games/nosurv.svg'),
        ('palworld', 'Palworld', 'Palworld dedicated server.', 'palworld', 'a2s', 8211, 8250, '/games/palworld.svg'),
        ('pathtit', 'Path of Titans', 'Path of Titans dedicated server.', 'pathtit', 'a2s', 7777, 7900, '/games/pathtit.svg'),
        ('pzomboid', 'Project Zomboid', 'Project Zomboid dedicated server.', 'pzomboid', 'a2s', 16261, 16300, '/games/pzomboid.svg'),
        ('rs2vn', 'Rising Storm 2 Vietnam', 'Rising Storm 2 Vietnam dedicated server.', 'rs2vn', 'a2s', 7777, 7900, '/games/rs2vn.svg'),
        ('rust', 'Rust', 'Rust dedicated server.', 'rust', 'none', 28015, 28100, '/games/rust.svg'),
        ('samp', 'SA-MP', 'San Andreas Multiplayer dedicated server.', 'samp', 'samp', 7777, 7877, '/games/samp.svg'),
        ('satisfac', 'Satisfactory', 'Satisfactory dedicated server.', 'satisfac', 'a2s', 7777, 7900, '/games/satisfac.svg'),
        ('scum', 'SCUM', 'SCUM dedicated server.', 'scum', 'a2s', 7777, 7900, '/games/scum.svg'),
        ('sotf', 'Sons Of The Forest', 'Sons Of The Forest dedicated server.', 'sotf', 'a2s', 8766, 8850, '/games/sotf.svg'),
        ('soulmask', 'Soulmask', 'Soulmask dedicated server.', 'soulmask', 'a2s', 7777, 7900, '/games/soulmask.svg'),
        ('spaceeng', 'Space Engineers', 'Space Engineers dedicated server.', 'spaceeng', 'a2s', 27016, 27100, '/games/spaceeng.svg'),
        ('squad', 'Squad', 'Squad dedicated server.', 'squad', 'a2s', 7787, 7900, '/games/squad.svg'),
        ('starrupt', 'StarRupture', 'StarRupture server profile.', 'starrupt', 'a2s', 25000, 25100, '/games/starrupt.svg'),
        ('tf2', 'Team Fortress 2', 'Team Fortress 2 dedicated server (Source)', 'tf2', 'a2s', 27015, 27060, '/games/tf2.svg'),
        ('theisle', 'The Isle', 'The Isle dedicated server.', 'theisle', 'a2s', 7777, 7900, '/games/theisle.svg'),
        ('ttworlds', 'TerraTech Worlds', 'TerraTech Worlds dedicated server.', 'ttworlds', 'a2s', 7777, 7900, '/games/ttworlds.svg'),
        ('untrm4', 'Unturned (RocketMod 4)', 'Unturned dedicated server with RocketMod 4', 'untrm4', 'a2s', 27015, 27060, '/games/untrm4.svg'),
        ('untrm5', 'Unturned (RocketMod 5)', 'Unturned dedicated server with RocketMod 5', 'untrm5', 'a2s', 27015, 27060, '/games/untrm5.svg'),
        ('unturned', 'Unturned (Vanilla)', 'Unturned dedicated server (no mods)', 'unturn', 'a2s', 27015, 27060, '/games/unturned.svg'),
        ('valheim', 'Valheim', 'Valheim dedicated server.', 'valheim', 'a2s', 2456, 2480, '/games/valheim.svg'),
        ('vrising', 'V Rising', 'V Rising dedicated server.', 'vrising', 'a2s', 9876, 9900, '/games/vrising.svg')
)
INSERT INTO core.games (tenant_id, slug, name, description, image_url, code, query_protocol, min_port, max_port, active)
SELECT t.id, c.slug, c.name, c.description, c.image_url, c.code, c.query_protocol, c.min_port, c.max_port, true
FROM core.tenants t
CROSS JOIN catalog c
ON CONFLICT (tenant_id, slug) DO UPDATE SET
    name = EXCLUDED.name,
    description = COALESCE(NULLIF(core.games.description, ''), EXCLUDED.description),

    image_url = COALESCE(core.games.image_url, EXCLUDED.image_url),
    code = COALESCE(NULLIF(core.games.code, ''), EXCLUDED.code),
    query_protocol = COALESCE(NULLIF(core.games.query_protocol, ''), EXCLUDED.query_protocol),
    min_port = EXCLUDED.min_port,
    max_port = EXCLUDED.max_port;

UPDATE core.servers s
SET game_id = 'mcjava'
WHERE s.game_id = 'minecraft'
  AND EXISTS (SELECT 1 FROM core.games g WHERE g.tenant_id = s.tenant_id AND g.slug = 'mcjava');

UPDATE core.game_versions gv
SET game_id = canon.id
FROM core.games legacy
JOIN core.games canon
  ON canon.tenant_id = legacy.tenant_id AND canon.slug = 'mcjava'
WHERE legacy.slug = 'minecraft'
  AND gv.game_id = legacy.id
  AND NOT EXISTS (
      SELECT 1 FROM core.game_versions x
      WHERE x.game_id = canon.id AND x.version = gv.version
  );

UPDATE core.tariffs t
SET game_id = canon.id
FROM core.games legacy
JOIN core.games canon
  ON canon.tenant_id = legacy.tenant_id AND canon.slug = 'mcjava'
WHERE legacy.slug = 'minecraft' AND t.game_id = legacy.id;

DELETE FROM core.games legacy
WHERE legacy.slug = 'minecraft'
  AND EXISTS (
      SELECT 1 FROM core.games canon
      WHERE canon.tenant_id = legacy.tenant_id AND canon.slug = 'mcjava'
  )
  AND NOT EXISTS (SELECT 1 FROM core.servers s WHERE s.game_id = 'minecraft')
  AND NOT EXISTS (SELECT 1 FROM core.game_versions gv WHERE gv.game_id = legacy.id);
