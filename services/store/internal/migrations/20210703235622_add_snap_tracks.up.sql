CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

create table snap_tracks
(
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at    timestamp with time zone,
    name          text, -- eg. "latest"
    snap_entry_id uuid
        constraint fk_snap_tracks_snap_entry
            references snap_entries,
    CONSTRAINT snap_tracks_pkey PRIMARY KEY (id)
);

create index idx_snap_tracks_deleted_at
    on snap_tracks (deleted_at);