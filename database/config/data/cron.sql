CREATE EXTENSION pg_cron;

-- Clean up old warnings
SELECT cron.schedule(
  job_name  => 'cleanup_old_warnings',
  schedule  => '0 0 * * *',  -- Every day at midnight
  command   => $$DELETE FROM warnings.warnings WHERE "ends" < now() - interval '1 day';$$
);
