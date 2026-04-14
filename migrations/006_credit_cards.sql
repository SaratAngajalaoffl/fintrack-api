-- Credit cards, including preferred categories and previous bill metadata.

CREATE TABLE IF NOT EXISTS credit_cards (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  max_balance NUMERIC(14,2) NOT NULL DEFAULT 0,
  used_balance NUMERIC(14,2) NOT NULL DEFAULT 0,
  locked_balance NUMERIC(14,2) NOT NULL DEFAULT 0,
  preferred_categories TEXT[] NOT NULL DEFAULT '{}',
  bill_generation_day SMALLINT NOT NULL,
  bill_due_day SMALLINT NOT NULL,
  previous_bill_cycle_label TEXT NULL,
  previous_bill_pdf_url TEXT NULL,
  previous_bill_paid BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_credit_cards_bill_generation_day
    CHECK (bill_generation_day >= 1 AND bill_generation_day <= 31),
  CONSTRAINT chk_credit_cards_bill_due_day
    CHECK (bill_due_day >= 1 AND bill_due_day <= 31),
  CONSTRAINT chk_credit_cards_balances_non_negative
    CHECK (
      max_balance >= 0
      AND used_balance >= 0
      AND locked_balance >= 0
    )
);

CREATE INDEX IF NOT EXISTS idx_credit_cards_user_id
  ON credit_cards(user_id);
