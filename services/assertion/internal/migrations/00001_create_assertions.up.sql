-- check if manager role exists, if not create
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'manager') THEN
       CREATE ROLE manager LOGIN;
    END IF;
END
$$;

-- Enable the uuid-ossp extension for uuid_generate_v4()
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS public.assertions
(
    id UUID NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone,
    deleted_at timestamp with time zone,
    assertion TEXT UNIQUE, -- Changed from blob to text to store UTF-8 encoded data
    CONSTRAINT assertions_pkey PRIMARY KEY (id),
    CONSTRAINT assertions_id_key UNIQUE (id),

    snap_entry_id UUID NOT NULL
)
