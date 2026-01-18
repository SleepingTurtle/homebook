package database

import (
	"database/sql"
	"fmt"
	"strings"

	"homebooks/internal/models"
)

// GetDeliverySalesForDate retrieves delivery sales for a specific date
func (db *DB) GetDeliverySalesForDate(date string) (*models.DeliverySales, error) {
	var d models.DeliverySales
	err := db.QueryRow(`
		SELECT id, date(date), grubhub_subtotal, grubhub_net, doordash_subtotal, doordash_net,
		       ubereats_earnings, ubereats_payout, notes
		FROM delivery_sales
		WHERE date = ?
	`, date).Scan(&d.ID, &d.Date, &d.GrubhubSubtotal, &d.GrubhubNet,
		&d.DoordashSubtotal, &d.DoordashNet, &d.UberEatsEarnings, &d.UberEatsPayout,
		&d.Notes)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query delivery sales: %w", err)
	}
	return &d, nil
}

// UpsertDeliverySales creates or updates delivery sales for a date
func (db *DB) UpsertDeliverySales(d models.DeliverySales) error {
	_, err := db.Exec(`
		INSERT INTO delivery_sales (date, grubhub_subtotal, grubhub_net, doordash_subtotal, doordash_net,
		                            ubereats_earnings, ubereats_payout, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			grubhub_subtotal = excluded.grubhub_subtotal,
			grubhub_net = excluded.grubhub_net,
			doordash_subtotal = excluded.doordash_subtotal,
			doordash_net = excluded.doordash_net,
			ubereats_earnings = excluded.ubereats_earnings,
			ubereats_payout = excluded.ubereats_payout,
			notes = excluded.notes,
			updated_at = CURRENT_TIMESTAMP
	`, d.Date, d.GrubhubSubtotal, d.GrubhubNet, d.DoordashSubtotal, d.DoordashNet,
		d.UberEatsEarnings, d.UberEatsPayout, d.Notes)
	if err != nil {
		return fmt.Errorf("upsert delivery sales: %w", err)
	}
	return nil
}

// GetDeliverySalesForDates retrieves delivery sales for multiple dates (bulk fetch)
func (db *DB) GetDeliverySalesForDates(dates []string) (map[string]*models.DeliverySales, error) {
	if len(dates) == 0 {
		return make(map[string]*models.DeliverySales), nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(dates))
	args := make([]interface{}, len(dates))
	for i, date := range dates {
		placeholders[i] = "?"
		args[i] = date
	}

	query := fmt.Sprintf(`
		SELECT id, date(date), grubhub_subtotal, grubhub_net, doordash_subtotal, doordash_net,
		       ubereats_earnings, ubereats_payout, notes
		FROM delivery_sales
		WHERE date IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query delivery sales for dates: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*models.DeliverySales)
	for rows.Next() {
		var d models.DeliverySales
		if err := rows.Scan(&d.ID, &d.Date, &d.GrubhubSubtotal, &d.GrubhubNet,
			&d.DoordashSubtotal, &d.DoordashNet, &d.UberEatsEarnings, &d.UberEatsPayout,
			&d.Notes); err != nil {
			return nil, fmt.Errorf("scan delivery sale: %w", err)
		}
		result[d.Date] = &d
	}

	return result, rows.Err()
}
