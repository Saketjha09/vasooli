-- Execution's recovered amount was only ever held in-memory during a
-- RunBatch call — nothing persisted it, so the API layer had no way to
-- query "money recovered" back out of the database. Add a nullable amount
-- column to audit_log, populated only on the execution agent's own row
-- (NULL for every other agent's audit entry).
ALTER TABLE audit_log ADD COLUMN amount numeric(12, 2);
