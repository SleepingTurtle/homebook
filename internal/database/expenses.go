package database

import (
	"database/sql"
	"fmt"

	"homebooks/internal/models"
)

func (db *DB) ListExpenses(filter models.ExpenseFilter) ([]models.Expense, float64, error) {
	query := `
		SELECT e.id, strftime('%m-%d-%Y', e.date), e.vendor_id, v.name, e.amount, e.invoice_number, e.status,
			   e.payment_type, e.check_number, COALESCE(strftime('%m-%d-%Y', e.date_opened), ''), COALESCE(strftime('%m-%d-%Y', e.date_paid), ''),
			   e.notes, e.created_at, e.updated_at
		FROM expenses e
		JOIN vendors v ON e.vendor_id = v.id
		WHERE 1=1
	`
	var args []interface{}

	if filter.StartDate != "" {
		query += " AND e.date >= ?"
		args = append(args, filter.StartDate)
	}
	if filter.EndDate != "" {
		query += " AND e.date <= ?"
		args = append(args, filter.EndDate)
	}
	if filter.Status != "" {
		query += " AND e.status = ?"
		args = append(args, filter.Status)
	}
	if filter.VendorID > 0 {
		query += " AND e.vendor_id = ?"
		args = append(args, filter.VendorID)
	}

	query += " ORDER BY date(e.date) DESC, e.id DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query expenses: %w", err)
	}
	defer rows.Close()

	var expenses []models.Expense
	var total float64
	for rows.Next() {
		var e models.Expense
		if err := rows.Scan(&e.ID, &e.Date, &e.VendorID, &e.VendorName, &e.Amount, &e.InvoiceNumber, &e.Status,
			&e.PaymentType, &e.CheckNumber, &e.DateOpened, &e.DatePaid, &e.Notes, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan expense: %w", err)
		}
		expenses = append(expenses, e)
		total += e.Amount
	}
	return expenses, total, rows.Err()
}

func (db *DB) ListUnpaidExpenses() ([]models.Expense, float64, error) {
	return db.ListExpenses(models.ExpenseFilter{Status: "not_paid"})
}

func (db *DB) GetExpense(id int64) (models.Expense, error) {
	var e models.Expense
	err := db.QueryRow(`
		SELECT e.id, date(e.date), e.vendor_id, v.name, e.amount, e.invoice_number, e.status,
			   e.payment_type, e.check_number, COALESCE(date(e.date_opened), ''), COALESCE(date(e.date_paid), ''),
			   e.notes, e.created_at, e.updated_at
		FROM expenses e
		JOIN vendors v ON e.vendor_id = v.id
		WHERE e.id = ?
	`, id).Scan(&e.ID, &e.Date, &e.VendorID, &e.VendorName, &e.Amount, &e.InvoiceNumber, &e.Status,
		&e.PaymentType, &e.CheckNumber, &e.DateOpened, &e.DatePaid, &e.Notes, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return e, fmt.Errorf("expense not found")
	}
	if err != nil {
		return e, fmt.Errorf("query expense: %w", err)
	}
	return e, nil
}

func (db *DB) CreateExpense(e models.Expense) (int64, error) {
	var dateOpened, datePaid interface{}
	if e.DateOpened != "" {
		dateOpened = e.DateOpened
	}
	if e.DatePaid != "" {
		datePaid = e.DatePaid
	}

	result, err := db.Exec(`
		INSERT INTO expenses (date, vendor_id, amount, invoice_number, status, payment_type, check_number, date_opened, date_paid, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, e.Date, e.VendorID, e.Amount, e.InvoiceNumber, e.Status, e.PaymentType, e.CheckNumber, dateOpened, datePaid, e.Notes)
	if err != nil {
		return 0, fmt.Errorf("insert expense: %w", err)
	}
	return result.LastInsertId()
}

func (db *DB) UpdateExpense(e models.Expense) error {
	var dateOpened, datePaid interface{}
	if e.DateOpened != "" {
		dateOpened = e.DateOpened
	}
	if e.DatePaid != "" {
		datePaid = e.DatePaid
	}

	_, err := db.Exec(`
		UPDATE expenses
		SET date = ?, vendor_id = ?, amount = ?, invoice_number = ?, status = ?, payment_type = ?,
			check_number = ?, date_opened = ?, date_paid = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, e.Date, e.VendorID, e.Amount, e.InvoiceNumber, e.Status, e.PaymentType, e.CheckNumber, dateOpened, datePaid, e.Notes, e.ID)
	if err != nil {
		return fmt.Errorf("update expense: %w", err)
	}
	return nil
}

func (db *DB) MarkExpensePaid(id int64) error {
	_, err := db.Exec(`
		UPDATE expenses
		SET status = 'paid', date_paid = date('now'), updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("mark expense paid: %w", err)
	}
	return nil
}

func (db *DB) DeleteExpense(id int64) error {
	_, err := db.Exec(`DELETE FROM expenses WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	return nil
}

// GetLastCheckNumber returns the most recent check number used for expenses
func (db *DB) GetLastExpenseCheckNumber() (string, error) {
	var checkNum sql.NullString
	err := db.QueryRow(`
		SELECT check_number FROM expenses
		WHERE check_number != '' AND payment_type = 'check'
		ORDER BY id DESC LIMIT 1
	`).Scan(&checkNum)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("query last check number: %w", err)
	}
	return checkNum.String, nil
}
