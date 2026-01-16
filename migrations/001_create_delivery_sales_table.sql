CREATE TABLE IF NOT EXISTS delivery_sales (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date DATE NOT NULL UNIQUE,
    grubhub_subtotal REAL DEFAULT 0,
    grubhub_net REAL DEFAULT 0,
    doordash_subtotal REAL DEFAULT 0,
    doordash_net REAL DEFAULT 0,
    ubereats_earnings REAL DEFAULT 0,
    ubereats_payout REAL DEFAULT 0,
    notes TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_delivery_sales_date ON delivery_sales(date);
