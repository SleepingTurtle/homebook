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
    due_date DATE,
    date_paid DATE,
    notes TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS payroll_weeks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(period_start, period_end)
);

CREATE TABLE IF NOT EXISTS payroll (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    week_id INTEGER NOT NULL REFERENCES payroll_weeks(id),
    employee_id INTEGER NOT NULL REFERENCES employees(id),
    total_hours REAL NOT NULL,
    hourly_rate REAL NOT NULL,
    payment_method TEXT CHECK(payment_method IN ('cash', 'check')) NOT NULL,
    check_number TEXT DEFAULT '',
    status TEXT CHECK(status IN ('paid', 'not_paid')) DEFAULT 'not_paid',
    date_paid DATE,
    notes TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(week_id, employee_id)
);

CREATE TABLE IF NOT EXISTS bank_reconciliations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    statement_date DATE NOT NULL,
    starting_balance REAL NOT NULL DEFAULT 0,
    ending_balance REAL NOT NULL DEFAULT 0,
    status TEXT CHECK(status IN ('pending', 'parsing', 'parsed', 'reconciling', 'completed')) DEFAULT 'pending',
    file_path TEXT DEFAULT '',
    account_last_four TEXT DEFAULT '',
    parse_job_id INTEGER,
    parsed_at DATETIME,
    reconciled_at DATETIME,
    notes TEXT DEFAULT '',
    -- Statement summary totals (from PDF)
    electronic_deposits REAL DEFAULT 0,
    electronic_payments REAL DEFAULT 0,
    checks_paid REAL DEFAULT 0,
    service_fees REAL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS bank_transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reconciliation_id INTEGER NOT NULL,
    posting_date DATE NOT NULL,
    description TEXT NOT NULL,
    amount REAL NOT NULL,
    transaction_type TEXT NOT NULL,
    category TEXT DEFAULT '',
    check_number TEXT DEFAULT '',
    vendor_hint TEXT DEFAULT '',
    reference_number TEXT DEFAULT '',
    matched_expense_id INTEGER,
    match_status TEXT CHECK(match_status IN ('unmatched', 'matched', 'ignored', 'created')) DEFAULT 'unmatched',
    match_confidence TEXT DEFAULT '',
    matched_at DATETIME,
    notes TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (reconciliation_id) REFERENCES bank_reconciliations(id),
    FOREIGN KEY (matched_expense_id) REFERENCES expenses(id)
);

CREATE TABLE IF NOT EXISTS jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_type TEXT NOT NULL,
    payload TEXT DEFAULT '',
    status TEXT CHECK(status IN ('pending', 'running', 'completed', 'failed')) DEFAULT 'pending',
    progress INTEGER DEFAULT 0,
    result TEXT DEFAULT '',
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME
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
CREATE INDEX IF NOT EXISTS idx_payroll_weeks_period ON payroll_weeks(period_start, period_end);
CREATE INDEX IF NOT EXISTS idx_payroll_week_id ON payroll(week_id);
CREATE INDEX IF NOT EXISTS idx_payroll_status ON payroll(status);
CREATE INDEX IF NOT EXISTS idx_payroll_employee_id ON payroll(employee_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_reconciliations_date ON bank_reconciliations(statement_date);
CREATE INDEX IF NOT EXISTS idx_reconciliations_status ON bank_reconciliations(status);
CREATE INDEX IF NOT EXISTS idx_bank_txn_recon ON bank_transactions(reconciliation_id);
CREATE INDEX IF NOT EXISTS idx_bank_txn_status ON bank_transactions(match_status);
CREATE INDEX IF NOT EXISTS idx_bank_txn_date ON bank_transactions(posting_date);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
