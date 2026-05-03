ALTER TABLE users
    DROP COLUMN IF EXISTS refresh_token_hash,
    DROP COLUMN IF EXISTS refresh_token_exp;
