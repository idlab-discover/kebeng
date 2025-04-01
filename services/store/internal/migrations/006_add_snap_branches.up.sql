CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

create table snap_branches
(
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    name          text,
    entry_id uuid
        constraint fk_snap_branches_entry
            references entry,
    snap_track_id uuid
        constraint fk_track_branches
            references track,
    snap_channel_id  uuid
        constraint fk_channel_branches
            references channel,
    CONSTRAINT snap_branches_pkey PRIMARY KEY (id)
);

create index idx_snap_branches_deleted_at
    on snap_branches (deleted_at);
