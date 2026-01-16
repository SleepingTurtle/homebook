package database

import (
	"database/sql"
	"fmt"
	"time"

	"homebooks/internal/models"
)

func (db *DB) ListSales(filter models.SalesFilter) ([]models.DailySale, error) {
	query := `
		SELECT id, strftime('%m-%d-%Y', date), shift, net_sales, taxes, credit_card, cash_receipt, cash_on_hand, notes, created_at, updated_at
		FROM daily_sales
		WHERE 1=1
	`
	var args []interface{}

	if filter.StartDate != "" {
		query += " AND date >= ?"
		args = append(args, filter.StartDate)
	}
	if filter.EndDate != "" {
		query += " AND date <= ?"
		args = append(args, filter.EndDate)
	}

	query += " ORDER BY date(date) DESC, CASE shift WHEN 'dinner' THEN 1 WHEN 'lunch' THEN 2 WHEN 'breakfast' THEN 3 END"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query sales: %w", err)
	}
	defer rows.Close()

	var sales []models.DailySale
	for rows.Next() {
		var s models.DailySale
		if err := rows.Scan(&s.ID, &s.Date, &s.Shift, &s.NetSales, &s.Taxes, &s.CreditCard, &s.CashReceipt, &s.CashOnHand, &s.Notes, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan sale: %w", err)
		}
		sales = append(sales, s)
	}
	return sales, rows.Err()
}

func (db *DB) ListRecentSales(days int) ([]models.DailySale, error) {
	rows, err := db.Query(`
		SELECT id, strftime('%m-%d-%Y', date), shift, net_sales, taxes, credit_card, cash_receipt, cash_on_hand, notes, created_at, updated_at
		FROM daily_sales
		WHERE date >= date('now', '-' || ? || ' days')
		ORDER BY date(date) DESC, CASE shift WHEN 'dinner' THEN 1 WHEN 'lunch' THEN 2 WHEN 'breakfast' THEN 3 END
	`, days)
	if err != nil {
		return nil, fmt.Errorf("query recent sales: %w", err)
	}
	defer rows.Close()

	var sales []models.DailySale
	for rows.Next() {
		var s models.DailySale
		if err := rows.Scan(&s.ID, &s.Date, &s.Shift, &s.NetSales, &s.Taxes, &s.CreditCard, &s.CashReceipt, &s.CashOnHand, &s.Notes, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan sale: %w", err)
		}
		sales = append(sales, s)
	}
	return sales, rows.Err()
}

// ListRecentSalesGrouped returns recent sales grouped by date for dashboard display
func (db *DB) ListRecentSalesGrouped(days int) ([]models.DateGroup, float64, error) {
	rows, err := db.Query(`
		SELECT id, date(date), strftime('%m-%d-%Y', date), shift, net_sales, taxes, credit_card, cash_receipt, cash_on_hand, notes, created_at, updated_at
		FROM daily_sales
		WHERE date >= date('now', '-' || ? || ' days')
		ORDER BY date(date) DESC, CASE shift WHEN 'dinner' THEN 1 WHEN 'lunch' THEN 2 WHEN 'breakfast' THEN 3 END
	`, days)
	if err != nil {
		return nil, 0, fmt.Errorf("query recent sales: %w", err)
	}
	defer rows.Close()

	var dateGroups []models.DateGroup
	var grandTotal float64
	dateGroupMap := make(map[string]*models.DateGroup)

	for rows.Next() {
		var s models.DailySale
		var rawDate string
		if err := rows.Scan(&s.ID, &rawDate, &s.Date, &s.Shift, &s.NetSales, &s.Taxes, &s.CreditCard, &s.CashReceipt, &s.CashOnHand, &s.Notes, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan sale: %w", err)
		}

		if _, exists := dateGroupMap[rawDate]; !exists {
			dateGroupMap[rawDate] = &models.DateGroup{
				Date:      s.Date,
				RawDate:   rawDate,
				Collapsed: true,
			}
		}
		dateGroupMap[rawDate].Sales = append(dateGroupMap[rawDate].Sales, s)
		dateGroupMap[rawDate].Total += s.NetSales
		grandTotal += s.NetSales
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Convert map to slice, maintaining date order (most recent first)
	for _, group := range dateGroupMap {
		dateGroups = append(dateGroups, *group)
	}
	// Sort by raw date descending
	for i := 0; i < len(dateGroups); i++ {
		for j := i + 1; j < len(dateGroups); j++ {
			if dateGroups[j].RawDate > dateGroups[i].RawDate {
				dateGroups[i], dateGroups[j] = dateGroups[j], dateGroups[i]
			}
		}
	}

	// Fetch delivery data for all dates
	var dates []string
	for _, dg := range dateGroups {
		dates = append(dates, dg.RawDate)
	}
	deliveryMap, err := db.GetDeliverySalesForDates(dates)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch delivery sales: %w", err)
	}

	// Attach delivery data to each DateGroup
	for i := range dateGroups {
		if delivery, ok := deliveryMap[dateGroups[i].RawDate]; ok {
			dateGroups[i].Delivery = delivery
		}
	}

	return dateGroups, grandTotal, nil
}

func (db *DB) GetSale(id int64) (models.DailySale, error) {
	var s models.DailySale
	err := db.QueryRow(`
		SELECT id, date(date), shift, net_sales, taxes, credit_card, cash_receipt, cash_on_hand, notes, created_at, updated_at
		FROM daily_sales
		WHERE id = ?
	`, id).Scan(&s.ID, &s.Date, &s.Shift, &s.NetSales, &s.Taxes, &s.CreditCard, &s.CashReceipt, &s.CashOnHand, &s.Notes, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return s, fmt.Errorf("sale not found")
	}
	if err != nil {
		return s, fmt.Errorf("query sale: %w", err)
	}
	return s, nil
}

// UpsertSale creates or updates a sale for the given date+shift combination
func (db *DB) UpsertSale(s models.DailySale) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO daily_sales (date, shift, net_sales, taxes, credit_card, cash_receipt, cash_on_hand, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(date, shift) DO UPDATE SET
			net_sales = excluded.net_sales,
			taxes = excluded.taxes,
			credit_card = excluded.credit_card,
			cash_receipt = excluded.cash_receipt,
			cash_on_hand = excluded.cash_on_hand,
			notes = excluded.notes,
			updated_at = CURRENT_TIMESTAMP
	`, s.Date, s.Shift, s.NetSales, s.Taxes, s.CreditCard, s.CashReceipt, s.CashOnHand, s.Notes)
	if err != nil {
		return 0, fmt.Errorf("upsert sale: %w", err)
	}
	return result.LastInsertId()
}

func (db *DB) UpdateSale(s models.DailySale) error {
	_, err := db.Exec(`
		UPDATE daily_sales
		SET date = ?, shift = ?, net_sales = ?, taxes = ?, credit_card = ?, cash_receipt = ?, cash_on_hand = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, s.Date, s.Shift, s.NetSales, s.Taxes, s.CreditCard, s.CashReceipt, s.CashOnHand, s.Notes, s.ID)
	if err != nil {
		return fmt.Errorf("update sale: %w", err)
	}
	return nil
}

func (db *DB) DeleteSale(id int64) error {
	_, err := db.Exec(`DELETE FROM daily_sales WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete sale: %w", err)
	}
	return nil
}

// GetShiftsForDate returns the shifts that already have entries for a given date
func (db *DB) GetShiftsForDate(date string) ([]string, error) {
	rows, err := db.Query(`SELECT shift FROM daily_sales WHERE date = ?`, date)
	if err != nil {
		return nil, fmt.Errorf("query shifts for date: %w", err)
	}
	defer rows.Close()

	var shifts []string
	for rows.Next() {
		var shift string
		if err := rows.Scan(&shift); err != nil {
			return nil, fmt.Errorf("scan shift: %w", err)
		}
		shifts = append(shifts, shift)
	}
	return shifts, rows.Err()
}

// addToDateGroup adds a sale to the appropriate date group within a sales group
func addToDateGroup(group *models.SalesGroup, sale models.DailySale, rawDate, displayDate string) {
	// Find existing date group or create new one
	var dateGroup *models.DateGroup
	for i := range group.DateGroups {
		if group.DateGroups[i].RawDate == rawDate {
			dateGroup = &group.DateGroups[i]
			break
		}
	}
	if dateGroup == nil {
		group.DateGroups = append(group.DateGroups, models.DateGroup{
			Date:      displayDate,
			RawDate:   rawDate,
			Collapsed: true,
		})
		dateGroup = &group.DateGroups[len(group.DateGroups)-1]
	}
	dateGroup.Sales = append(dateGroup.Sales, sale)
	dateGroup.Total += sale.NetSales
	group.Total += sale.NetSales
}

// ListSalesGrouped returns all sales organized into time-based groups
func (db *DB) ListSalesGrouped() (*models.GroupedSalesData, error) {
	// Query all sales with raw date for grouping and formatted date for display
	rows, err := db.Query(`
		SELECT id, date(date), strftime('%m-%d-%Y', date), shift, net_sales, taxes, credit_card, cash_receipt, cash_on_hand, notes, created_at, updated_at
		FROM daily_sales
		ORDER BY date(date) DESC, CASE shift WHEN 'dinner' THEN 1 WHEN 'lunch' THEN 2 WHEN 'breakfast' THEN 3 END
	`)
	if err != nil {
		return nil, fmt.Errorf("query sales: %w", err)
	}
	defer rows.Close()

	// Determine time boundaries
	now := time.Now()
	today := now.Format("2006-01-02")

	// Start of month
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")

	currentYear := now.Year()
	currentMonth := now.Month()

	// Initialize groups
	result := &models.GroupedSalesData{
		Today:      &models.SalesGroup{Label: "Today", Collapsed: false},
		ThisMonth:  &models.SalesGroup{Label: now.Format("January 2006"), Collapsed: false},
		PrevMonths: []models.SalesGroup{},
		PrevYears:  []models.SalesGroup{},
	}

	// Maps for grouping previous months and years
	prevMonthsMap := make(map[string]*models.SalesGroup)
	prevYearsMap := make(map[int]*models.SalesGroup)

	for rows.Next() {
		var s models.DailySale
		var rawDate string
		if err := rows.Scan(&s.ID, &rawDate, &s.Date, &s.Shift, &s.NetSales, &s.Taxes, &s.CreditCard, &s.CashReceipt, &s.CashOnHand, &s.Notes, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan sale: %w", err)
		}

		// Parse the raw date to determine grouping
		saleDate, err := time.Parse("2006-01-02", rawDate)
		if err != nil {
			continue // Skip invalid dates
		}
		saleYear := saleDate.Year()

		// Determine which bucket this sale belongs to
		if rawDate == today {
			// Today - show individual shifts (no date grouping)
			result.Today.Sales = append(result.Today.Sales, s)
			result.Today.Total += s.NetSales
		} else if rawDate >= startOfMonth {
			// Current month (excluding today) - group by date
			addToDateGroup(result.ThisMonth, s, rawDate, s.Date)
		} else if saleYear == currentYear && saleDate.Month() != currentMonth {
			// Previous months in current year - group by date
			monthKey := saleDate.Format("January 2006")
			if _, exists := prevMonthsMap[monthKey]; !exists {
				prevMonthsMap[monthKey] = &models.SalesGroup{
					Label:     monthKey,
					Collapsed: true,
				}
			}
			addToDateGroup(prevMonthsMap[monthKey], s, rawDate, s.Date)
		} else {
			// Previous years - group by date
			if _, exists := prevYearsMap[saleYear]; !exists {
				prevYearsMap[saleYear] = &models.SalesGroup{
					Label:     fmt.Sprintf("%d", saleYear),
					Collapsed: true,
				}
			}
			addToDateGroup(prevYearsMap[saleYear], s, rawDate, s.Date)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Convert previous months map to sorted slice (most recent first)
	for _, group := range prevMonthsMap {
		result.PrevMonths = append(result.PrevMonths, *group)
	}
	// Sort by parsing the label back to time (most recent first)
	for i := 0; i < len(result.PrevMonths); i++ {
		for j := i + 1; j < len(result.PrevMonths); j++ {
			ti, _ := time.Parse("January 2006", result.PrevMonths[i].Label)
			tj, _ := time.Parse("January 2006", result.PrevMonths[j].Label)
			if tj.After(ti) {
				result.PrevMonths[i], result.PrevMonths[j] = result.PrevMonths[j], result.PrevMonths[i]
			}
		}
	}

	// Convert previous years map to sorted slice (most recent first)
	for _, group := range prevYearsMap {
		result.PrevYears = append(result.PrevYears, *group)
	}
	// Sort by year (most recent first)
	for i := 0; i < len(result.PrevYears); i++ {
		for j := i + 1; j < len(result.PrevYears); j++ {
			var yi, yj int
			fmt.Sscanf(result.PrevYears[i].Label, "%d", &yi)
			fmt.Sscanf(result.PrevYears[j].Label, "%d", &yj)
			if yj > yi {
				result.PrevYears[i], result.PrevYears[j] = result.PrevYears[j], result.PrevYears[i]
			}
		}
	}

	// Collect all unique dates to fetch delivery data
	dateSet := make(map[string]bool)
	dateSet[today] = true // Include today
	for _, dg := range result.ThisMonth.DateGroups {
		dateSet[dg.RawDate] = true
	}
	for _, group := range result.PrevMonths {
		for _, dg := range group.DateGroups {
			dateSet[dg.RawDate] = true
		}
	}
	for _, group := range result.PrevYears {
		for _, dg := range group.DateGroups {
			dateSet[dg.RawDate] = true
		}
	}

	// Convert to slice
	var dates []string
	for date := range dateSet {
		dates = append(dates, date)
	}

	// Fetch delivery data for all dates
	deliveryMap, err := db.GetDeliverySalesForDates(dates)
	if err != nil {
		return nil, fmt.Errorf("fetch delivery sales: %w", err)
	}

	// Attach delivery data to Today (create a synthetic DateGroup for today's delivery)
	if delivery, ok := deliveryMap[today]; ok {
		result.Today.Delivery = delivery
	}

	// Attach delivery data to ThisMonth DateGroups
	for i := range result.ThisMonth.DateGroups {
		if delivery, ok := deliveryMap[result.ThisMonth.DateGroups[i].RawDate]; ok {
			result.ThisMonth.DateGroups[i].Delivery = delivery
		}
	}

	// Attach delivery data to PrevMonths DateGroups
	for i := range result.PrevMonths {
		for j := range result.PrevMonths[i].DateGroups {
			if delivery, ok := deliveryMap[result.PrevMonths[i].DateGroups[j].RawDate]; ok {
				result.PrevMonths[i].DateGroups[j].Delivery = delivery
			}
		}
	}

	// Attach delivery data to PrevYears DateGroups
	for i := range result.PrevYears {
		for j := range result.PrevYears[i].DateGroups {
			if delivery, ok := deliveryMap[result.PrevYears[i].DateGroups[j].RawDate]; ok {
				result.PrevYears[i].DateGroups[j].Delivery = delivery
			}
		}
	}

	return result, nil
}
