package database

import (
	"database/sql"
	"fmt"

	"homebooks/internal/models"
)

// ListReconciliations returns all bank reconciliations ordered by date descending
func (db *DB) ListReconciliations() ([]models.BankReconciliation, error) {
	rows, err := db.Query(`
		SELECT id, date(statement_date), strftime('%m-%d-%Y', statement_date),
			   starting_balance, ending_balance, status, file_path,
			   account_last_four, parse_job_id, parsed_at, reconciled_at,
			   notes, electronic_deposits, electronic_payments, checks_paid, service_fees,
			   created_at, updated_at
		FROM bank_reconciliations
		ORDER BY statement_date DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query reconciliations: %w", err)
	}
	defer rows.Close()

	var reconciliations []models.BankReconciliation
	for rows.Next() {
		var r models.BankReconciliation
		var parseJobID sql.NullInt64
		var parsedAt, reconciledAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.StatementDate, &r.StatementDateDisplay,
			&r.StartingBalance, &r.EndingBalance, &r.Status, &r.FilePath,
			&r.AccountLastFour, &parseJobID, &parsedAt, &reconciledAt,
			&r.Notes, &r.ElectronicDeposits, &r.ElectronicPayments, &r.ChecksPaid, &r.ServiceFees,
			&r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan reconciliation: %w", err)
		}
		if parseJobID.Valid {
			r.ParseJobID = &parseJobID.Int64
		}
		if parsedAt.Valid {
			r.ParsedAt = &parsedAt.Time
		}
		if reconciledAt.Valid {
			r.ReconciledAt = &reconciledAt.Time
		}
		reconciliations = append(reconciliations, r)
	}
	return reconciliations, rows.Err()
}

// GetReconciliation returns a single reconciliation by ID
func (db *DB) GetReconciliation(id int64) (models.BankReconciliation, error) {
	var r models.BankReconciliation
	var parseJobID sql.NullInt64
	var parsedAt, reconciledAt sql.NullTime
	err := db.QueryRow(`
		SELECT id, date(statement_date), strftime('%m-%d-%Y', statement_date),
			   starting_balance, ending_balance, status, file_path,
			   account_last_four, parse_job_id, parsed_at, reconciled_at,
			   notes, electronic_deposits, electronic_payments, checks_paid, service_fees,
			   created_at, updated_at
		FROM bank_reconciliations
		WHERE id = ?
	`, id).Scan(&r.ID, &r.StatementDate, &r.StatementDateDisplay,
		&r.StartingBalance, &r.EndingBalance, &r.Status, &r.FilePath,
		&r.AccountLastFour, &parseJobID, &parsedAt, &reconciledAt,
		&r.Notes, &r.ElectronicDeposits, &r.ElectronicPayments, &r.ChecksPaid, &r.ServiceFees,
		&r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return r, fmt.Errorf("reconciliation not found")
	}
	if err != nil {
		return r, fmt.Errorf("query reconciliation: %w", err)
	}
	if parseJobID.Valid {
		r.ParseJobID = &parseJobID.Int64
	}
	if parsedAt.Valid {
		r.ParsedAt = &parsedAt.Time
	}
	if reconciledAt.Valid {
		r.ReconciledAt = &reconciledAt.Time
	}
	return r, nil
}

// CreateReconciliation creates a new bank reconciliation
func (db *DB) CreateReconciliation(r models.BankReconciliation) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO bank_reconciliations (statement_date, starting_balance, ending_balance, status, file_path, notes)
		VALUES (?, ?, ?, ?, ?, ?)
	`, r.StatementDate, r.StartingBalance, r.EndingBalance, r.Status, r.FilePath, r.Notes)
	if err != nil {
		return 0, fmt.Errorf("insert reconciliation: %w", err)
	}
	return result.LastInsertId()
}

// UpdateReconciliation updates an existing reconciliation
func (db *DB) UpdateReconciliation(r models.BankReconciliation) error {
	_, err := db.Exec(`
		UPDATE bank_reconciliations
		SET statement_date = ?, starting_balance = ?, ending_balance = ?, status = ?,
		    file_path = ?, account_last_four = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, r.StatementDate, r.StartingBalance, r.EndingBalance, r.Status,
		r.FilePath, r.AccountLastFour, r.Notes, r.ID)
	if err != nil {
		return fmt.Errorf("update reconciliation: %w", err)
	}
	return nil
}

// UpdateReconciliationStatus updates just the status of a reconciliation
func (db *DB) UpdateReconciliationStatus(id int64, status string) error {
	_, err := db.Exec(`
		UPDATE bank_reconciliations
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, id)
	if err != nil {
		return fmt.Errorf("update reconciliation status: %w", err)
	}
	return nil
}

// UpdateReconciliationParseJob sets the parse job ID for a reconciliation
func (db *DB) UpdateReconciliationParseJob(id int64, jobID int64) error {
	_, err := db.Exec(`
		UPDATE bank_reconciliations
		SET parse_job_id = ?, status = 'parsing', updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, jobID, id)
	if err != nil {
		return fmt.Errorf("update reconciliation parse job: %w", err)
	}
	return nil
}

// UpdateReconciliationParsed updates a reconciliation after parsing completes
func (db *DB) UpdateReconciliationParsed(id int64, startingBalance, endingBalance float64, accountLastFour string,
	electronicDeposits, electronicPayments, checksPaid, serviceFees float64) error {
	_, err := db.Exec(`
		UPDATE bank_reconciliations
		SET starting_balance = ?, ending_balance = ?, account_last_four = ?,
		    electronic_deposits = ?, electronic_payments = ?, checks_paid = ?, service_fees = ?,
		    status = 'parsed', parsed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, startingBalance, endingBalance, accountLastFour,
		electronicDeposits, electronicPayments, checksPaid, serviceFees, id)
	if err != nil {
		return fmt.Errorf("update reconciliation parsed: %w", err)
	}
	return nil
}

// UpdateReconciliationCompleted marks a reconciliation as completed
func (db *DB) UpdateReconciliationCompleted(id int64) error {
	_, err := db.Exec(`
		UPDATE bank_reconciliations
		SET status = 'completed', reconciled_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("update reconciliation completed: %w", err)
	}
	return nil
}

// DeleteReconciliation deletes a reconciliation by ID
func (db *DB) DeleteReconciliation(id int64) error {
	_, err := db.Exec(`DELETE FROM bank_reconciliations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete reconciliation: %w", err)
	}
	return nil
}

// GetReconciledMonths returns a set of months (YYYY-MM format) that have reconciliations
func (db *DB) GetReconciledMonths() (map[string]bool, error) {
	rows, err := db.Query(`
		SELECT DISTINCT strftime('%Y-%m', statement_date)
		FROM bank_reconciliations
	`)
	if err != nil {
		return nil, fmt.Errorf("query reconciled months: %w", err)
	}
	defer rows.Close()

	months := make(map[string]bool)
	for rows.Next() {
		var month string
		if err := rows.Scan(&month); err != nil {
			return nil, fmt.Errorf("scan month: %w", err)
		}
		months[month] = true
	}
	return months, rows.Err()
}
