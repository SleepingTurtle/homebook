-- HomeBooks Database Schema

CREATE TABLE IF NOT EXISTS vendors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    category TEXT DEFAULT '',
    description TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS employees (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    hourly_rate REAL NOT NULL,
    payment_method TEXT CHECK(payment_method IN ('cash', 'check')) DEFAULT 'cash',
    active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS daily_sales (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date DATE NOT NULL,
    shift TEXT CHECK(shift IN ('breakfast', 'lunch', 'dinner')) NOT NULL,
    net_sales REAL NOT NULL,
    taxes REAL NOT NULL,
    credit_card REAL NOT NULL,
    cash_receipt REAL NOT NULL,
    cash_on_hand REAL NOT NULL,
    notes TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(date, shift)
);

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

CREATE TABLE IF NOT EXISTS expenses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date DATE NOT NULL,
    vendor_id INTEGER NOT NULL REFERENCES vendors(id),
    amount REAL NOT NULL,
    invoice_number TEXT DEFAULT '',
    status TEXT CHECK(status IN ('paid', 'not_paid')) DEFAULT 'not_paid',
    payment_type TEXT CHECK(payment_type IN ('cash', 'check', 'debit', 'credit', '')) DEFAULT '',
    check_number TEXT DEFAULT '',
    date_opened DATE,
    date_paid DATE,
    notes TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS payroll (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    employee_id INTEGER NOT NULL REFERENCES employees(id),
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    total_hours REAL NOT NULL,
    hourly_rate REAL NOT NULL,
    payment_method TEXT CHECK(payment_method IN ('cash', 'check')) NOT NULL,
    check_number TEXT DEFAULT '',
    status TEXT CHECK(status IN ('paid', 'not_paid')) DEFAULT 'not_paid',
    date_paid DATE,
    notes TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_daily_sales_date ON daily_sales(date);
CREATE INDEX IF NOT EXISTS idx_delivery_sales_date ON delivery_sales(date);
CREATE INDEX IF NOT EXISTS idx_expenses_date ON expenses(date);
CREATE INDEX IF NOT EXISTS idx_expenses_status ON expenses(status);
CREATE INDEX IF NOT EXISTS idx_expenses_vendor_id ON expenses(vendor_id);
CREATE INDEX IF NOT EXISTS idx_payroll_period ON payroll(period_start, period_end);
CREATE INDEX IF NOT EXISTS idx_payroll_status ON payroll(status);
CREATE INDEX IF NOT EXISTS idx_payroll_employee_id ON payroll(employee_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
