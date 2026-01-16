package database

import (
	"database/sql"
	"fmt"

	"homebooks/internal/models"
)

func (db *DB) ListPayroll(filter models.PayrollFilter) ([]models.Payroll, float64, error) {
	query := `
		SELECT p.id, p.employee_id, e.name, strftime('%m-%d-%Y', p.period_start), strftime('%m-%d-%Y', p.period_end), p.total_hours,
			   p.hourly_rate, p.payment_method, p.check_number, p.status,
			   COALESCE(strftime('%m-%d-%Y', p.date_paid), ''), p.notes, p.created_at, p.updated_at
		FROM payroll p
		JOIN employees e ON p.employee_id = e.id
		WHERE 1=1
	`
	var args []interface{}

	if filter.StartDate != "" {
		query += " AND p.period_start >= ?"
		args = append(args, filter.StartDate)
	}
	if filter.EndDate != "" {
		query += " AND p.period_end <= ?"
		args = append(args, filter.EndDate)
	}
	if filter.Status != "" {
		query += " AND p.status = ?"
		args = append(args, filter.Status)
	}
	if filter.EmployeeID > 0 {
		query += " AND p.employee_id = ?"
		args = append(args, filter.EmployeeID)
	}

	query += " ORDER BY date(p.period_end) DESC, e.name"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query payroll: %w", err)
	}
	defer rows.Close()

	var payrolls []models.Payroll
	var total float64
	for rows.Next() {
		var p models.Payroll
		if err := rows.Scan(&p.ID, &p.EmployeeID, &p.EmployeeName, &p.PeriodStart, &p.PeriodEnd, &p.TotalHours,
			&p.HourlyRate, &p.PaymentMethod, &p.CheckNumber, &p.Status, &p.DatePaid, &p.Notes, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan payroll: %w", err)
		}
		payrolls = append(payrolls, p)
		total += p.TotalPay()
	}
	return payrolls, total, rows.Err()
}

func (db *DB) ListUnpaidPayroll() ([]models.Payroll, float64, error) {
	return db.ListPayroll(models.PayrollFilter{Status: "not_paid"})
}

func (db *DB) GetPayroll(id int64) (models.Payroll, error) {
	var p models.Payroll
	err := db.QueryRow(`
		SELECT p.id, p.employee_id, e.name, date(p.period_start), date(p.period_end), p.total_hours,
			   p.hourly_rate, p.payment_method, p.check_number, p.status,
			   COALESCE(date(p.date_paid), ''), p.notes, p.created_at, p.updated_at
		FROM payroll p
		JOIN employees e ON p.employee_id = e.id
		WHERE p.id = ?
	`, id).Scan(&p.ID, &p.EmployeeID, &p.EmployeeName, &p.PeriodStart, &p.PeriodEnd, &p.TotalHours,
		&p.HourlyRate, &p.PaymentMethod, &p.CheckNumber, &p.Status, &p.DatePaid, &p.Notes, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return p, fmt.Errorf("payroll not found")
	}
	if err != nil {
		return p, fmt.Errorf("query payroll: %w", err)
	}
	return p, nil
}

func (db *DB) CreatePayroll(p models.Payroll) (int64, error) {
	var datePaid interface{}
	if p.DatePaid != "" {
		datePaid = p.DatePaid
	}

	result, err := db.Exec(`
		INSERT INTO payroll (employee_id, period_start, period_end, total_hours, hourly_rate, payment_method, check_number, status, date_paid, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.EmployeeID, p.PeriodStart, p.PeriodEnd, p.TotalHours, p.HourlyRate, p.PaymentMethod, p.CheckNumber, p.Status, datePaid, p.Notes)
	if err != nil {
		return 0, fmt.Errorf("insert payroll: %w", err)
	}
	return result.LastInsertId()
}

func (db *DB) UpdatePayroll(p models.Payroll) error {
	var datePaid interface{}
	if p.DatePaid != "" {
		datePaid = p.DatePaid
	}

	_, err := db.Exec(`
		UPDATE payroll
		SET employee_id = ?, period_start = ?, period_end = ?, total_hours = ?, hourly_rate = ?,
			payment_method = ?, check_number = ?, status = ?, date_paid = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, p.EmployeeID, p.PeriodStart, p.PeriodEnd, p.TotalHours, p.HourlyRate, p.PaymentMethod, p.CheckNumber, p.Status, datePaid, p.Notes, p.ID)
	if err != nil {
		return fmt.Errorf("update payroll: %w", err)
	}
	return nil
}

func (db *DB) MarkPayrollPaid(id int64) error {
	_, err := db.Exec(`
		UPDATE payroll
		SET status = 'paid', date_paid = date('now'), updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("mark payroll paid: %w", err)
	}
	return nil
}

func (db *DB) DeletePayroll(id int64) error {
	_, err := db.Exec(`DELETE FROM payroll WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete payroll: %w", err)
	}
	return nil
}

// GetLastPayrollCheckNumber returns the most recent check number used for payroll
func (db *DB) GetLastPayrollCheckNumber() (string, error) {
	var checkNum sql.NullString
	err := db.QueryRow(`
		SELECT check_number FROM payroll
		WHERE check_number != '' AND payment_method = 'check'
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
