-- Detected device of an operation on a session, in the columns of sessions (plan
-- revue-securite, D3): the label, in French, is composed by the interface of EF-07, not
-- fixed by the server. NULL for the other operations. The former text is not converted:
-- nothing was deployed, only development data lose it.
ALTER TABLE audit_log DROP COLUMN detected_device;
ALTER TABLE audit_log ADD COLUMN device_type TEXT;
ALTER TABLE audit_log ADD COLUMN os TEXT;
ALTER TABLE audit_log ADD COLUMN browser TEXT;
ALTER TABLE audit_log ADD COLUMN installed_app INTEGER CHECK (installed_app IN (0, 1));
