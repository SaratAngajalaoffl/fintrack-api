-- Bank accounts.

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'bank_account_type') THEN
    CREATE TYPE bank_account_type AS ENUM ('savings', 'current');
  END IF;
END
$$;

CREATE TABLE IF NOT EXISTS bank_accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  account_type bank_account_type NOT NULL,
  initial_balance NUMERIC(14,2) NOT NULL DEFAULT 0,
  balance NUMERIC(14,2) NOT NULL DEFAULT 0,
  last_debit_at TIMESTAMPTZ NULL,
  last_credit_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bank_accounts_user_id
  ON bank_accounts(user_id);

CREATE INDEX IF NOT EXISTS idx_bank_accounts_user_type
  ON bank_accounts(user_id, account_type);
