-- Create the uuid-ossp extension if it is not already available.
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create table for account-key assertions.
CREATE TABLE IF NOT EXISTS public.account_key_assertion (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),  
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),  
    deleted_at TIMESTAMP WITH TIME ZONE,                
    
    authority_id TEXT NOT NULL CHECK (authority_id <> ''),                 
    revision INTEGER NOT NULL,                  
    public_key_sha3_384 TEXT NOT NULL CHECK (public_key_sha3_384 <> ''),          
    account_id UUID NOT NULL,                   
    name TEXT NOT NULL CHECK (name <> ''),                         
    since TIMESTAMP WITH TIME ZONE NOT NULL,    
    body_length INTEGER NOT NULL,               
    sign_key_sha3_384 TEXT NOT NULL CHECK (sign_key_sha3_384 <> ''),            

    CONSTRAINT account_key_assertion_pkey PRIMARY KEY (id)
);
