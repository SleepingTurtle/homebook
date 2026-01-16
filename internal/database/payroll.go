package database

import (
	"database/sql"
	"fmt"

	"homebooks/internal/models"
)

// GetOrCreatePayrollWeek gets an existing week or creates a new one
func (db *DB) GetOrCreatePayrollWeek(weekStart, weekEnd string) (int64, error) {
	// Try to get existing week
	var id int64
	err := db.QueryRow(`SELECT id FROM payroll_weeks WHERE period_start = ? AND period_end = ?`, weekStart, weekEnd).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("query payroll week: %w", err)
	}

	// Create new week
	result, err := db.Exec(`INSERT INTO payroll_weeks (period_start, period_end) VALUES (?, ?)`, weekStart, weekEnd)
	if err != nil {
		return 0, fmt.Errorf("insert payroll week: %w", err)
	}
	return result.LastInsertId()
}

func (db *DB) ListPayroll(filter models.PayrollFilter) ([]models.Payroll, float64, error) {
	query := `
		SELECT p.id, p.week_id, p.employee_id, e.name,
			   strftime('%m-%d-%Y', w.period_start), strftime('%m-%d-%Y', w.period_end),
			   p.total_hours, p.hourly_rate, p.payment_method, p.check_number, p.status,
			   COALESCE(strftime('%m-%d-%Y', p.date_paid), ''), p.notes
		FROM payroll p
		JOIN employees e ON p.employee_id = e.id
		JOIN payroll_weeks w ON p.week_id = w.id
		WHERE 1=1
	`
	var args []interface{}

	if filter.StartDate != "" {
		query += " AND w.period_start >= ?"
		args = append(args, filter.StartDate)
	}
	if filter.EndDate != "" {
		query += " AND w.period_end <= ?"
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

	query += " ORDER BY date(w.period_end) DESC, e.name"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query payroll: %w", err)
	}
	defer rows.Close()

	var payrolls []models.Payroll
	var total float64
	for rows.Next() {
		var p models.Payroll
		if err := rows.Scan(&p.ID, &p.WeekID, &p.EmployeeID, &p.EmployeeName, &p.PeriodStart, &p.PeriodEnd, &p.TotalHours,
			&p.HourlyRate, &p.PaymentMethod, &p.CheckNumber, &p.Status, &p.DatePaid, &p.Notes); err != nil {
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
		SELECT p.id, p.week_id, p.employee_id, e.name,
			   date(w.period_start), date(w.period_end),
			   p.total_hours, p.hourly_rate, p.payment_method, p.check_number, p.status,
			   COALESCE(date(p.date_paid), ''), p.notes
		FROM payroll p
		JOIN employees e ON p.employee_id = e.id
		JOIN payroll_weeks w ON p.week_id = w.id
		WHERE p.id = ?
	`, id).Scan(&p.ID, &p.WeekID, &p.EmployeeID, &p.EmployeeName, &p.PeriodStart, &p.PeriodEnd, &p.TotalHours,
		&p.HourlyRate, &p.PaymentMethod, &p.CheckNumber, &p.Status, &p.DatePaid, &p.Notes)
	if err == sql.ErrNoRows {
		return p, fmt.Errorf("payroll not found")
	}
	if err != nil {
		return p, fmt.Errorf("query payroll: %w", err)
	}
	return p, nil
}

func (db *DB) CreatePayroll(p models.Payroll) (int64, error) {
	var datePaid any
	if p.DatePaid != "" {
		datePaid = p.DatePaid
	}

	result, err := db.Exec(`
		INSERT INTO payroll (week_id, employee_id, total_hours, hourly_rate, payment_method, check_number, status, date_paid, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.WeekID, p.EmployeeID, p.TotalHours, p.HourlyRate, p.PaymentMethod, p.CheckNumber, p.Status, datePaid, p.Notes)
	if err != nil {
		return 0, fmt.Errorf("insert payroll: %w", err)
	}
	return result.LastInsertId()
}

func (db *DB) UpdatePayroll(p models.Payroll) error {
	var datePaid any
	if p.DatePaid != "" {
		datePaid = p.DatePaid
	}

	_, err := db.Exec(`
		UPDATE payroll
		SET week_id = ?, employee_id = ?, total_hours = ?, hourly_rate = ?,
			payment_method = ?, check_number = ?, status = ?, date_paid = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, p.WeekID, p.EmployeeID, p.TotalHours, p.HourlyRate, p.PaymentMethod, p.CheckNumber, p.Status, datePaid, p.Notes, p.ID)
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

// GetWeeklyPayroll returns payroll entries for all active employees for a given week
// Returns a map of employee_id -> Payroll (nil if no entry exists for that employee)
func (db *DB) GetWeeklyPayroll(weekStart, weekEnd string) ([]models.WeeklyPayrollEntry, float64, error) {
	// Get all active employees
	employees, err := db.ListEmployees(true)
	if err != nil {
		return nil, 0, fmt.Errorf("list employees: %w", err)
	}

	// Get existing payroll entries for this week via payroll_weeks join
	rows, err := db.Query(`
		SELECT p.id, p.week_id, p.employee_id, p.total_hours, p.hourly_rate, p.payment_method,
			   p.check_number, p.status, COALESCE(strftime('%m-%d-%Y', p.date_paid), ''), p.notes
		FROM payroll p
		JOIN payroll_weeks w ON p.week_id = w.id
		WHERE w.period_start = ? AND w.period_end = ?
	`, weekStart, weekEnd)
	if err != nil {
		return nil, 0, fmt.Errorf("query weekly payroll: %w", err)
	}
	defer rows.Close()

	// Build map of employee_id -> payroll
	payrollMap := make(map[int64]*models.Payroll)
	for rows.Next() {
		var p models.Payroll
		if err := rows.Scan(&p.ID, &p.WeekID, &p.EmployeeID, &p.TotalHours, &p.HourlyRate, &p.PaymentMethod,
			&p.CheckNumber, &p.Status, &p.DatePaid, &p.Notes); err != nil {
			return nil, 0, fmt.Errorf("scan payroll: %w", err)
		}
		payrollMap[p.EmployeeID] = &p
	}

	// Build result with all employees
	var entries []models.WeeklyPayrollEntry
	var total float64
	for _, emp := range employees {
		entry := models.WeeklyPayrollEntry{
			Employee: emp,
		}
		if p, exists := payrollMap[emp.ID]; exists {
			entry.Payroll = p
			total += p.TotalPay()
		}
		entries = append(entries, entry)
	}

	return entries, total, nil
}

// GetPayrollWeek returns a payroll week by ID
func (db *DB) GetPayrollWeek(id int64) (models.PayrollWeek, error) {
	var w models.PayrollWeek
	err := db.QueryRow(`SELECT id, date(period_start), date(period_end) FROM payroll_weeks WHERE id = ?`, id).Scan(&w.ID, &w.PeriodStart, &w.PeriodEnd)
	if err == sql.ErrNoRows {
		return w, fmt.Errorf("payroll week not found")
	}
	if err != nil {
		return w, fmt.Errorf("query payroll week: %w", err)
	}
	return w, nil
}

// GetWeeklyPayrollByWeekID returns payroll entries for a given week ID
func (db *DB) GetWeeklyPayrollByWeekID(weekID int64) ([]models.WeeklyPayrollEntry, float64, error) {
	// Get all employees (active or with payroll for this week)
	rows, err := db.Query(`
		SELECT DISTINCT e.id, e.name, e.hourly_rate, e.payment_method, e.active
		FROM employees e
		LEFT JOIN payroll p ON p.employee_id = e.id AND p.week_id = ?
		WHERE e.active = 1 OR p.id IS NOT NULL
		ORDER BY e.name
	`, weekID)
	if err != nil {
		return nil, 0, fmt.Errorf("list employees: %w", err)
	}
	defer rows.Close()

	var employees []models.Employee
	for rows.Next() {
		var e models.Employee
		if err := rows.Scan(&e.ID, &e.Name, &e.HourlyRate, &e.PaymentMethod, &e.Active); err != nil {
			return nil, 0, fmt.Errorf("scan employee: %w", err)
		}
		employees = append(employees, e)
	}

	// Get existing payroll entries for this week
	payrollRows, err := db.Query(`
		SELECT p.id, p.week_id, p.employee_id, p.total_hours, p.hourly_rate, p.payment_method,
			   p.check_number, p.status, COALESCE(strftime('%m-%d-%Y', p.date_paid), ''), p.notes
		FROM payroll p
		WHERE p.week_id = ?
	`, weekID)
	if err != nil {
		return nil, 0, fmt.Errorf("query weekly payroll: %w", err)
	}
	defer payrollRows.Close()

	// Build map of employee_id -> payroll
	payrollMap := make(map[int64]*models.Payroll)
	for payrollRows.Next() {
		var p models.Payroll
		if err := payrollRows.Scan(&p.ID, &p.WeekID, &p.EmployeeID, &p.TotalHours, &p.HourlyRate, &p.PaymentMethod,
			&p.CheckNumber, &p.Status, &p.DatePaid, &p.Notes); err != nil {
			return nil, 0, fmt.Errorf("scan payroll: %w", err)
		}
		payrollMap[p.EmployeeID] = &p
	}

	// Build result with all employees
	var entries []models.WeeklyPayrollEntry
	var total float64
	for _, emp := range employees {
		entry := models.WeeklyPayrollEntry{
			Employee: emp,
		}
		if p, exists := payrollMap[emp.ID]; exists {
			entry.Payroll = p
			total += p.TotalPay()
		}
		entries = append(entries, entry)
	}

	return entries, total, nil
}

// UpsertWeeklyPayroll creates or updates a payroll entry for an employee for a specific week
func (db *DB) UpsertWeeklyPayroll(employeeID int64, weekStart, weekEnd string, hours float64, hourlyRate float64, paymentMethod string) error {
	// Get or create the payroll week
	weekID, err := db.GetOrCreatePayrollWeek(weekStart, weekEnd)
	if err != nil {
		return fmt.Errorf("get or create payroll week: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO payroll (week_id, employee_id, total_hours, hourly_rate, payment_method, status)
		VALUES (?, ?, ?, ?, ?, 'not_paid')
		ON CONFLICT(week_id, employee_id) DO UPDATE SET
			total_hours = excluded.total_hours,
			hourly_rate = excluded.hourly_rate,
			payment_method = excluded.payment_method,
			updated_at = CURRENT_TIMESTAMP
		WHERE status = 'not_paid'
	`, weekID, employeeID, hours, hourlyRate, paymentMethod)
	if err != nil {
		return fmt.Errorf("upsert weekly payroll: %w", err)
	}
	return nil
}

// MarkPayrollPaidWithDetails marks a payroll entry as paid with payment details
func (db *DB) MarkPayrollPaidWithDetails(id int64, paymentMethod, checkNumber string) error {
	_, err := db.Exec(`
		UPDATE payroll
		SET status = 'paid', payment_method = ?, check_number = ?, date_paid = date('now'), updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, paymentMethod, checkNumber, id)
	if err != nil {
		return fmt.Errorf("mark payroll paid: %w", err)
	}
	return nil
}

// ListPayrollWeeks returns a summary of past payroll weeks
func (db *DB) ListPayrollWeeks(limit int) ([]models.PayrollWeekSummary, error) {
	rows, err := db.Query(`
		SELECT
			w.id,
			w.period_start,
			w.period_end,
			strftime('%m-%d-%Y', w.period_start) as start_display,
			strftime('%m-%d-%Y', w.period_end) as end_display,
			COUNT(*) as employee_count,
			SUM(p.total_hours) as total_hours,
			SUM(p.total_hours * p.hourly_rate) as total_pay,
			SUM(CASE WHEN p.status = 'paid' THEN 1 ELSE 0 END) as paid_count
		FROM payroll_weeks w
		JOIN payroll p ON p.week_id = w.id
		GROUP BY w.id
		ORDER BY w.period_start DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query payroll weeks: %w", err)
	}
	defer rows.Close()

	var weeks []models.PayrollWeekSummary
	for rows.Next() {
		var w models.PayrollWeekSummary
		if err := rows.Scan(&w.WeekID, &w.PeriodStart, &w.PeriodEnd, &w.PeriodStartDisplay, &w.PeriodEndDisplay,
			&w.EmployeeCount, &w.TotalHours, &w.TotalPay, &w.PaidCount); err != nil {
			return nil, fmt.Errorf("scan payroll week: %w", err)
		}
		weeks = append(weeks, w)
	}
	return weeks, rows.Err()
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
