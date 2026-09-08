

INSERT INTO core.games (tenant_id, slug, name, description, query_protocol, min_port, max_port, active, meta)
SELECT t.id, g.slug, g.name, g.description, g.query_protocol, g.min_port, g.max_port, true, g.meta::jsonb
FROM core.tenants t
CROSS JOIN (VALUES
    ('7d2d', '7 Days to Die', '7 Days to Die dedicated server.', 'a2s', 26900, 27000, '{}'),
    ('arksa', 'Ark Survival Ascended', 'ARK: Survival Ascended dedicated server.', 'a2s', 7777, 28000, '{}'),
    ('arkse', 'ARK: Survival Evolved', 'ARK: Survival Evolved dedicated server.', 'a2s', 7777, 28000, '{}'),
    ('arma3', 'Arma 3', 'Arma 3 dedicated server.', 'a2s', 2302, 2400, '{}'),
    ('dayz', 'DayZ Standalone', 'DayZ Standalone dedicated server.', 'a2s', 2302, 2400, '{}'),
    ('factorio', 'Factorio', 'Factorio dedicated server.', 'a2s', 34197, 34297, '{}'),
    ('palworld', 'Palworld', 'Palworld dedicated server.', 'a2s', 8211, 8300, '{"steam_updatable": true}'),
    ('pzomboid', 'Project Zomboid', 'Project Zomboid dedicated server.', 'a2s', 16261, 16350, '{}'),
    ('valheim', 'Valheim', 'Valheim dedicated server.', 'a2s', 2456, 2500, '{}'),
    ('tf2', 'Team Fortress 2', 'Team Fortress 2 dedicated server.', 'a2s', 27015, 27100, '{"steam_updatable": true}'),
    ('css', 'Counter-Strike: Source', 'CSS dedicated server.', 'a2s', 27015, 27100, '{"steam_updatable": true}'),
    ('unturned', 'Unturned', 'Unturned dedicated server.', 'a2s', 27015, 27100, '{}'),
    ('crmp', 'CR:MP', 'CR:MP dedicated server.', 'a2s', 7777, 7900, '{}'),
    ('samp', 'SA-MP', 'San Andreas Multiplayer.', 'a2s', 7777, 7900, '{}'),
    ('armaref', 'Arma Reforger', 'Arma Reforger dedicated server.', 'a2s', 2001, 2100, '{}'),
    ('conan', 'Conan Exiles', 'Conan Exiles dedicated server.', 'a2s', 7777, 7900, '{}'),
    ('enshroud', 'Enshrouded', 'Enshrouded dedicated server.', 'a2s', 15637, 15700, '{}'),
    ('satisfac', 'Satisfactory', 'Satisfactory dedicated server.', 'a2s', 7777, 7900, '{}'),
    ('scum', 'SCUM', 'SCUM dedicated server.', 'a2s', 7777, 7900, '{}'),
    ('sotf', 'Sons Of The Forest', 'Sons Of The Forest dedicated server.', 'a2s', 8766, 8850, '{}'),
    ('squad', 'Squad', 'Squad dedicated server.', 'a2s', 7787, 7900, '{"steam_updatable": true}'),
    ('theisle', 'The Isle', 'The Isle dedicated server.', 'a2s', 7777, 7900, '{}')
) AS g(slug, name, description, query_protocol, min_port, max_port, meta)
ON CONFLICT (tenant_id, slug) DO NOTHING;

INSERT INTO core.game_versions (tenant_id, game_id, version, source_type, docker_image, steam_app_id, active, sort_order)
SELECT g.tenant_id, g.id, 'latest', v.source_type, v.docker_image, v.steam_app_id, true, 0
FROM core.games g
JOIN (VALUES
    ('minecraft', 'docker', 'itzg/minecraft-server', NULL::bigint),
    ('cs2', 'docker', 'cm2network/cs2', 730),
    ('rust', 'docker', 'didstopia/rust-server', 258550),
    ('gmod', 'docker', 'cm2network/gmod', 4000),
    ('tf2', 'docker', 'cm2network/tf2', 232250),
    ('css', 'docker', 'cm2network/css', 240),
    ('squad', 'steam', NULL, 393380),
    ('palworld', 'steam', NULL, 2394010)
) AS v(slug, source_type, docker_image, steam_app_id) ON v.slug = g.slug
WHERE NOT EXISTS (
    SELECT 1 FROM core.game_versions gv WHERE gv.game_id = g.id AND gv.version = 'latest'
);
