-- Create the uuid-ossp extension if it is not already available.
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create table for account-key assertions.
CREATE TABLE IF NOT EXISTS public.snap_revision_assertion (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),  
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),  
    deleted_at TIMESTAMP WITH TIME ZONE,                
    
    authority_id TEXT NOT NULL CHECK (authority_id <> ''),                 
    snap_sha3_384 TEXT NOT NULL CHECK (snap_sha3_384 <> ''),
    developer_id UUID NOT NULL,
    snap_id UUID NOT NULL,
    snap_revision INTEGER NOT NULL,                  
    snap_size INTEGER NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    sign_key_sha3_384 TEXT NOT NULL CHECK (sign_key_sha3_384 <> ''),            

    CONSTRAINT account_key_assertion_pkey PRIMARY KEY (id)
);
