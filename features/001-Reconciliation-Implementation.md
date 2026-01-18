# Bank Reconciliation Implementation Plan

## Overview

Implement bank statement parsing and reconciliation for TD Bank PDF statements. This enables matching bank transactions against recorded expenses to verify all payments are accounted for.

## Current State

Already implemented:
- `bank_reconciliations` table (basic: id, statement_date, balances, status, file_path, notes)
- `BankReconciliation` model
- Filestore package for uploads
- Upload handler that saves file and creates reconciliation record
- Reconciliations list page with upload form

## Implementation Phases

---

### Phase 1: Job Queue System

Create a simple SQLite-backed background job system for async PDF processing.

**Files to create:**
- `internal/database/schema.sql` - Add jobs table
- `internal/models/job.go` - Job model
- `internal/database/jobs.go` - Job CRUD operations
- `internal/jobs/worker.go` - Background worker

**Schema:**
```sql
CREATE TABLE IF NOT EXISTS jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_type TEXT NOT NULL,
    payload TEXT DEFAULT '',
    status TEXT DEFAULT 'pending',
    progress INTEGER DEFAULT 0,
    result TEXT DEFAULT '',
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
```

**Database functions:**
- `CreateJob(jobType string, payload any) (int64, error)`
- `ClaimNextJob() (*Job, error)` - atomic SELECT + UPDATE
- `UpdateJobProgress(id int64, progress int) error`
- `CompleteJob(id int64, result string) error`
- `FailJob(id int64, errMsg string) error`
- `RetryJob(id int64) error`
- `GetJob(id int64) (*Job, error)`

**Worker:**
- Polls for pending jobs every 2 seconds
- Processes jobs with registered handlers
- Handles retries and failure states
- Graceful shutdown via stop channel

**Wire up:**
- Initialize worker in main.go
- Start worker before HTTP server
- Stop worker on shutdown

---

### Phase 2: Bank Statement Schema

Expand the schema to support parsed transactions and matching.

**Modify existing:**
- Rename/expand `bank_reconciliations` → keep as-is but add more fields OR create new `bank_statements` table

**Decision:** Keep `bank_reconciliations` as the parent record, add `bank_transactions` table for parsed line items.

**Schema additions:**
```sql
-- Add columns to bank_reconciliations
ALTER TABLE bank_reconciliations ADD COLUMN account_last_four TEXT DEFAULT '';
ALTER TABLE bank_reconciliations ADD COLUMN parse_job_id INTEGER;
ALTER TABLE bank_reconciliations ADD COLUMN parsed_at DATETIME;
ALTER TABLE bank_reconciliations ADD COLUMN reconciled_at DATETIME;

-- New table for parsed transactions
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
    match_status TEXT DEFAULT 'unmatched',
    match_confidence TEXT DEFAULT '',
    matched_at DATETIME,
    notes TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (reconciliation_id) REFERENCES bank_reconciliations(id),
    FOREIGN KEY (matched_expense_id) REFERENCES expenses(id)
);
CREATE INDEX IF NOT EXISTS idx_bank_txn_recon ON bank_transactions(reconciliation_id);
CREATE INDEX IF NOT EXISTS idx_bank_txn_status ON bank_transactions(match_status);
CREATE INDEX IF NOT EXISTS idx_bank_txn_date ON bank_transactions(posting_date);
```

**Models:**
- Update `BankReconciliation` with new fields
- Create `BankTransaction` model

**Database functions:**
- `CreateBankTransaction(reconID int64, txn *BankTransaction) (int64, error)`
- `GetBankTransactions(reconID int64) ([]BankTransaction, error)`
- `GetUnmatchedBankTransactions(reconID int64) ([]BankTransaction, error)`
- `MatchBankTransaction(txnID, expenseID int64, confidence string) error`
- `IgnoreBankTransaction(txnID int64, reason string) error`
- `UpdateReconciliationParsed(id int64, startBal, endBal float64) error`

---

### Phase 3: TD Bank PDF Parser

Create the PDF parser using `pdftotext -layout`.

**Files to create:**
- `internal/parser/tdbank.go` - TD Bank specific parser

**Dependencies:**
- Requires `poppler-utils` (pdftotext) installed on system
- Add to Dockerfile: `RUN apk add --no-cache poppler-utils`

**Parser structure:**
```go
type TDBankParser struct{}

type ParsedStatement struct {
    AccountLastFour  string
    BeginningBalance float64
    EndingBalance    float64
    Transactions     []ParsedTransaction
}

type ParsedTransaction struct {
    PostingDate     string  // YYYY-MM-DD
    Description     string
    Amount          float64 // negative for debits
    TransactionType string  // deposit, check, debit, ach, fee, transfer
    Category        string
    CheckNumber     string
    VendorHint      string
    ReferenceNumber string
}

func (p *TDBankParser) Parse(pdfPath string) (*ParsedStatement, error)
```

**Parsing strategy:**
1. Run `pdftotext -layout <file> -` to get text
2. Extract header info (account number, balances)
3. Split into sections by markers
4. Parse each section with appropriate patterns
5. Infer year from statement date
6. Extract vendor hints from descriptions

**Section markers:**
| Section | Start | End |
|---------|-------|-----|
| Electronic Deposits | `Electronic Deposits` | `Other Credits` or `Checks Paid` |
| Other Credits | `Other Credits` | `Checks Paid` |
| Checks Paid | `Checks Paid` | `Electronic Payments` |
| Electronic Payments | `Electronic Payments` | `Other Withdrawals` |
| Other Withdrawals | `Other Withdrawals` | `Service Charges` |
| Service Charges | `Service Charges` | `DAILY BALANCE SUMMARY` |

**Testing:**
- Create `cmd/parsetest/main.go` to test parser standalone
- Run against sample statement PDF

---

### Phase 4: Statement Processing Job

Create the job handler that ties parsing to the job queue.

**Files to create:**
- `internal/jobs/parse_statement.go`

**Job handler:**
1. Unmarshal payload (reconciliation_id, file_path)
2. Update reconciliation status to "parsing"
3. Run parser
4. Update reconciliation with parsed balances
5. Insert all transactions
6. Update progress throughout
7. Run auto-matching
8. Update status to "parsed"

**Update upload handler:**
- After saving file and creating reconciliation, queue parse job
- Store job_id in reconciliation record
- Return job_id to frontend for polling

---

### Phase 5: Auto-Matching Logic

Create the matcher that links bank transactions to expenses.

**Files to create:**
- `internal/reconciliation/matcher.go`

**Matching rules (in order of confidence):**
1. **Check number exact match** - If bank txn has check number matching expense check number → `auto_exact`
2. **Amount + date fuzzy** - Same amount, date within 3 days → `auto_fuzzy`
3. **Vendor hint + amount** - Known vendor + same amount → `auto_fuzzy`

**Function:**
```go
func AutoMatch(db *database.DB, reconID int64) (matched int, err error)
```

**Called:**
- At end of parse job
- Can be manually triggered via "Re-match" button

---

### Phase 6: HTTP Handlers & API

Add handlers for the reconciliation workflow.

**New handlers:**
- `GET /reconciliations/{id}` - View reconciliation detail with transactions
- `GET /reconciliations/{id}/edit` - Edit/reconcile interface
- `POST /reconciliations/{id}/match` - Manually match transaction to expense
- `POST /reconciliations/{id}/ignore` - Mark transaction as ignored
- `POST /reconciliations/{id}/create-expense` - Create expense from transaction
- `GET /api/jobs/{id}` - Poll job status (JSON)

**Update existing:**
- `POST /reconciliations/upload` - Queue parse job, return job_id

---

### Phase 7: UI Templates

Create the reconciliation UI.

**Templates to create:**
- `reconciliation_detail.html` - View completed reconciliation
- `reconciliation_edit.html` - Active reconciliation workspace

**UI sections:**
1. **Header** - Statement date, balances, status, progress bar (during parsing)
2. **Summary cards** - Total matched, unmatched, ignored counts
3. **Transactions table** - Sortable/filterable list
   - Date, description, amount, type, status, matched expense, actions
4. **Match modal** - Select expense to match
5. **Create expense form** - Quick-create from transaction

**JavaScript:**
- Job status polling during parse
- Match modal interactions
- HTMX for partial updates (optional, can use full page reloads)

---

## Implementation Order

1. **Phase 1: Job Queue** - Foundation for async work
2. **Phase 2: Schema** - Data model for transactions
3. **Phase 3: Parser** - PDF extraction (test standalone first)
4. **Phase 4: Parse Job** - Wire parser to job queue
5. **Phase 5: Auto-Match** - Automatic expense linking
6. **Phase 6: Handlers** - HTTP endpoints
7. **Phase 7: UI** - User interface

---

## Files Summary

**Create:**
```
internal/
├── jobs/
│   ├── worker.go
│   └── parse_statement.go
├── parser/
│   └── tdbank.go
├── reconciliation/
│   └── matcher.go
└── database/
    ├── jobs.go
    └── bank_transactions.go

web/templates/
├── reconciliation_detail.html
└── reconciliation_edit.html

cmd/parsetest/
└── main.go
```

**Modify:**
```
internal/database/schema.sql      - Add jobs table, bank_transactions table
internal/models/models.go         - Add Job, BankTransaction models
internal/handlers/handlers.go     - Add reconciliation handlers
cmd/server/main.go                - Initialize worker, add routes
```

---

## Dependencies

- `poppler-utils` for pdftotext (system package)
- No new Go dependencies required

---

## Testing Checklist

- [ ] Job queue creates, claims, completes, fails jobs correctly
- [ ] Parser extracts correct balances from sample PDF
- [ ] Parser extracts all transaction types
- [ ] Year inference works for Dec/Jan boundary
- [ ] Auto-matching links check numbers correctly
- [ ] Auto-matching links by amount+date
- [ ] Manual matching works
- [ ] Create expense from transaction works
- [ ] UI shows correct counts and statuses
- [ ] Job progress polling works
