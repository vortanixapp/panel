DELETE FROM core.tenant_settings
WHERE key LIKE 'services.license_cloud.%'
   OR key LIKE 'updates.%'
   OR key LIKE 'dockerhub.%';
