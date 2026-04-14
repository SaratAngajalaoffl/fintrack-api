-- Restrict expense category colors to selected Catppuccin Mocha accents.

ALTER TABLE expense_categories
  DROP CONSTRAINT IF EXISTS chk_expense_categories_color;

UPDATE expense_categories
SET color = 'mauve'
WHERE color NOT IN (
  'rosewater',
  'flamingo',
  'pink',
  'mauve',
  'red',
  'maroon',
  'peach',
  'yellow',
  'green',
  'teal',
  'sky',
  'sapphire',
  'blue',
  'lavender'
);

ALTER TABLE expense_categories
  ADD CONSTRAINT chk_expense_categories_color
  CHECK (
    color IN (
      'rosewater',
      'flamingo',
      'pink',
      'mauve',
      'red',
      'maroon',
      'peach',
      'yellow',
      'green',
      'teal',
      'sky',
      'sapphire',
      'blue',
      'lavender'
    )
  );
