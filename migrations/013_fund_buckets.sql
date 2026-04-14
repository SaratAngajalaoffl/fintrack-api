-- Add fund buckets that lock and release portions of bank-account balances.

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'fund_bucket_priority') THEN
    CREATE TYPE fund_bucket_priority AS ENUM ('high', 'medium', 'low');
  END IF;
END
$$;

CREATE TABLE IF NOT EXISTS fund_buckets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  target_amount NUMERIC(14,2) NOT NULL,
  bank_account_id UUID NOT NULL REFERENCES bank_accounts(id) ON DELETE CASCADE,
  current_value NUMERIC(14,2) NOT NULL DEFAULT 0,
  is_locked BOOLEAN NOT NULL DEFAULT TRUE,
  priority fund_bucket_priority NOT NULL DEFAULT 'medium',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_fund_buckets_target_amount_positive CHECK (target_amount > 0),
  CONSTRAINT chk_fund_buckets_current_value_nonnegative CHECK (current_value >= 0)
);

CREATE INDEX IF NOT EXISTS idx_fund_buckets_user_id
  ON fund_buckets(user_id);

CREATE INDEX IF NOT EXISTS idx_fund_buckets_bank_account_id
  ON fund_buckets(bank_account_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_fund_buckets_unique_name_per_account
  ON fund_buckets(user_id, bank_account_id, name);
