# HomeBooks: Bank Reconciliation Feature

## Context

HomeBooks is a Go web application for back-of-house restaurant management. It currently has:
- Daily sales tracking (POS Z-readouts)
- Expense management (vendors, invoices, payment status)
- Payroll tracking
- File upload infrastructure (files table, local filesystem storage)
- A reconciliation page with PDF uploader

Now we need to add **bank statement parsing and reconciliation**. The restaurant uses **TD Bank Business Premier Checking** and downloads monthly PDF statements.

## What Needs to Be Built

### 1. Job Queue (SQLite-backed)

A simple background job system for async PDF processing.

**Schema:**

```sql
CREATE TABLE jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_type TEXT NOT NULL,              -- 'parse_statement'
    payload TEXT,                        -- JSON: {"statement_id": 123, "file_path": "..."}
    status TEXT DEFAULT 'pending',       -- pending, running, completed, failed
    progress INTEGER DEFAULT 0,          -- 0-100 percentage
    result TEXT,                         -- JSON result or error message
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME
);

CREATE INDEX idx_jobs_status ON jobs(status);
```

**Worker Implementation:**

```go
// internal/jobs/worker.go

type Worker struct {
    db       *database.DB
    handlers map[string]JobHandler
    stop     chan struct{}
}

type JobHandler func(ctx context.Context, job *Job) error

func (w *Worker) Start() {
    go func() {
        for {
            select {
            case <-w.stop:
                return
            default:
                job := w.db.ClaimNextJob()
                if job == nil {
                    time.Sleep(2 * time.Second)
                    continue
                }
                w.processJob(job)
            }
        }
    }()
}

func (w *Worker) processJob(job *Job) {
    handler, ok := w.handlers[job.Type]
    if !ok {
        w.db.FailJob(job.ID, "unknown job type")
        return
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    if err := handler(ctx, job); err != nil {
        if job.Attempts >= job.MaxAttempts {
            w.db.FailJob(job.ID, err.Error())
        } else {
            w.db.RetryJob(job.ID)
        }
        return
    }
    
    w.db.CompleteJob(job.ID)
}
```

**Database Operations:**

```go
// internal/database/jobs.go

func (db *DB) CreateJob(jobType string, payload interface{}) (int64, error)
func (db *DB) ClaimNextJob() *Job  // SELECT + UPDATE in transaction
func (db *DB) UpdateJobProgress(id int64, progress int) error
func (db *DB) CompleteJob(id int64) error
func (db *DB) FailJob(id int64, errMsg string) error
func (db *DB) RetryJob(id int64) error
func (db *DB) GetJob(id int64) (*Job, error)
```

### 2. Bank Statement Schema

```sql
CREATE TABLE bank_statements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL,            -- FK to files table
    statement_month DATE NOT NULL,       -- First day of month (2025-12-01)
    bank_name TEXT DEFAULT 'TD Bank',
    account_number TEXT,                 -- Last 4 digits only
    beginning_balance REAL,
    ending_balance REAL,
    status TEXT DEFAULT 'pending',       -- pending, parsing, parsed, reconciling, reconciled
    parsed_at DATETIME,
    reconciled_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (file_id) REFERENCES files(id),
    UNIQUE(statement_month)              -- One statement per month
);

CREATE TABLE bank_transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    statement_id INTEGER NOT NULL,
    posting_date DATE NOT NULL,
    description TEXT NOT NULL,           -- Full raw description
    amount REAL NOT NULL,                -- Negative for debits, positive for credits
    transaction_type TEXT NOT NULL,      -- deposit, check, debit, ach, fee, transfer
    category TEXT,                       -- income_cards, income_delivery, expense, fee, transfer
    check_number TEXT,                   -- For checks only
    vendor_hint TEXT,                    -- Extracted vendor name (best guess)
    reference_number TEXT,               -- Any reference/confirmation numbers
    
    -- Reconciliation fields
    matched_expense_id INTEGER,          -- FK to expenses
    match_status TEXT DEFAULT 'unmatched', -- unmatched, matched, ignored, created
    match_confidence TEXT,               -- auto_exact, auto_fuzzy, manual
    matched_at DATETIME,
    notes TEXT,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (statement_id) REFERENCES bank_statements(id),
    FOREIGN KEY (matched_expense_id) REFERENCES expenses(id)
);

CREATE INDEX idx_bank_transactions_statement ON bank_transactions(statement_id);
CREATE INDEX idx_bank_transactions_status ON bank_transactions(match_status);
CREATE INDEX idx_bank_transactions_date ON bank_transactions(posting_date);
```

### 3. TD Bank PDF Parser

The statement is a searchable PDF from TD Bank. Use `pdftotext -layout` to extract text while preserving columnar structure.

**Install dependency in Docker:**

```dockerfile
RUN apk add --no-cache poppler-utils
```

**Parser Structure:**

```go
// internal/parser/tdbank.go

type TDBankParser struct{}

type ParsedStatement struct {
    AccountNumber    string
    StatementPeriod  string
    BeginningBalance float64
    EndingBalance    float64
    Transactions     []ParsedTransaction
}

type ParsedTransaction struct {
    PostingDate     string
    Description     string
    Amount          float64
    TransactionType string  // deposit, check, debit, ach, fee, transfer
    Category        string
    CheckNumber     string
    VendorHint      string
    ReferenceNumber string
}

func (p *TDBankParser) Parse(pdfPath string) (*ParsedStatement, error) {
    // 1. Run pdftotext
    text, err := p.extractText(pdfPath)
    if err != nil {
        return nil, err
    }
    
    // 2. Extract header info (account number, balances)
    stmt := p.parseHeader(text)
    
    // 3. Split into sections and parse each
    sections := p.splitSections(text)
    
    stmt.Transactions = append(stmt.Transactions, p.parseDeposits(sections["deposits"])...)
    stmt.Transactions = append(stmt.Transactions, p.parseCredits(sections["credits"])...)
    stmt.Transactions = append(stmt.Transactions, p.parseChecks(sections["checks"])...)
    stmt.Transactions = append(stmt.Transactions, p.parsePayments(sections["payments"])...)
    stmt.Transactions = append(stmt.Transactions, p.parseWithdrawals(sections["withdrawals"])...)
    stmt.Transactions = append(stmt.Transactions, p.parseFees(sections["fees"])...)
    
    return stmt, nil
}

func (p *TDBankParser) extractText(pdfPath string) (string, error) {
    cmd := exec.Command("pdftotext", "-layout", pdfPath, "-")
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("pdftotext failed: %w", err)
    }
    return string(output), nil
}
```

**Section Markers in TD Bank Statements:**

| Section | Start Marker | End Marker |
|---------|--------------|------------|
| Header | `ACCOUNT SUMMARY` | `DAILY ACCOUNT ACTIVITY` |
| Electronic Deposits | `Electronic Deposits` | `Other Credits` or `Checks Paid` |
| Other Credits | `Other Credits` | `Checks Paid` |
| Checks Paid | `Checks Paid` | `Electronic Payments` |
| Electronic Payments | `Electronic Payments` | `Other Withdrawals` |
| Other Withdrawals | `Other Withdrawals` | `Service Charges` |
| Service Charges | `Service Charges` | `DAILY BALANCE SUMMARY` |

**Transaction Patterns:**

```go
// Electronic Deposits - single line
// 12/01 CCD DEPOSIT, BANKCARD MTOT DEP 548298210013492 1,148.90
var depositPattern = regexp.MustCompile(`^(\d{2}/\d{2})\s+(.+?)\s+([\d,]+\.\d{2})\s*$`)

// Checks - table format
// 12/01 2730 500.00
var checkPattern = regexp.MustCompile(`(\d{2}/\d{2})\s+(\d+)\*?\s+([\d,]+\.\d{2})`)

// Electronic Payments - multi-line, amount on first line
// 12/02 DEBIT POS AP, AUT 120225 DDA PURCHASE AP                    1,525.50
//       JETRO CASH CARRY BROOKLYN * NY
//       4085404039877380
var paymentFirstLine = regexp.MustCompile(`^(\d{2}/\d{2})\s+(.+?)\s+([\d,]+\.\d{2})\s*$`)
var paymentContinuation = regexp.MustCompile(`^\s{6,}(\S.*)$`)
```

**Vendor Extraction:**

Extract clean vendor names from messy bank descriptions:

```go
var vendorPatterns = map[string]*regexp.Regexp{
    "Jetro":         regexp.MustCompile(`(?i)JETRO\s*CASH\s*CARRY`),
    "Chef's Choice": regexp.MustCompile(`(?i)CHEF.?S?\s*CHOICE`),
    "Cogent Waste":  regexp.MustCompile(`(?i)COGENT\s*WASTE`),
    "Con Edison":    regexp.MustCompile(`(?i)CON\s*ED`),
    "National Grid": regexp.MustCompile(`(?i)NGRID|NATIONAL\s*GRID`),
    "Uber Eats":     regexp.MustCompile(`(?i)UBER\s*(USA|EATS)?`),
    "Grubhub":       regexp.MustCompile(`(?i)GRUBHUB`),
    "Verizon":       regexp.MustCompile(`(?i)VERIZON`),
    "AT&T":          regexp.MustCompile(`(?i)\bATT\b|AT&T`),
    "Sampar's":      regexp.MustCompile(`(?i)SAMPARS?`),
    "Clover":        regexp.MustCompile(`(?i)CLOVER\s*FEE`),
    "Cintas":        regexp.MustCompile(`(?i)CINTAS`),
}

func extractVendorHint(description string) string {
    for vendor, pattern := range vendorPatterns {
        if pattern.MatchString(description) {
            return vendor
        }
    }
    // Fallback: extract first recognizable business name
    // Look for patterns like "BUSINESS NAME CITY * STATE"
    return ""
}
```

**Category Detection:**

```go
func categorizeTransaction(txnType, description string, amount float64) string {
    if amount > 0 {
        // Credits
        if strings.Contains(description, "BANKCARD") {
            return "income_cards"
        }
        if strings.Contains(description, "UBER") || strings.Contains(description, "GRUBHUB") {
            return "income_delivery"
        }
        if strings.Contains(description, "REFUND") {
            return "refund"
        }
        return "income_other"
    }
    
    // Debits
    if txnType == "check" {
        return "expense_check"
    }
    if txnType == "fee" {
        return "fee"
    }
    if strings.Contains(description, "Transfer") {
        return "transfer"
    }
    return "expense"
}
```

### 4. Statement Processing Job

```go
// internal/jobs/parse_statement.go

func ParseStatementHandler(db *database.DB, parser *parser.TDBankParser) JobHandler {
    return func(ctx context.Context, job *Job) error {
        var payload struct {
            StatementID int64  `json:"statement_id"`
            FilePath    string `json:"file_path"`
        }
        json.Unmarshal([]byte(job.Payload), &payload)
        
        // Update status
        db.UpdateBankStatementStatus(payload.StatementID, "parsing")
        db.UpdateJobProgress(job.ID, 10)
        
        // Parse PDF
        result, err := parser.Parse(payload.FilePath)
        if err != nil {
            return fmt.Errorf("parse failed: %w", err)
        }
        db.UpdateJobProgress(job.ID, 50)
        
        // Update statement with parsed header info
        db.UpdateBankStatementParsed(payload.StatementID, result.BeginningBalance, result.EndingBalance)
        
        // Insert transactions
        for i, txn := range result.Transactions {
            db.CreateBankTransaction(payload.StatementID, &txn)
            
            // Update progress periodically
            if i%10 == 0 {
                progress := 50 + (40 * i / len(result.Transactions))
                db.UpdateJobProgress(job.ID, progress)
            }
        }
        
        db.UpdateJobProgress(job.ID, 95)
        
        // Run auto-matching
        matched := autoMatchTransactions(db, payload.StatementID)
        
        db.UpdateBankStatementStatus(payload.StatementID, "parsed")
        db.UpdateJobProgress(job.ID, 100)
        
        return nil
    }
}
```

### 5. Auto-Matching Logic

```go
// internal/reconciliation/matcher.go

func autoMatchTransactions(db *database.DB, statementID int64) int {
    transactions, _ := db.GetUnmatchedBankTransactions(statementID)
    statement, _ := db.GetBankStatement(statementID)
    
    // Get expenses from the statement month
    startDate := statement.StatementMonth
    endDate := startDate.AddDate(0, 1, -1)
    expenses, _ := db.GetExpenses(startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), "", 0)
    
    matched := 0
    
    for _, txn := range transactions {
        // Skip non-expense transactions
        if txn.Amount >= 0 || txn.TransactionType == "fee" || txn.TransactionType == "transfer" {
            continue
        }
        
        // Try to find matching expense
        for _, exp := range expenses {
            if exp.Status != "paid" {
                continue
            }
            
            // Check number match (highest confidence)
            if txn.CheckNumber != "" && exp.CheckNumber == txn.CheckNumber {
                db.MatchBankTransaction(txn.ID, exp.ID, "auto_exact")
                matched++
                break
            }
            
            // Exact amount + close date match
            if math.Abs(txn.Amount) == exp.Amount {
                txnDate, _ := time.Parse("2006-01-02", txn.PostingDate)
                expDate, _ := time.Parse("2006-01-02", exp.DatePaid)
                daysDiff := math.Abs(txnDate.Sub(expDate).Hours() / 24)
                
                if daysDiff <= 3 {
                    db.MatchBankTransaction(txn.ID, exp.ID, "auto_fuzzy")
                    matched++
                    break
                }
            }
        }
    }
    
    return matched
}
```

### 6. HTTP Handlers

```go
// POST /reconciliation/upload
// Receives PDF, creates file record, creates statement record, queues parse job
func (h *Handlers) ReconciliationUpload(w http.ResponseWriter, r *http.Request) {
    // Parse multipart form
    file, header, _ := r.FormFile("statement")
    defer file.Close()
    
    // Save file using existing file storage
    fileRecord, _ := h.fileStore.Save(file, header.Filename, "statement")
    
    // Parse month from form or filename
    month := r.FormValue("month") // "2025-12"
    statementMonth, _ := time.Parse("2006-01", month)
    
    // Create statement record
    stmtID, _ := h.db.CreateBankStatement(fileRecord.ID, statementMonth)
    
    // Queue parse job
    jobID, _ := h.db.CreateJob("parse_statement", map[string]interface{}{
        "statement_id": stmtID,
        "file_path":    fileRecord.Path,
    })
    
    // Return job ID for polling
    json.NewEncoder(w).Encode(map[string]interface{}{
        "job_id":       jobID,
        "statement_id": stmtID,
    })
}

// GET /jobs/{id}
// Returns job status for polling
func (h *Handlers) JobStatus(w http.ResponseWriter, r *http.Request) {
    id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
    job, _ := h.db.GetJob(id)
    
    json.NewEncoder(w).Encode(map[string]interface{}{
        "id":       job.ID,
        "status":   job.Status,
        "progress": job.Progress,
        "result":   job.Result,
    })
}

// GET /reconciliation/{id}
// Shows reconciliation UI for a statement
func (h *Handlers) ReconciliationView(w http.ResponseWriter, r *http.Request) {
    // ... fetch statement, transactions, potential matches
}

// POST /reconciliation/match
// Manually match a bank transaction to an expense
func (h *Handlers) ReconciliationMatch(w http.ResponseWriter, r *http.Request) {
    txnID, _ := strconv.ParseInt(r.FormValue("transaction_id"), 10, 64)
    expenseID, _ := strconv.ParseInt(r.FormValue("expense_id"), 10, 64)
    
    h.db.MatchBankTransaction(txnID, expenseID, "manual")
    
    http.Redirect(w, r, "/reconciliation/"+r.FormValue("statement_id"), http.StatusSeeOther)
}

// POST /reconciliation/ignore
// Mark a bank transaction as ignored (transfers, etc)
func (h *Handlers) ReconciliationIgnore(w http.ResponseWriter, r *http.Request) {
    txnID, _ := strconv.ParseInt(r.FormValue("transaction_id"), 10, 64)
    h.db.IgnoreBankTransaction(txnID, r.FormValue("reason"))
    // redirect back
}

// POST /reconciliation/create-expense
// Create a new expense from a bank transaction
func (h *Handlers) ReconciliationCreateExpense(w http.ResponseWriter, r *http.Request) {
    txnID, _ := strconv.ParseInt(r.FormValue("transaction_id"), 10, 64)
    txn, _ := h.db.GetBankTransaction(txnID)
    
    // Create expense from transaction data
    vendorID := resolveOrCreateVendor(h.db, txn.VendorHint, r.FormValue("vendor_id"))
    
    expenseID, _ := h.db.CreateExpense(
        txn.PostingDate,
        vendorID,
        math.Abs(txn.Amount),
        "", // invoice
        "paid",
        txn.TransactionType, // payment type
        txn.CheckNumber,
        "", // date opened
        txn.PostingDate, // date paid
        "Created from bank reconciliation",
    )
    
    // Link them
    h.db.MatchBankTransaction(txnID, expenseID, "created")
    
    // redirect back
}
```

### 7. Wiring It Up

**In main.go:**

```go
func main() {
    // ... existing setup ...
    
    // Initialize worker
    worker := jobs.NewWorker(db)
    worker.Register("parse_statement", jobs.ParseStatementHandler(db, parser.NewTDBankParser()))
    worker.Start()
    defer worker.Stop()
    
    // ... routes ...
    mux.HandleFunc("/reconciliation/upload", h.ReconciliationUpload)
    mux.HandleFunc("/jobs", h.JobStatus)
    mux.HandleFunc("/reconciliation/", h.ReconciliationView)
    mux.HandleFunc("/reconciliation/match", h.ReconciliationMatch)
    mux.HandleFunc("/reconciliation/ignore", h.ReconciliationIgnore)
    mux.HandleFunc("/reconciliation/create-expense", h.ReconciliationCreateExpense)
}
```

## Testing the Parser

Before wiring everything up, test the parser standalone:

```go
func main() {
    parser := parser.NewTDBankParser()
    result, err := parser.Parse("/path/to/statement.pdf")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Account: %s\n", result.AccountNumber)
    fmt.Printf("Beginning: $%.2f\n", result.BeginningBalance)
    fmt.Printf("Ending: $%.2f\n", result.EndingBalance)
    fmt.Printf("Transactions: %d\n", len(result.Transactions))
    
    for _, txn := range result.Transactions {
        fmt.Printf("  %s | %-12s | %10.2f | %s\n", 
            txn.PostingDate, txn.TransactionType, txn.Amount, txn.VendorHint)
    }
}
```

## Key Implementation Notes

1. **pdftotext must be installed** - Add `poppler-utils` to Dockerfile

2. **Multi-line descriptions** - Electronic Payments span multiple lines. The amount is on line 1, vendor name on line 2. Accumulate continuation lines until you hit the next date.

3. **Check table is two-column** - The Checks Paid section has two check entries per row. Parse both columns.

4. **Year inference** - Dates are MM/DD format. Use the statement period to determine the year.

5. **Amount parsing** - Remove commas before parsing: `strings.Replace(amountStr, ",", "", -1)`

6. **Negative amounts for debits** - Store debits as negative, credits as positive for easy summing.

7. **Graceful degradation** - If parsing fails for a section, log it and continue. Don't fail the whole job.

## File Structure

```
internal/
├── jobs/
│   ├── worker.go           # Job queue worker
│   ├── job.go              # Job model
│   └── parse_statement.go  # Statement parsing job handler
├── parser/
│   ├── parser.go           # Parser interface
│   └── tdbank.go           # TD Bank specific parser
├── reconciliation/
│   └── matcher.go          # Auto-matching logic
└── database/
    ├── jobs.go             # Job CRUD operations
    ├── statements.go       # Bank statement operations
    └── bank_transactions.go # Bank transaction operations
```