package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"homebooks/internal/database"
	"homebooks/internal/models"
	"homebooks/internal/parser"
	"homebooks/internal/reconciliation"
)

// ParseStatementPayload is the JSON payload for parse_statement jobs
type ParseStatementPayload struct {
	ReconciliationID int64  `json:"reconciliation_id"`
	FilePath         string `json:"file_path"`
}

// ParseStatementHandler creates a job handler for parsing bank statements
func ParseStatementHandler(fileStorePath string) JobHandler {
	return func(ctx context.Context, job *models.Job, db *database.DB) error {
		// Parse payload
		var payload ParseStatementPayload
		if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
			return fmt.Errorf("unmarshal payload: %w", err)
		}

		// Update status to parsing
		if err := db.UpdateReconciliationStatus(payload.ReconciliationID, "parsing"); err != nil {
			return fmt.Errorf("update status: %w", err)
		}
		db.UpdateJobProgress(job.ID, 5)

		// Build full file path
		fullPath := fileStorePath + "/" + payload.FilePath

		// Parse the PDF
		p := parser.NewTDBankParser()
		result, err := p.Parse(fullPath)
		if err != nil {
			db.UpdateReconciliationStatus(payload.ReconciliationID, "pending")
			return fmt.Errorf("parse PDF: %w", err)
		}
		db.UpdateJobProgress(job.ID, 40)

		// Update reconciliation with parsed header info and summary values
		if err := db.UpdateReconciliationParsed(
			payload.ReconciliationID,
			result.BeginningBalance,
			result.EndingBalance,
			result.AccountLastFour,
			result.ElectronicDeposits,
			result.ElectronicPayments,
			result.ChecksPaid,
			result.ServiceFees,
		); err != nil {
			return fmt.Errorf("update reconciliation parsed: %w", err)
		}
		db.UpdateJobProgress(job.ID, 50)

		// Delete any existing transactions (in case of re-parse)
		if err := db.DeleteBankTransactions(payload.ReconciliationID); err != nil {
			return fmt.Errorf("delete existing transactions: %w", err)
		}

		// Insert all transactions
		totalTxns := len(result.Transactions)
		for i, txn := range result.Transactions {
			bankTxn := &models.BankTransaction{
				ReconciliationID: payload.ReconciliationID,
				PostingDate:      txn.PostingDate,
				Description:      txn.Description,
				Amount:           txn.Amount,
				TransactionType:  txn.TransactionType,
				Category:         txn.Category,
				CheckNumber:      txn.CheckNumber,
				VendorHint:       txn.VendorHint,
				ReferenceNumber:  txn.ReferenceNumber,
				MatchStatus:      "unmatched",
			}

			if _, err := db.CreateBankTransaction(bankTxn); err != nil {
				return fmt.Errorf("create transaction %d: %w", i, err)
			}

			// Update progress periodically
			if i%10 == 0 || i == totalTxns-1 {
				progress := 50 + (40 * (i + 1) / totalTxns)
				db.UpdateJobProgress(job.ID, progress)
			}

			// Check for cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		db.UpdateJobProgress(job.ID, 95)

		// Run auto-matching
		matched, _ := reconciliation.AutoMatch(db, payload.ReconciliationID)

		// Mark as parsed (not completed - user still needs to review)
		if err := db.UpdateReconciliationStatus(payload.ReconciliationID, "parsed"); err != nil {
			return fmt.Errorf("update status to parsed: %w", err)
		}

		db.UpdateJobProgress(job.ID, 100)

		// Set result with summary
		resultJSON, _ := json.Marshal(map[string]any{
			"transactions_count": totalTxns,
			"matched_count":      matched,
			"beginning_balance":  result.BeginningBalance,
			"ending_balance":     result.EndingBalance,
			"account_last_four":  result.AccountLastFour,
		})
		db.CompleteJob(job.ID, string(resultJSON))

		return nil
	}
}
