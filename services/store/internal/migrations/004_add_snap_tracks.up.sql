CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

create table track
(
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at    timestamp with time zone,
    name          text, -- eg. "latest"
    entry_id uuid
        constraint fk_track_entry
            references entry,
    CONSTRAINT track_pkey PRIMARY KEY (id)
);

create index idx_track_deleted_at
    on track (deleted_at);