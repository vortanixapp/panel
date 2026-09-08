

ALTER TABLE core.nodes
    ADD COLUMN IF NOT EXISTS max_servers INTEGER;

ALTER TABLE core.nodes
    ADD COLUMN IF NOT EXISTS ram_overcommit NUMERIC(4,2) NOT NULL DEFAULT 1.00
        CHECK (ram_overcommit > 0 AND ram_overcommit <= 4);

ALTER TABLE core.nodes
    ADD COLUMN IF NOT EXISTS reserved_ram_mb INTEGER NOT NULL DEFAULT 1024
        CHECK (reserved_ram_mb >= 0);
