UPDATE core.games
SET max_port = 26999
WHERE slug IN ('mcjava', 'mcpaper', 'mcspigot', 'mcforge', 'mcfabric', 'minecraft')
  AND min_port = 25565 AND max_port = 25600;

UPDATE core.games
SET max_port = 19999
WHERE slug IN ('mcbedrock', 'mcbedrk', 'bedrock')
  AND min_port = 19132 AND max_port = 19160;
