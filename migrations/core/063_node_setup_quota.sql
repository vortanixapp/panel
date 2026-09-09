ALTER TABLE core.node_setups
    DROP CONSTRAINT IF EXISTS node_setups_component_check;

ALTER TABLE core.node_setups
    ADD CONSTRAINT node_setups_component_check
    CHECK (component IN ('packages', 'docker', 'mysql', 'ftp', 'quota', 'daemon', 'agent'));
