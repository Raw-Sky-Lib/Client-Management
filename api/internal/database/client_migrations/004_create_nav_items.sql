CREATE TABLE IF NOT EXISTS nav_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    label       TEXT NOT NULL,
    url         TEXT NOT NULL,
    "order"     INTEGER NOT NULL DEFAULT 0,
    is_external BOOLEAN NOT NULL DEFAULT false
);
