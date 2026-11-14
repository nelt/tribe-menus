-- Taken quantity of an item (plan courses-suite, lot B; D1, D9, Q41).

-- What a checked computed item held when its quantity increased: it is unchecked and shows
-- what remains to take ("750 g · encore 250 g"). In the base unit of the family of its
-- quantity (ADR 0003, point 7), displayed in its unit. Only on an unchecked computed item
-- with a quantity, and below that quantity: checking the item erases it.
ALTER TABLE shopping_items ADD COLUMN taken_amount INTEGER
    CHECK (taken_amount IS NULL OR (
        taken_amount > 0 AND amount IS NOT NULL AND taken_amount < amount AND checked = 0 AND manual = 0
    ));
