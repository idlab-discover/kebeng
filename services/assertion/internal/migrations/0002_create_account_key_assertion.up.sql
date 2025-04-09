-- Create the uuid-ossp extension if it is not already available.
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create table for account-key assertions.
CREATE TABLE IF NOT EXISTS public.account_key_assertion (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),  
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),  
    deleted_at TIMESTAMP WITH TIME ZONE,                
    
    type TEXT NOT NULL,                         
    authority_id TEXT NOT NULL,                 
    revision INTEGER NOT NULL,                  
    public_key_sha3_384 TEXT NOT NULL,          
    account_id UUID NOT NULL,                   
    name TEXT NOT NULL,                         
    since TIMESTAMP WITH TIME ZONE NOT NULL,    
    body_length INTEGER NOT NULL,               
    sign_key_sha3_384 TEXT NOT NULL,            

    CONSTRAINT account_key_assertions_pkey PRIMARY KEY (id)
);
