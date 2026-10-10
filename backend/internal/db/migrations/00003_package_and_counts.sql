-- +goose Up
-- Package/footprint for electronic parts ("0603", "SOT-23"): together with
-- the name it identifies a part, e.g. "Resistor 10k" in 0402 vs 0603.
ALTER TABLE items ADD COLUMN package TEXT NOT NULL DEFAULT '';

-- How well a stock lot's quantity is known:
--   exact     counted
--   approx    an estimate ("~10"); taking more than it says empties it
--   uncounted plenty, nobody counted (screws, nuts); quantity is meaningless
ALTER TABLE stock ADD COLUMN count_state TEXT NOT NULL DEFAULT 'exact'
    CHECK (count_state IN ('exact', 'approx', 'uncounted'));

-- +goose Down
ALTER TABLE stock DROP COLUMN count_state;
ALTER TABLE items DROP COLUMN package;
