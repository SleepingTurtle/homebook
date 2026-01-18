package main

import (
	"fmt"
	"os"

	"homebooks/internal/parser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: parsetest <path-to-pdf>")
		os.Exit(1)
	}

	pdfPath := os.Args[1]

	p := parser.NewTDBankParser()
	result, err := p.Parse(pdfPath)
	if err != nil {
		fmt.Printf("Error parsing PDF: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Account (last 4): %s\n", result.AccountLastFour)
	fmt.Printf("Beginning Balance: $%.2f\n", result.BeginningBalance)
	fmt.Printf("Ending Balance: $%.2f\n", result.EndingBalance)
	fmt.Printf("Transactions: %d\n\n", len(result.Transactions))

	// Summary by type
	typeCounts := make(map[string]int)
	typeAmounts := make(map[string]float64)
	for _, txn := range result.Transactions {
		typeCounts[txn.TransactionType]++
		typeAmounts[txn.TransactionType] += txn.Amount
	}

	fmt.Println("Summary by Type:")
	fmt.Println("----------------")
	for t, count := range typeCounts {
		fmt.Printf("  %-12s: %3d transactions, total: $%10.2f\n", t, count, typeAmounts[t])
	}

	// List all transactions
	fmt.Println("\nAll Transactions:")
	fmt.Println("-----------------")
	for _, txn := range result.Transactions {
		vendorInfo := ""
		if txn.VendorHint != "" {
			vendorInfo = fmt.Sprintf(" [%s]", txn.VendorHint)
		}
		checkInfo := ""
		if txn.CheckNumber != "" {
			checkInfo = fmt.Sprintf(" (Check #%s)", txn.CheckNumber)
		}
		fmt.Printf("  %s | %-10s | %10.2f | %-12s | %s%s%s\n",
			txn.PostingDate,
			txn.TransactionType,
			txn.Amount,
			txn.Category,
			truncate(txn.Description, 50),
			vendorInfo,
			checkInfo,
		)
	}

	// Verify balance
	fmt.Println("\nBalance Verification:")
	fmt.Println("---------------------")
	var totalCredits, totalDebits float64
	for _, txn := range result.Transactions {
		if txn.Amount > 0 {
			totalCredits += txn.Amount
		} else {
			totalDebits += txn.Amount
		}
	}
	calculatedEnding := result.BeginningBalance + totalCredits + totalDebits
	fmt.Printf("  Beginning Balance:  $%10.2f\n", result.BeginningBalance)
	fmt.Printf("  Total Credits:      $%10.2f\n", totalCredits)
	fmt.Printf("  Total Debits:       $%10.2f\n", totalDebits)
	fmt.Printf("  Calculated Ending:  $%10.2f\n", calculatedEnding)
	fmt.Printf("  Statement Ending:   $%10.2f\n", result.EndingBalance)
	diff := calculatedEnding - result.EndingBalance
	if diff != 0 {
		fmt.Printf("  DIFFERENCE:         $%10.2f (may indicate missing transactions)\n", diff)
	} else {
		fmt.Println("  Balances match!")
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
