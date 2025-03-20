CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

create table snap_channels
(
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    created_at    timestamp with time zone,
    updated_at    timestamp with time zone,
    deleted_at    timestamp with time zone,
    name          text,
    snap_track_id uuid
        constraint fk_snap_tracks_channels
            references snap_tracks,
    snap_entry_id uuid
        constraint fk_snap_channels_snap_entry
            references snap_entries,
    revision_id   uuid
        constraint fk_snap_channels_revision
            references snap_revisions,
    CONSTRAINT snap_channels_pkey PRIMARY KEY (id)
);

create index idx_snap_channels_deleted_at
    on snap_channels (deleted_at);

