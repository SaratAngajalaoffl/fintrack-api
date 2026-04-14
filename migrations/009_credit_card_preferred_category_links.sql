-- Normalize credit card preferred categories into a join table.

CREATE TABLE IF NOT EXISTS credit_card_preferred_categories (
  credit_card_id UUID NOT NULL REFERENCES credit_cards(id) ON DELETE CASCADE,
  expense_category_id UUID NOT NULL REFERENCES expense_categories(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (credit_card_id, expense_category_id)
);

CREATE INDEX IF NOT EXISTS idx_credit_card_preferred_categories_user_id
  ON credit_card_preferred_categories(user_id);

CREATE INDEX IF NOT EXISTS idx_credit_card_preferred_categories_category_id
  ON credit_card_preferred_categories(expense_category_id);

-- Backfill links from the existing text[] column where names match current user categories.
INSERT INTO credit_card_preferred_categories (
  credit_card_id,
  expense_category_id,
  user_id
)
SELECT
  cc.id AS credit_card_id,
  ec.id AS expense_category_id,
  cc.user_id
FROM credit_cards cc
CROSS JOIN LATERAL unnest(COALESCE(cc.preferred_categories, '{}')) AS category_name(name)
JOIN expense_categories ec
  ON ec.user_id = cc.user_id
 AND ec.name = category_name.name
ON CONFLICT (credit_card_id, expense_category_id) DO NOTHING;
