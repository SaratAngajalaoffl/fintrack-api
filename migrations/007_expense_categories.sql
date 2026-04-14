-- Expense categories configured by each user.

CREATE TABLE IF NOT EXISTS expense_categories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  icon_url TEXT NOT NULL,
  color TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_expense_categories_color
    CHECK (
      color IN (
        'text',
        'subtext-1',
        'subtext-0',
        'overlay-2',
        'overlay-1',
        'overlay-0',
        'surface-2',
        'surface-1',
        'surface-0',
        'base',
        'mantle',
        'crust',
        'red',
        'mauve'
      )
    )
);

CREATE INDEX IF NOT EXISTS idx_expense_categories_user_id
  ON expense_categories(user_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_expense_categories_unique_name_per_user
  ON expense_categories(user_id, name);
