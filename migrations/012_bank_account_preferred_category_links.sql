-- Normalize bank account preferred categories into a join table.

CREATE TABLE IF NOT EXISTS bank_account_preferred_categories (
  bank_account_id UUID NOT NULL REFERENCES bank_accounts(id) ON DELETE CASCADE,
  expense_category_id UUID NOT NULL REFERENCES expense_categories(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (bank_account_id, expense_category_id)
);

CREATE INDEX IF NOT EXISTS idx_bank_account_preferred_categories_user_id
  ON bank_account_preferred_categories(user_id);

CREATE INDEX IF NOT EXISTS idx_bank_account_preferred_categories_category_id
  ON bank_account_preferred_categories(expense_category_id);
