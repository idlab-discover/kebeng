CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

create table snap_branches
(
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    created_at    timestamp with time zone,
    updated_at    timestamp with time zone,
    deleted_at    timestamp with time zone,
    name          text,
    snap_risk_id  uuid
        constraint fk_snap_channels_branches
            references snap_channels,
    snap_entry_id uuid
        constraint fk_snap_branches_snap_entry
            references snap_entries,
    revision_id   uuid
        constraint fk_snap_branches_revision
            references snap_revisions,
    CONSTRAINT snap_branches_pkey PRIMARY KEY (id)
);

create index idx_snap_branches_deleted_at
    on snap_branches (deleted_at);
