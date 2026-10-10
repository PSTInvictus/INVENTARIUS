-- +goose Up
-- Reservations say "this is already meant for something": part of a stock
-- lot, or a whole asset, set aside for a project/department. A reservation
-- is active until released, used up, or past its "until" date.
CREATE TABLE reservations (
    id            INTEGER PRIMARY KEY,
    item_id       INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    stock_id      INTEGER REFERENCES stock(id) ON DELETE CASCADE,
    asset_id      INTEGER REFERENCES assets(id) ON DELETE CASCADE,
    quantity      REAL NOT NULL CHECK (quantity >= 0),
    user_id       INTEGER NOT NULL REFERENCES users(id),
    project_id    INTEGER REFERENCES projects(id),
    department_id INTEGER REFERENCES departments(id),
    note          TEXT NOT NULL,
    until         DATE,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    released_at   DATETIME,
    CHECK ((stock_id IS NULL) != (asset_id IS NULL))
);
CREATE INDEX reservations_stock ON reservations(stock_id) WHERE released_at IS NULL;
CREATE INDEX reservations_asset ON reservations(asset_id) WHERE released_at IS NULL;
CREATE INDEX reservations_item ON reservations(item_id) WHERE released_at IS NULL;

-- SQLite cannot change a CHECK constraint in place: rebuild movements to
-- allow the new kinds and link movements to reservations.
CREATE TABLE movements_new (
    id               INTEGER PRIMARY KEY,
    at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_id          INTEGER NOT NULL REFERENCES users(id),
    kind             TEXT NOT NULL CHECK (kind IN (
                         'receive', 'take', 'return', 'move', 'adjust', 'write_off',
                         'borrow', 'give_back', 'asset_create', 'asset_update',
                         'reserve', 'unreserve')),
    item_id          INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    stock_id         INTEGER REFERENCES stock(id) ON DELETE SET NULL,
    asset_id         INTEGER REFERENCES assets(id) ON DELETE SET NULL,
    from_location_id INTEGER REFERENCES locations(id) ON DELETE SET NULL,
    to_location_id   INTEGER REFERENCES locations(id) ON DELETE SET NULL,
    quantity         REAL NOT NULL DEFAULT 0,
    project_id       INTEGER REFERENCES projects(id),
    department_id    INTEGER REFERENCES departments(id),
    note             TEXT NOT NULL DEFAULT '',
    reservation_id   INTEGER REFERENCES reservations(id) ON DELETE SET NULL
);
INSERT INTO movements_new (id, at, user_id, kind, item_id, stock_id, asset_id, from_location_id,
                           to_location_id, quantity, project_id, department_id, note)
SELECT id, at, user_id, kind, item_id, stock_id, asset_id, from_location_id,
       to_location_id, quantity, project_id, department_id, note
FROM movements;
DROP TABLE movements;
ALTER TABLE movements_new RENAME TO movements;
CREATE INDEX movements_item ON movements(item_id, at);
CREATE INDEX movements_at ON movements(at);

-- +goose Down
CREATE TABLE movements_old (
    id               INTEGER PRIMARY KEY,
    at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_id          INTEGER NOT NULL REFERENCES users(id),
    kind             TEXT NOT NULL CHECK (kind IN (
                         'receive', 'take', 'return', 'move', 'adjust', 'write_off',
                         'borrow', 'give_back', 'asset_create', 'asset_update')),
    item_id          INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    stock_id         INTEGER REFERENCES stock(id) ON DELETE SET NULL,
    asset_id         INTEGER REFERENCES assets(id) ON DELETE SET NULL,
    from_location_id INTEGER REFERENCES locations(id) ON DELETE SET NULL,
    to_location_id   INTEGER REFERENCES locations(id) ON DELETE SET NULL,
    quantity         REAL NOT NULL DEFAULT 0,
    project_id       INTEGER REFERENCES projects(id),
    department_id    INTEGER REFERENCES departments(id),
    note             TEXT NOT NULL DEFAULT ''
);
INSERT INTO movements_old
SELECT id, at, user_id, kind, item_id, stock_id, asset_id, from_location_id,
       to_location_id, quantity, project_id, department_id, note
FROM movements WHERE kind NOT IN ('reserve', 'unreserve');
DROP TABLE movements;
ALTER TABLE movements_old RENAME TO movements;
CREATE INDEX movements_item ON movements(item_id, at);
CREATE INDEX movements_at ON movements(at);
DROP TABLE reservations;
