-- Move bill metadata off credit_cards into credit_card_bills.

CREATE TABLE IF NOT EXISTS credit_card_bills (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  credit_card_id UUID NOT NULL REFERENCES credit_cards(id) ON DELETE CASCADE,
  bill_generation_date DATE NOT NULL,
  bill_due_date DATE NOT NULL,
  bill_pdf_url TEXT NULL,
  is_bill_paid BOOLEAN NOT NULL DEFAULT FALSE,
  bill_payment_date DATE NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_credit_card_bills_due_after_generation
    CHECK (bill_due_date >= bill_generation_date),
  CONSTRAINT chk_credit_card_bills_payment_date_required
    CHECK (
      (is_bill_paid = FALSE AND bill_payment_date IS NULL)
      OR (is_bill_paid = TRUE AND bill_payment_date IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_card_bills_card_generation
  ON credit_card_bills (credit_card_id, bill_generation_date);

CREATE INDEX IF NOT EXISTS idx_credit_card_bills_user_id
  ON credit_card_bills (user_id);

CREATE INDEX IF NOT EXISTS idx_credit_card_bills_card_id
  ON credit_card_bills (credit_card_id);

ALTER TABLE credit_cards
  DROP COLUMN IF EXISTS previous_bill_cycle_label,
  DROP COLUMN IF EXISTS previous_bill_pdf_url,
  DROP COLUMN IF EXISTS previous_bill_paid;
