package database

import (
	"database/sql"
	"fmt"

	"homebooks/internal/models"
)

func (db *DB) ListVendors() ([]models.Vendor, error) {
	rows, err := db.Query(`
		SELECT id, name, category, description, created_at
		FROM vendors
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("query vendors: %w", err)
	}
	defer rows.Close()

	var vendors []models.Vendor
	for rows.Next() {
		var v models.Vendor
		if err := rows.Scan(&v.ID, &v.Name, &v.Category, &v.Description, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan vendor: %w", err)
		}
		vendors = append(vendors, v)
	}
	return vendors, rows.Err()
}

func (db *DB) GetVendor(id int64) (models.Vendor, error) {
	var v models.Vendor
	err := db.QueryRow(`
		SELECT id, name, category, description, created_at
		FROM vendors
		WHERE id = ?
	`, id).Scan(&v.ID, &v.Name, &v.Category, &v.Description, &v.CreatedAt)
	if err == sql.ErrNoRows {
		return v, fmt.Errorf("vendor not found")
	}
	if err != nil {
		return v, fmt.Errorf("query vendor: %w", err)
	}
	return v, nil
}

func (db *DB) CreateVendor(name, category, description string) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO vendors (name, category, description) VALUES (?, ?, ?)
	`, name, category, description)
	if err != nil {
		return 0, fmt.Errorf("insert vendor: %w", err)
	}
	return result.LastInsertId()
}

func (db *DB) UpdateVendor(id int64, name, category, description string) error {
	_, err := db.Exec(`
		UPDATE vendors SET name = ?, category = ?, description = ? WHERE id = ?
	`, name, category, description, id)
	if err != nil {
		return fmt.Errorf("update vendor: %w", err)
	}
	return nil
}

func (db *DB) DeleteVendor(id int64) error {
	_, err := db.Exec(`DELETE FROM vendors WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete vendor: %w", err)
	}
	return nil
}
