-- Remove legacy preferred_categories array from credit_cards.
-- Preferred category links are canonical in credit_card_preferred_categories.

ALTER TABLE credit_cards
DROP COLUMN IF EXISTS preferred_categories;
