-- +goose Up
-- Ownership: an item, stock lot or asset belongs to the whole team (both
-- owner columns NULL), to one project, or to one department. Never both.

CREATE TABLE projects (
    id   INTEGER PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL
);

CREATE TABLE departments (
    id         INTEGER PRIMARY KEY,
    project_id INTEGER NOT NULL REFERENCES projects(id),
    name       TEXT NOT NULL,
    UNIQUE (project_id, name)
);

CREATE TABLE users (
    id         INTEGER PRIMARY KEY,
    email      TEXT NOT NULL UNIQUE COLLATE NOCASE,
    name       TEXT NOT NULL,
    picture    TEXT NOT NULL DEFAULT '',
    role       TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('member', 'dept_lead', 'admin')),
    active     BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_departments (
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    department_id INTEGER NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, department_id)
);

CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
);

-- Label code: L<id>
CREATE TABLE locations (
    id          INTEGER PRIMARY KEY,
    parent_id   INTEGER REFERENCES locations(id),
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE categories (
    id        INTEGER PRIMARY KEY,
    parent_id INTEGER REFERENCES categories(id),
    name      TEXT NOT NULL
);

-- Label code: I<id>
CREATE TABLE items (
    id                  INTEGER PRIMARY KEY,
    name                TEXT NOT NULL,
    kind                TEXT NOT NULL CHECK (kind IN ('consumable', 'tool', 'hazmat', 'flight_hw')),
    category_id         INTEGER REFERENCES categories(id),
    description         TEXT NOT NULL DEFAULT '',
    manufacturer        TEXT NOT NULL DEFAULT '',
    mpn                 TEXT NOT NULL DEFAULT '',
    supplier_url        TEXT NOT NULL DEFAULT '',
    datasheet_url       TEXT NOT NULL DEFAULT '',
    unit                TEXT NOT NULL DEFAULT 'pcs',
    min_stock           REAL,
    photo               TEXT NOT NULL DEFAULT '',
    hazard_class        TEXT NOT NULL DEFAULT '',
    storage_notes       TEXT NOT NULL DEFAULT '',
    owner_project_id    INTEGER REFERENCES projects(id),
    owner_department_id INTEGER REFERENCES departments(id),
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (owner_project_id IS NULL OR owner_department_id IS NULL)
);
CREATE INDEX items_category ON items(category_id);

-- Counted quantities of an item in one place. Label code: S<id>
CREATE TABLE stock (
    id                  INTEGER PRIMARY KEY,
    item_id             INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    location_id         INTEGER NOT NULL REFERENCES locations(id),
    quantity            REAL NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    lot                 TEXT NOT NULL DEFAULT '',
    expires_on          DATE,
    owner_project_id    INTEGER REFERENCES projects(id),
    owner_department_id INTEGER REFERENCES departments(id),
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (owner_project_id IS NULL OR owner_department_id IS NULL)
);
CREATE INDEX stock_item ON stock(item_id);
CREATE INDEX stock_location ON stock(location_id);

-- One physical, individually tracked unit (tools, flight hardware). Label code: A<id>
CREATE TABLE assets (
    id                  INTEGER PRIMARY KEY,
    item_id             INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    serial              TEXT NOT NULL DEFAULT '',
    location_id         INTEGER REFERENCES locations(id),
    condition           TEXT NOT NULL DEFAULT '',
    status              TEXT NOT NULL DEFAULT 'available'
                        CHECK (status IN ('available', 'on_loan', 'broken', 'in_repair', 'retired')),
    notes               TEXT NOT NULL DEFAULT '',
    owner_project_id    INTEGER REFERENCES projects(id),
    owner_department_id INTEGER REFERENCES departments(id),
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (owner_project_id IS NULL OR owner_department_id IS NULL)
);
CREATE INDEX assets_item ON assets(item_id);
CREATE INDEX assets_location ON assets(location_id);

CREATE TABLE loans (
    id            INTEGER PRIMARY KEY,
    asset_id      INTEGER NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    user_id       INTEGER NOT NULL REFERENCES users(id),
    project_id    INTEGER REFERENCES projects(id),
    department_id INTEGER REFERENCES departments(id),
    taken_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    due_at        DATETIME,
    returned_at   DATETIME,
    note          TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX loans_one_open_per_asset ON loans(asset_id) WHERE returned_at IS NULL;
CREATE INDEX loans_user ON loans(user_id);

-- Append-only audit log. quantity is the signed change for stock movements.
CREATE TABLE movements (
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
CREATE INDEX movements_item ON movements(item_id, at);
CREATE INDEX movements_at ON movements(at);

CREATE TABLE print_jobs (
    id         INTEGER PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_id    INTEGER NOT NULL REFERENCES users(id),
    code       TEXT NOT NULL,
    symbology  TEXT NOT NULL CHECK (symbology IN ('datamatrix', 'qr', 'code128')),
    caption    TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'queued'
               CHECK (status IN ('queued', 'printing', 'done', 'failed', 'cancelled')),
    attempts   INTEGER NOT NULL DEFAULT 0,
    error      TEXT NOT NULL DEFAULT '',
    printed_at DATETIME
);
CREATE INDEX print_jobs_status ON print_jobs(status, id);

INSERT INTO projects (id, code, name) VALUES (1, 'INVICTUS', 'INVICTUS'), (2, 'ICARUS', 'ICARUS');
INSERT INTO departments (project_id, name) VALUES
    (1, 'Structures'),
    (1, 'Ground'),
    (1, 'Electronics & Software'),
    (1, 'Propulsion'),
    (1, 'Recovery & Payload'),
    (1, 'Aerodynamics'),
    (2, 'Telemetry'),
    (2, 'Power Systems'),
    (2, 'Payload'),
    (2, 'Control'),
    (2, 'On Board Computer'),
    (2, 'Structures & Thermal Control');

-- +goose Down
DROP TABLE print_jobs;
DROP TABLE movements;
DROP TABLE loans;
DROP TABLE assets;
DROP TABLE stock;
DROP TABLE items;
DROP TABLE categories;
DROP TABLE locations;
DROP TABLE sessions;
DROP TABLE user_departments;
DROP TABLE users;
DROP TABLE departments;
DROP TABLE projects;
