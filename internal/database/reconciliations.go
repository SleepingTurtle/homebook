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
			   starting_balance, ending_balance, status, file_path, notes, created_at, updated_at
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
		if err := rows.Scan(&r.ID, &r.StatementDate, &r.StatementDateDisplay,
			&r.StartingBalance, &r.EndingBalance, &r.Status, &r.FilePath, &r.Notes,
			&r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan reconciliation: %w", err)
		}
		reconciliations = append(reconciliations, r)
	}
	return reconciliations, rows.Err()
}

// GetReconciliation returns a single reconciliation by ID
func (db *DB) GetReconciliation(id int64) (models.BankReconciliation, error) {
	var r models.BankReconciliation
	err := db.QueryRow(`
		SELECT id, date(statement_date), strftime('%m-%d-%Y', statement_date),
			   starting_balance, ending_balance, status, file_path, notes, created_at, updated_at
		FROM bank_reconciliations
		WHERE id = ?
	`, id).Scan(&r.ID, &r.StatementDate, &r.StatementDateDisplay,
		&r.StartingBalance, &r.EndingBalance, &r.Status, &r.FilePath, &r.Notes,
		&r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return r, fmt.Errorf("reconciliation not found")
	}
	if err != nil {
		return r, fmt.Errorf("query reconciliation: %w", err)
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
		SET statement_date = ?, starting_balance = ?, ending_balance = ?, status = ?, file_path = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, r.StatementDate, r.StartingBalance, r.EndingBalance, r.Status, r.FilePath, r.Notes, r.ID)
	if err != nil {
		return fmt.Errorf("update reconciliation: %w", err)
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
