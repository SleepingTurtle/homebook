package database

import (
	"database/sql"
	"fmt"

	"homebooks/internal/models"
)

// CreateBankTransaction inserts a new bank transaction
func (db *DB) CreateBankTransaction(txn *models.BankTransaction) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO bank_transactions (
			reconciliation_id, posting_date, description, amount, transaction_type,
			category, check_number, vendor_hint, reference_number
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, txn.ReconciliationID, txn.PostingDate, txn.Description, txn.Amount, txn.TransactionType,
		txn.Category, txn.CheckNumber, txn.VendorHint, txn.ReferenceNumber)
	if err != nil {
		return 0, fmt.Errorf("insert bank transaction: %w", err)
	}
	return result.LastInsertId()
}

// GetBankTransactions returns all transactions for a reconciliation
func (db *DB) GetBankTransactions(reconciliationID int64) ([]models.BankTransaction, error) {
	rows, err := db.Query(`
		SELECT bt.id, bt.reconciliation_id, date(bt.posting_date), bt.description, bt.amount,
			   bt.transaction_type, bt.category, bt.check_number, bt.vendor_hint, bt.reference_number,
			   bt.matched_expense_id, bt.match_status, bt.match_confidence, bt.matched_at,
			   bt.notes, bt.created_at,
			   COALESCE(v.name, ''), COALESCE(date(e.date), '')
		FROM bank_transactions bt
		LEFT JOIN expenses e ON bt.matched_expense_id = e.id
		LEFT JOIN vendors v ON e.vendor_id = v.id
		WHERE bt.reconciliation_id = ?
		ORDER BY bt.posting_date, bt.id
	`, reconciliationID)
	if err != nil {
		return nil, fmt.Errorf("query bank transactions: %w", err)
	}
	defer rows.Close()

	var transactions []models.BankTransaction
	for rows.Next() {
		var t models.BankTransaction
		var matchedExpenseID sql.NullInt64
		var matchedAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.ReconciliationID, &t.PostingDate, &t.Description, &t.Amount,
			&t.TransactionType, &t.Category, &t.CheckNumber, &t.VendorHint, &t.ReferenceNumber,
			&matchedExpenseID, &t.MatchStatus, &t.MatchConfidence, &matchedAt,
			&t.Notes, &t.CreatedAt,
			&t.MatchedExpenseVendor, &t.MatchedExpenseDate); err != nil {
			return nil, fmt.Errorf("scan bank transaction: %w", err)
		}
		if matchedExpenseID.Valid {
			t.MatchedExpenseID = &matchedExpenseID.Int64
		}
		if matchedAt.Valid {
			t.MatchedAt = &matchedAt.Time
		}
		transactions = append(transactions, t)
	}
	return transactions, rows.Err()
}

// GetBankTransaction returns a single transaction by ID
func (db *DB) GetBankTransaction(id int64) (*models.BankTransaction, error) {
	var t models.BankTransaction
	var matchedExpenseID sql.NullInt64
	var matchedAt sql.NullTime
	err := db.QueryRow(`
		SELECT bt.id, bt.reconciliation_id, date(bt.posting_date), bt.description, bt.amount,
			   bt.transaction_type, bt.category, bt.check_number, bt.vendor_hint, bt.reference_number,
			   bt.matched_expense_id, bt.match_status, bt.match_confidence, bt.matched_at,
			   bt.notes, bt.created_at,
			   COALESCE(v.name, ''), COALESCE(date(e.date), '')
		FROM bank_transactions bt
		LEFT JOIN expenses e ON bt.matched_expense_id = e.id
		LEFT JOIN vendors v ON e.vendor_id = v.id
		WHERE bt.id = ?
	`, id).Scan(&t.ID, &t.ReconciliationID, &t.PostingDate, &t.Description, &t.Amount,
		&t.TransactionType, &t.Category, &t.CheckNumber, &t.VendorHint, &t.ReferenceNumber,
		&matchedExpenseID, &t.MatchStatus, &t.MatchConfidence, &matchedAt,
		&t.Notes, &t.CreatedAt,
		&t.MatchedExpenseVendor, &t.MatchedExpenseDate)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("bank transaction not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query bank transaction: %w", err)
	}
	if matchedExpenseID.Valid {
		t.MatchedExpenseID = &matchedExpenseID.Int64
	}
	if matchedAt.Valid {
		t.MatchedAt = &matchedAt.Time
	}
	return &t, nil
}

// GetUnmatchedBankTransactions returns unmatched transactions for a reconciliation
func (db *DB) GetUnmatchedBankTransactions(reconciliationID int64) ([]models.BankTransaction, error) {
	rows, err := db.Query(`
		SELECT id, reconciliation_id, date(posting_date), description, amount,
			   transaction_type, category, check_number, vendor_hint, reference_number,
			   matched_expense_id, match_status, match_confidence, matched_at,
			   notes, created_at
		FROM bank_transactions
		WHERE reconciliation_id = ? AND match_status = 'unmatched'
		ORDER BY posting_date, id
	`, reconciliationID)
	if err != nil {
		return nil, fmt.Errorf("query unmatched transactions: %w", err)
	}
	defer rows.Close()

	var transactions []models.BankTransaction
	for rows.Next() {
		var t models.BankTransaction
		var matchedExpenseID sql.NullInt64
		var matchedAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.ReconciliationID, &t.PostingDate, &t.Description, &t.Amount,
			&t.TransactionType, &t.Category, &t.CheckNumber, &t.VendorHint, &t.ReferenceNumber,
			&matchedExpenseID, &t.MatchStatus, &t.MatchConfidence, &matchedAt,
			&t.Notes, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan bank transaction: %w", err)
		}
		if matchedExpenseID.Valid {
			t.MatchedExpenseID = &matchedExpenseID.Int64
		}
		if matchedAt.Valid {
			t.MatchedAt = &matchedAt.Time
		}
		transactions = append(transactions, t)
	}
	return transactions, rows.Err()
}

// MatchBankTransaction links a bank transaction to an expense
func (db *DB) MatchBankTransaction(txnID, expenseID int64, confidence string) error {
	_, err := db.Exec(`
		UPDATE bank_transactions
		SET matched_expense_id = ?, match_status = 'matched', match_confidence = ?, matched_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, expenseID, confidence, txnID)
	if err != nil {
		return fmt.Errorf("match bank transaction: %w", err)
	}
	return nil
}

// IgnoreBankTransaction marks a transaction as ignored
func (db *DB) IgnoreBankTransaction(txnID int64, reason string) error {
	_, err := db.Exec(`
		UPDATE bank_transactions
		SET match_status = 'ignored', notes = ?, matched_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, reason, txnID)
	if err != nil {
		return fmt.Errorf("ignore bank transaction: %w", err)
	}
	return nil
}

// MarkBankTransactionCreated marks a transaction as having a created expense
func (db *DB) MarkBankTransactionCreated(txnID, expenseID int64) error {
	_, err := db.Exec(`
		UPDATE bank_transactions
		SET matched_expense_id = ?, match_status = 'created', match_confidence = 'created', matched_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, expenseID, txnID)
	if err != nil {
		return fmt.Errorf("mark bank transaction created: %w", err)
	}
	return nil
}

// UnmatchBankTransaction removes the match from a transaction
func (db *DB) UnmatchBankTransaction(txnID int64) error {
	_, err := db.Exec(`
		UPDATE bank_transactions
		SET matched_expense_id = NULL, match_status = 'unmatched', match_confidence = '', matched_at = NULL
		WHERE id = ?
	`, txnID)
	if err != nil {
		return fmt.Errorf("unmatch bank transaction: %w", err)
	}
	return nil
}

// UpdateBankTransactionType updates the transaction type
func (db *DB) UpdateBankTransactionType(txnID int64, txnType string) error {
	_, err := db.Exec(`UPDATE bank_transactions SET transaction_type = ? WHERE id = ?`, txnType, txnID)
	if err != nil {
		return fmt.Errorf("update bank transaction type: %w", err)
	}
	return nil
}

// UpdateBankTransactionTypeAndSign updates the transaction type and adjusts amount sign
// Also marks deposits as matched since they correspond to sales, not expenses
func (db *DB) UpdateBankTransactionTypeAndSign(txnID int64, txnType string, shouldBePositive bool) error {
	var query string
	if txnType == "deposit" {
		// Deposits are positive and auto-matched (they match to sales, not expenses)
		query = `UPDATE bank_transactions SET transaction_type = ?, amount = ABS(amount), match_status = 'matched', match_confidence = 'deposit', matched_at = CURRENT_TIMESTAMP WHERE id = ?`
	} else if shouldBePositive {
		// Other credits (ach, refund) - make positive, reset match status
		query = `UPDATE bank_transactions SET transaction_type = ?, amount = ABS(amount), match_status = 'unmatched', matched_expense_id = NULL, match_confidence = '', matched_at = NULL WHERE id = ?`
	} else {
		// Debits - make negative, reset match status
		query = `UPDATE bank_transactions SET transaction_type = ?, amount = -ABS(amount), match_status = 'unmatched', matched_expense_id = NULL, match_confidence = '', matched_at = NULL WHERE id = ?`
	}
	_, err := db.Exec(query, txnType, txnID)
	if err != nil {
		return fmt.Errorf("update bank transaction type and sign: %w", err)
	}
	return nil
}

// DeleteBankTransactions deletes all transactions for a reconciliation
func (db *DB) DeleteBankTransactions(reconciliationID int64) error {
	_, err := db.Exec(`DELETE FROM bank_transactions WHERE reconciliation_id = ?`, reconciliationID)
	if err != nil {
		return fmt.Errorf("delete bank transactions: %w", err)
	}
	return nil
}

// GetReconciliationStats returns summary statistics for a reconciliation
type ReconciliationStats struct {
	TotalTransactions  int
	TotalCredits       float64
	TotalDebits        float64
	MatchedCount       int
	UnmatchedCount     int
	IgnoredCount       int
	CreatedCount       int
	ElectronicDeposits float64
	ElectronicPayments float64
	ChecksPaid         float64
	ServiceFees        float64
}

func (db *DB) GetReconciliationStats(reconciliationID int64) (*ReconciliationStats, error) {
	var stats ReconciliationStats
	err := db.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN amount < 0 THEN ABS(amount) ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN match_status = 'matched' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN match_status = 'unmatched' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN match_status = 'ignored' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN match_status = 'created' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN transaction_type = 'deposit' AND amount > 0 THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN transaction_type IN ('ach', 'debit') AND amount < 0 THEN ABS(amount) ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN transaction_type = 'check' AND amount < 0 THEN ABS(amount) ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN transaction_type = 'fee' THEN ABS(amount) ELSE 0 END), 0)
		FROM bank_transactions
		WHERE reconciliation_id = ?
	`, reconciliationID).Scan(&stats.TotalTransactions, &stats.TotalCredits, &stats.TotalDebits,
		&stats.MatchedCount, &stats.UnmatchedCount, &stats.IgnoredCount, &stats.CreatedCount,
		&stats.ElectronicDeposits, &stats.ElectronicPayments, &stats.ChecksPaid, &stats.ServiceFees)
	if err != nil {
		return nil, fmt.Errorf("query reconciliation stats: %w", err)
	}
	return &stats, nil
}
