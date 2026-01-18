package reconciliation

import (
	"math"
	"time"

	"homebooks/internal/database"
)

// AutoMatch attempts to automatically match bank transactions to expenses
// Returns the number of transactions matched
func AutoMatch(db *database.DB, reconciliationID int64) (int, error) {
	// Get the reconciliation to determine date range
	recon, err := db.GetReconciliation(reconciliationID)
	if err != nil {
		return 0, err
	}

	// Get unmatched transactions
	transactions, err := db.GetUnmatchedBankTransactions(reconciliationID)
	if err != nil {
		return 0, err
	}

	// Parse statement date to get month range
	stmtDate, err := time.Parse("2006-01-02", recon.StatementDate)
	if err != nil {
		return 0, err
	}

	// Get expenses from the statement month (with some buffer)
	startDate := time.Date(stmtDate.Year(), stmtDate.Month(), 1, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Second) // Last moment of the month

	// Add buffer for transactions that might post late
	startDate = startDate.AddDate(0, 0, -7) // 7 days before
	endDate = endDate.AddDate(0, 0, 7)      // 7 days after

	expenses, err := db.ListExpensesDateRange(startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return 0, err
	}

	matched := 0

	for _, txn := range transactions {
		// Skip non-expense transactions (deposits, credits)
		if txn.Amount >= 0 {
			continue
		}

		// Skip fees - they don't match to expenses
		if txn.TransactionType == "fee" {
			continue
		}

		txnAmount := math.Abs(txn.Amount)

		// Try to find a matching expense
		for _, exp := range expenses {
			// Skip unpaid expenses
			if exp.Status != "paid" {
				continue
			}

			// Skip already matched expenses
			if isExpenseMatched(db, reconciliationID, exp.ID) {
				continue
			}

			// Match strategy 1: Check number exact match (highest confidence)
			if txn.CheckNumber != "" && exp.CheckNumber != "" && txn.CheckNumber == exp.CheckNumber {
				if err := db.MatchBankTransaction(txn.ID, exp.ID, "auto_exact"); err == nil {
					matched++
					break
				}
			}

			// Match strategy 2: Exact amount + close date
			if txnAmount == exp.Amount {
				txnDate, _ := time.Parse("2006-01-02", txn.PostingDate)
				expDate, _ := time.Parse("2006-01-02", exp.DatePaid)

				daysDiff := math.Abs(txnDate.Sub(expDate).Hours() / 24)

				if daysDiff <= 3 {
					if err := db.MatchBankTransaction(txn.ID, exp.ID, "auto_fuzzy"); err == nil {
						matched++
						break
					}
				}
			}

			// Match strategy 3: Amount match + vendor hint
			if txnAmount == exp.Amount && txn.VendorHint != "" {
				// Check if vendor hint matches expense vendor name (case insensitive)
				if containsIgnoreCase(exp.VendorName, txn.VendorHint) {
					if err := db.MatchBankTransaction(txn.ID, exp.ID, "auto_fuzzy"); err == nil {
						matched++
						break
					}
				}
			}
		}
	}

	return matched, nil
}

// isExpenseMatched checks if an expense is already matched to a transaction in this reconciliation
func isExpenseMatched(db *database.DB, reconciliationID, expenseID int64) bool {
	transactions, _ := db.GetBankTransactions(reconciliationID)
	for _, t := range transactions {
		if t.MatchedExpenseID != nil && *t.MatchedExpenseID == expenseID {
			return true
		}
	}
	return false
}

// containsIgnoreCase checks if haystack contains needle (case insensitive)
func containsIgnoreCase(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		(haystack == needle ||
			containsLower(toLower(haystack), toLower(needle)))
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func containsLower(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
