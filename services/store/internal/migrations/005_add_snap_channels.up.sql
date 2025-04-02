CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

create table channel
(
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    deleted_at timestamp with time zone,
    name          text, -- eg. "stable"
    entry_id uuid
        constraint fk_channel_entry
            references entry,
    snap_track_id uuid
        constraint fk_track_channels
            references track,

    CONSTRAINT channel_pkey PRIMARY KEY (id)
);

create index idx_channel_deleted_at
    on channel (deleted_at);

