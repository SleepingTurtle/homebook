package database

import (
	"database/sql"
	"fmt"

	"homebooks/internal/models"
)

func (db *DB) ListEmployees(activeOnly bool) ([]models.Employee, error) {
	query := `
		SELECT id, name, hourly_rate, payment_method, active
		FROM employees
	`
	if activeOnly {
		query += " WHERE active = 1"
	}
	query += " ORDER BY name"

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query employees: %w", err)
	}
	defer rows.Close()

	var employees []models.Employee
	for rows.Next() {
		var e models.Employee
		var active int
		if err := rows.Scan(&e.ID, &e.Name, &e.HourlyRate, &e.PaymentMethod, &active); err != nil {
			return nil, fmt.Errorf("scan employee: %w", err)
		}
		e.Active = active == 1
		employees = append(employees, e)
	}
	return employees, rows.Err()
}

func (db *DB) GetEmployee(id int64) (models.Employee, error) {
	var e models.Employee
	var active int
	err := db.QueryRow(`
		SELECT id, name, hourly_rate, payment_method, active
		FROM employees
		WHERE id = ?
	`, id).Scan(&e.ID, &e.Name, &e.HourlyRate, &e.PaymentMethod, &active)
	if err == sql.ErrNoRows {
		return e, fmt.Errorf("employee not found")
	}
	if err != nil {
		return e, fmt.Errorf("query employee: %w", err)
	}
	e.Active = active == 1
	return e, nil
}

func (db *DB) CreateEmployee(name string, hourlyRate float64, paymentMethod string) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO employees (name, hourly_rate, payment_method) VALUES (?, ?, ?)
	`, name, hourlyRate, paymentMethod)
	if err != nil {
		return 0, fmt.Errorf("insert employee: %w", err)
	}
	return result.LastInsertId()
}

func (db *DB) UpdateEmployee(id int64, name string, hourlyRate float64, paymentMethod string) error {
	_, err := db.Exec(`
		UPDATE employees SET name = ?, hourly_rate = ?, payment_method = ? WHERE id = ?
	`, name, hourlyRate, paymentMethod, id)
	if err != nil {
		return fmt.Errorf("update employee: %w", err)
	}
	return nil
}

func (db *DB) DeactivateEmployee(id int64) error {
	_, err := db.Exec(`UPDATE employees SET active = 0 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deactivate employee: %w", err)
	}
	return nil
}

func (db *DB) ReactivateEmployee(id int64) error {
	_, err := db.Exec(`UPDATE employees SET active = 1 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("reactivate employee: %w", err)
	}
	return nil
}
