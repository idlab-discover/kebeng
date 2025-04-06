CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS public.upload
(
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    account_id uuid NOT NULL,
    snap_name text COLLATE pg_catalog."default",
    status text COLLATE pg_catalog."default",
    status_details_url text COLLATE pg_catalog."default",
    CONSTRAINT upload_pkey PRIMARY KEY (id),
    entry_id uuid constraint entry_id_fk references entry(id)
);

-- Create a function to generate the URL
CREATE OR REPLACE FUNCTION generate_upload_url()
RETURNS TRIGGER AS $$
BEGIN
    NEW.status_details_url := format('/dev/api/snaps/%s/status', NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create a trigger to call the function before inserting a new row
CREATE TRIGGER set_upload_url
BEFORE INSERT ON public.upload
FOR EACH ROW
EXECUTE FUNCTION generate_upload_url();