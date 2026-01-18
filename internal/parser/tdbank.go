package parser

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// ParsedStatement represents a fully parsed bank statement
type ParsedStatement struct {
	AccountLastFour  string
	StatementMonth   string // YYYY-MM
	BeginningBalance float64
	EndingBalance    float64
	Transactions     []ParsedTransaction
	// Summary totals from statement (for verification)
	SummaryDeposits    float64
	SummaryPayments    float64
	SummaryChecks      float64
	SummaryFees        float64
	SummaryCredits     float64
	SummaryWithdrawals float64
}

// ParsedTransaction represents a single parsed transaction
type ParsedTransaction struct {
	PostingDate     string  // YYYY-MM-DD
	Description     string  // Full description (including continuation lines)
	Amount          float64 // Negative for debits, positive for credits
	TransactionType string  // deposit, credit, check, debit, ach, fee, transfer, withdrawal
	Category        string  // income_cards, income_delivery, expense, fee, transfer, etc.
	CheckNumber     string  // For checks only
	VendorHint      string  // Extracted vendor name (best guess)
	ReferenceNumber string  // Any reference/confirmation numbers
}

// TDBankParser parses TD Bank PDF statements
type TDBankParser struct {
	statementYear  int
	statementMonth int
	debug          bool
}

// NewTDBankParser creates a new TD Bank parser
func NewTDBankParser() *TDBankParser {
	return &TDBankParser{
		debug: true, // Enable debug output
	}
}

// Parse extracts data from a TD Bank PDF statement
func (p *TDBankParser) Parse(pdfPath string) (*ParsedStatement, error) {
	// Extract text using pdftotext
	text, err := p.extractText(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("extract text: %w", err)
	}

	p.debugLog("Extracted %d characters from PDF", len(text))

	// Truncate at check images (pages 9+) - they start with #XXXX patterns
	text = p.truncateAtCheckImages(text)
	p.debugLog("After truncation: %d characters", len(text))

	// Parse header to get balances and account info
	stmt, err := p.parseHeader(text)
	if err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}

	p.debugLog("Header parsed: Account=%s, Beginning=%.2f, Ending=%.2f",
		stmt.AccountLastFour, stmt.BeginningBalance, stmt.EndingBalance)

	// Parse summary totals from ACCOUNT SUMMARY section
	p.parseSummaryTotals(text, stmt)

	// Split into sections and parse each
	sections := p.splitSections(text)

	// Debug: show what we got in each section
	for name, content := range sections {
		lines := strings.Split(strings.TrimSpace(content), "\n")
		p.debugLog("Section '%s': %d lines", name, len(lines))
		if len(lines) > 0 && lines[0] != "" {
			p.debugLog("  First: %q", truncateString(lines[0], 80))
		}
		if len(lines) > 1 && lines[1] != "" {
			p.debugLog("  Second: %q", truncateString(lines[1], 80))
		}
	}

	// Parse each section type
	deposits := p.parseDeposits(sections["deposits"])
	p.debugLog("Parsed %d deposits", len(deposits))
	stmt.Transactions = append(stmt.Transactions, deposits...)

	credits := p.parseCredits(sections["credits"])
	p.debugLog("Parsed %d credits", len(credits))
	stmt.Transactions = append(stmt.Transactions, credits...)

	checks := p.parseChecks(sections["checks"])
	p.debugLog("Parsed %d checks", len(checks))
	stmt.Transactions = append(stmt.Transactions, checks...)

	payments := p.parsePayments(sections["payments"])
	p.debugLog("Parsed %d electronic payments", len(payments))
	stmt.Transactions = append(stmt.Transactions, payments...)

	withdrawals := p.parseWithdrawals(sections["withdrawals"])
	p.debugLog("Parsed %d withdrawals", len(withdrawals))
	stmt.Transactions = append(stmt.Transactions, withdrawals...)

	fees := p.parseFees(sections["fees"])
	p.debugLog("Parsed %d fees", len(fees))
	stmt.Transactions = append(stmt.Transactions, fees...)

	p.debugLog("Total transactions: %d", len(stmt.Transactions))

	// Verify parsed totals against summary
	p.verifyTotals(stmt)

	return stmt, nil
}

// parseSummaryTotals extracts the summary totals from ACCOUNT SUMMARY section
func (p *TDBankParser) parseSummaryTotals(text string, stmt *ParsedStatement) {
	// Find ACCOUNT SUMMARY section
	summaryIdx := strings.Index(text, "ACCOUNT SUMMARY")
	if summaryIdx == -1 {
		return
	}

	// Get a chunk of text after ACCOUNT SUMMARY (the summary table)
	summaryEnd := summaryIdx + 1000
	if summaryEnd > len(text) {
		summaryEnd = len(text)
	}
	summarySection := text[summaryIdx:summaryEnd]

	// Pattern: "Electronic Deposits                38,934.62"
	// These are on the same line, not followed by transaction details
	patterns := map[string]*float64{
		`Electronic Deposits\s+([\d,]+\.\d{2})`: &stmt.SummaryDeposits,
		`Electronic Payments\s+([\d,]+\.\d{2})`: &stmt.SummaryPayments,
		`Checks Paid\s+([\d,]+\.\d{2})`:         &stmt.SummaryChecks,
		`Service Charges\s+([\d,]+\.\d{2})`:     &stmt.SummaryFees,
		`Other Credits\s+([\d,]+\.\d{2})`:       &stmt.SummaryCredits,
		`Other Withdrawals\s+([\d,]+\.\d{2})`:   &stmt.SummaryWithdrawals,
	}

	for pattern, target := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(summarySection); len(match) > 1 {
			*target = parseAmount(match[1])
		}
	}

	p.debugLog("Summary totals: Deposits=%.2f, Payments=%.2f, Checks=%.2f, Fees=%.2f",
		stmt.SummaryDeposits, stmt.SummaryPayments, stmt.SummaryChecks, stmt.SummaryFees)
}

// verifyTotals compares parsed transaction totals against summary totals
func (p *TDBankParser) verifyTotals(stmt *ParsedStatement) {
	var depositSum, paymentSum, checkSum, feeSum, creditSum, withdrawalSum float64

	for _, txn := range stmt.Transactions {
		switch txn.TransactionType {
		case "deposit":
			depositSum += txn.Amount
		case "credit":
			creditSum += txn.Amount
		case "debit":
			paymentSum += -txn.Amount // Make positive for comparison
		case "check":
			checkSum += -txn.Amount
		case "fee":
			feeSum += -txn.Amount
		case "withdrawal":
			withdrawalSum += -txn.Amount
		}
	}

	p.debugLog("Verification (parsed vs expected):")
	p.debugLog("  Deposits:    %.2f vs %.2f (diff: %.2f)", depositSum, stmt.SummaryDeposits, depositSum-stmt.SummaryDeposits)
	p.debugLog("  Payments:    %.2f vs %.2f (diff: %.2f)", paymentSum, stmt.SummaryPayments, paymentSum-stmt.SummaryPayments)
	p.debugLog("  Checks:      %.2f vs %.2f (diff: %.2f)", checkSum, stmt.SummaryChecks, checkSum-stmt.SummaryChecks)
	p.debugLog("  Fees:        %.2f vs %.2f (diff: %.2f)", feeSum, stmt.SummaryFees, feeSum-stmt.SummaryFees)
	p.debugLog("  Credits:     %.2f vs %.2f (diff: %.2f)", creditSum, stmt.SummaryCredits, creditSum-stmt.SummaryCredits)
	p.debugLog("  Withdrawals: %.2f vs %.2f (diff: %.2f)", withdrawalSum, stmt.SummaryWithdrawals, withdrawalSum-stmt.SummaryWithdrawals)
}

// debugLog prints debug output if debug mode is enabled
func (p *TDBankParser) debugLog(format string, args ...interface{}) {
	if p.debug {
		log.Printf("[TDBankParser] "+format, args...)
	}
}

// truncateString shortens a string for display
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractText runs pdftotext to get text from PDF
func (p *TDBankParser) extractText(pdfPath string) (string, error) {
	cmd := exec.Command("pdftotext", "-layout", pdfPath, "-")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext failed: %w", err)
	}
	return string(output), nil
}

// truncateAtCheckImages removes the check image pages from the text
func (p *TDBankParser) truncateAtCheckImages(text string) string {
	// Find "DAILY BALANCE SUMMARY" - everything after the page containing this is check images
	balanceSummaryIdx := strings.LastIndex(text, "DAILY BALANCE SUMMARY")
	if balanceSummaryIdx == -1 {
		return text
	}

	// Find the end of the balance summary section (next page footer)
	afterSummary := text[balanceSummaryIdx:]
	footerIdx := strings.Index(afterSummary, "Call 1-800-937-2000")
	if footerIdx != -1 {
		return text[:balanceSummaryIdx+footerIdx]
	}

	return text
}

// Regex patterns for parsing
var (
	periodPattern           = regexp.MustCompile(`Statement Period:\s+(\w+)\s+\d+\s+(\d{4})`)
	accountPattern          = regexp.MustCompile(`Account\s*#[:\s]+[\d-]*(\d{4})`)
	beginningBalancePattern = regexp.MustCompile(`Beginning\s+Balance\s+\$?([\d,]+\.\d{2})`)
	endingBalancePattern    = regexp.MustCompile(`Ending\s+Balance\s+\$?(-?[\d,]+\.\d{2})`)
)

// parseHeader extracts account info and balances
func (p *TDBankParser) parseHeader(text string) (*ParsedStatement, error) {
	stmt := &ParsedStatement{}

	if match := accountPattern.FindStringSubmatch(text); len(match) > 1 {
		stmt.AccountLastFour = match[1]
	}

	if match := periodPattern.FindStringSubmatch(text); len(match) > 2 {
		year, _ := strconv.Atoi(match[2])
		p.statementYear = year
		p.statementMonth = monthNameToNumber(match[1])
		stmt.StatementMonth = fmt.Sprintf("%d-%02d", year, p.statementMonth)
	}

	if match := beginningBalancePattern.FindStringSubmatch(text); len(match) > 1 {
		stmt.BeginningBalance = parseAmount(match[1])
	}

	if match := endingBalancePattern.FindStringSubmatch(text); len(match) > 1 {
		stmt.EndingBalance = parseAmount(match[1])
	}

	return stmt, nil
}

// splitSections extracts each transaction section from the text
// This is the key function - it must correctly identify section boundaries
func (p *TDBankParser) splitSections(text string) map[string]string {
	sections := make(map[string]string)

	// Section patterns - we look for the header followed by POSTING DATE on next line
	// This distinguishes transaction sections from summary mentions
	//
	// Example in text:
	//   Electronic Deposits
	//   POSTING DATE      DESCRIPTION                                      AMOUNT
	//   12/01             CCD DEPOSIT...                                   1,148.90

	type sectionDef struct {
		// Patterns that start this section (including "continued" variants)
		startPatterns []*regexp.Regexp
		// Markers that end this section
		endMarkers []string
	}

	// Build regex patterns that match section headers
	// We require POSTING DATE or DATE header on a nearby line to confirm it's a transaction section
	sectionDefs := map[string]sectionDef{
		"deposits": {
			startPatterns: []*regexp.Regexp{
				regexp.MustCompile(`Electronic Deposits\s*\n+POSTING DATE`),
				regexp.MustCompile(`Electronic Deposits \(continued\)\s*\n+POSTING DATE`),
			},
			endMarkers: []string{"Other Credits", "Checks Paid", "Subtotal:"},
		},
		"credits": {
			startPatterns: []*regexp.Regexp{
				regexp.MustCompile(`Other Credits\s*\n+POSTING DATE`),
			},
			endMarkers: []string{"Checks Paid", "Electronic Payments", "Subtotal:"},
		},
		"checks": {
			startPatterns: []*regexp.Regexp{
				// Checks section has different header: DATE SERIAL NO. AMOUNT
				regexp.MustCompile(`Checks Paid[^\n]*\n+DATE\s+SERIAL`),
			},
			endMarkers: []string{"Electronic Payments", "Subtotal:"},
		},
		"payments": {
			startPatterns: []*regexp.Regexp{
				regexp.MustCompile(`Electronic Payments\s*\n+POSTING DATE`),
				regexp.MustCompile(`Electronic Payments \(continued\)\s*\n+POSTING DATE`),
			},
			endMarkers: []string{"Other Withdrawals", "Subtotal:"},
		},
		"withdrawals": {
			startPatterns: []*regexp.Regexp{
				regexp.MustCompile(`Other Withdrawals\s*\n+POSTING DATE`),
			},
			endMarkers: []string{"Service Charges", "Subtotal:"},
		},
		"fees": {
			startPatterns: []*regexp.Regexp{
				regexp.MustCompile(`Service Charges\s*\n+POSTING DATE`),
			},
			endMarkers: []string{"DAILY BALANCE SUMMARY", "Subtotal:"},
		},
	}

	// Page markers to stop at
	pageBreakMarkers := []string{
		"Call 1-800-937-2000",
		"Bank Deposits FDIC Insured",
	}

	for sectionName, def := range sectionDefs {
		var allContent strings.Builder

		for _, pattern := range def.startPatterns {
			// Find all matches of this pattern
			matches := pattern.FindAllStringIndex(text, -1)

			for _, match := range matches {
				// Start after the header lines (pattern includes POSTING DATE line)
				startIdx := match[1]

				// Skip to next line after POSTING DATE header
				remaining := text[startIdx:]
				if nlIdx := strings.Index(remaining, "\n"); nlIdx != -1 {
					startIdx += nlIdx + 1
				}

				// Find end of this section chunk
				remaining = text[startIdx:]
				endIdx := len(remaining)

				// Check section end markers
				for _, endMarker := range def.endMarkers {
					if idx := strings.Index(remaining, endMarker); idx != -1 && idx < endIdx {
						endIdx = idx
					}
				}

				// Check page break markers
				for _, marker := range pageBreakMarkers {
					if idx := strings.Index(remaining, marker); idx != -1 && idx < endIdx {
						endIdx = idx
					}
				}

				// Extract and clean the chunk
				chunk := remaining[:endIdx]
				chunk = p.cleanSectionContent(chunk)

				if strings.TrimSpace(chunk) != "" {
					allContent.WriteString(chunk)
					allContent.WriteString("\n")
				}
			}
		}

		sections[sectionName] = allContent.String()
	}

	return sections
}

// cleanSectionContent removes page headers/footers and other noise
func (p *TDBankParser) cleanSectionContent(content string) string {
	lines := strings.Split(content, "\n")
	var cleaned []string

	skipPatterns := []string{
		"Call 1-800-937-2000",
		"Bank Deposits FDIC Insured",
		"STATEMENT OF ACCOUNT",
		"xxxxxx",
		"Page:",
		"DAILY ACCOUNT ACTIVITY",
		"POSTING DATE",
		"TRINI BREAKFAST SHED",
		"Statement Period:",
		"Cust Ref #:",
		"Primary Account #:",
	}

	for _, line := range lines {
		skip := false
		trimmed := strings.TrimSpace(line)

		// Skip empty lines at start
		if trimmed == "" && len(cleaned) == 0 {
			continue
		}

		// Skip page markers
		for _, pattern := range skipPatterns {
			if strings.Contains(line, pattern) {
				skip = true
				break
			}
		}

		// Skip lines that are just whitespace or very short
		if len(trimmed) < 3 {
			skip = true
		}

		if !skip {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}

// Transaction line pattern - date at start, amount at end
var transactionLinePattern = regexp.MustCompile(`^(\d{2}/\d{2})\s+(.+?)\s{2,}([\d,]+\.\d{2})\s*$`)

// Continuation line - starts with significant whitespace, no date
var continuationLinePattern = regexp.MustCompile(`^\s{6,}(\S.*)$`)

// parseDeposits parses the Electronic Deposits section
func (p *TDBankParser) parseDeposits(section string) []ParsedTransaction {
	if strings.TrimSpace(section) == "" {
		return nil
	}

	var transactions []ParsedTransaction
	scanner := bufio.NewScanner(strings.NewReader(section))

	for scanner.Scan() {
		line := scanner.Text()

		if match := transactionLinePattern.FindStringSubmatch(line); len(match) > 3 {
			txn := ParsedTransaction{
				PostingDate:     p.formatDate(match[1]),
				Description:     strings.TrimSpace(match[2]),
				Amount:          parseAmount(match[3]), // Positive for deposits
				TransactionType: "deposit",
			}
			txn.Category = categorizeTransaction(txn.TransactionType, txn.Description, txn.Amount)
			txn.VendorHint = extractVendorHint(txn.Description)
			transactions = append(transactions, txn)
		}
	}

	return transactions
}

// parseCredits parses the Other Credits section
func (p *TDBankParser) parseCredits(section string) []ParsedTransaction {
	if strings.TrimSpace(section) == "" {
		return nil
	}

	var transactions []ParsedTransaction
	scanner := bufio.NewScanner(strings.NewReader(section))

	for scanner.Scan() {
		line := scanner.Text()

		if match := transactionLinePattern.FindStringSubmatch(line); len(match) > 3 {
			txn := ParsedTransaction{
				PostingDate:     p.formatDate(match[1]),
				Description:     strings.TrimSpace(match[2]),
				Amount:          parseAmount(match[3]), // Positive for credits
				TransactionType: "credit",
			}
			txn.Category = categorizeTransaction(txn.TransactionType, txn.Description, txn.Amount)
			txn.VendorHint = extractVendorHint(txn.Description)
			transactions = append(transactions, txn)
		}
	}

	return transactions
}

// parseChecks parses the Checks Paid section
// Format is a two-column table:
// DATE  SERIAL NO.  AMOUNT    DATE  SERIAL NO.  AMOUNT
// 12/01 2730        500.00    12/18 2739*       1,614.50
func (p *TDBankParser) parseChecks(section string) []ParsedTransaction {
	if strings.TrimSpace(section) == "" {
		return nil
	}

	var transactions []ParsedTransaction
	seen := make(map[string]bool) // Dedupe by check number

	// Pattern matches: DATE SERIAL AMOUNT
	checkPattern := regexp.MustCompile(`(\d{2}/\d{2})\s+(\d+)\*?\s+([\d,]+\.\d{2})`)

	matches := checkPattern.FindAllStringSubmatch(section, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			checkNum := match[2]

			if seen[checkNum] {
				continue
			}
			seen[checkNum] = true

			txn := ParsedTransaction{
				PostingDate:     p.formatDate(match[1]),
				Description:     "Check #" + checkNum,
				Amount:          -parseAmount(match[3]), // Negative for checks
				TransactionType: "check",
				CheckNumber:     checkNum,
				Category:        "expense_check",
			}
			transactions = append(transactions, txn)
		}
	}

	return transactions
}

// parsePayments parses the Electronic Payments section
// These entries can span multiple lines
func (p *TDBankParser) parsePayments(section string) []ParsedTransaction {
	if strings.TrimSpace(section) == "" {
		return nil
	}

	var transactions []ParsedTransaction
	var currentTxn *ParsedTransaction

	scanner := bufio.NewScanner(strings.NewReader(section))
	for scanner.Scan() {
		line := scanner.Text()

		// Check for new transaction line (starts with date)
		if match := transactionLinePattern.FindStringSubmatch(line); len(match) > 3 {
			// Save previous transaction
			if currentTxn != nil {
				p.finalizeTransaction(currentTxn)
				transactions = append(transactions, *currentTxn)
			}

			currentTxn = &ParsedTransaction{
				PostingDate:     p.formatDate(match[1]),
				Description:     strings.TrimSpace(match[2]),
				Amount:          -parseAmount(match[3]), // Negative for payments
				TransactionType: "debit",
			}
		} else if currentTxn != nil {
			// Check for continuation line
			if match := continuationLinePattern.FindStringSubmatch(line); len(match) > 1 {
				continuation := strings.TrimSpace(match[1])
				// Skip card numbers (16 digits)
				if !regexp.MustCompile(`^\d{16}$`).MatchString(continuation) {
					currentTxn.Description += " " + continuation
				}
			}
		}
	}

	// Don't forget the last transaction
	if currentTxn != nil {
		p.finalizeTransaction(currentTxn)
		transactions = append(transactions, *currentTxn)
	}

	return transactions
}

// parseWithdrawals parses the Other Withdrawals section
func (p *TDBankParser) parseWithdrawals(section string) []ParsedTransaction {
	if strings.TrimSpace(section) == "" {
		return nil
	}

	var transactions []ParsedTransaction
	var currentTxn *ParsedTransaction

	scanner := bufio.NewScanner(strings.NewReader(section))
	for scanner.Scan() {
		line := scanner.Text()

		if match := transactionLinePattern.FindStringSubmatch(line); len(match) > 3 {
			if currentTxn != nil {
				p.finalizeTransaction(currentTxn)
				transactions = append(transactions, *currentTxn)
			}

			currentTxn = &ParsedTransaction{
				PostingDate:     p.formatDate(match[1]),
				Description:     strings.TrimSpace(match[2]),
				Amount:          -parseAmount(match[3]), // Negative for withdrawals
				TransactionType: "withdrawal",
			}
		} else if currentTxn != nil {
			if match := continuationLinePattern.FindStringSubmatch(line); len(match) > 1 {
				currentTxn.Description += " " + strings.TrimSpace(match[1])
			}
		}
	}

	if currentTxn != nil {
		p.finalizeTransaction(currentTxn)
		transactions = append(transactions, *currentTxn)
	}

	return transactions
}

// parseFees parses the Service Charges section
func (p *TDBankParser) parseFees(section string) []ParsedTransaction {
	if strings.TrimSpace(section) == "" {
		return nil
	}

	var transactions []ParsedTransaction
	scanner := bufio.NewScanner(strings.NewReader(section))

	for scanner.Scan() {
		line := scanner.Text()

		if match := transactionLinePattern.FindStringSubmatch(line); len(match) > 3 {
			txn := ParsedTransaction{
				PostingDate:     p.formatDate(match[1]),
				Description:     strings.TrimSpace(match[2]),
				Amount:          -parseAmount(match[3]), // Negative for fees
				TransactionType: "fee",
				Category:        "fee",
				VendorHint:      "TD Bank",
			}
			transactions = append(transactions, txn)
		}
	}

	return transactions
}

// finalizeTransaction sets category and vendor hint
func (p *TDBankParser) finalizeTransaction(txn *ParsedTransaction) {
	txn.Category = categorizeTransaction(txn.TransactionType, txn.Description, txn.Amount)
	txn.VendorHint = extractVendorHint(txn.Description)
}

// formatDate converts MM/DD to YYYY-MM-DD
func (p *TDBankParser) formatDate(mmdd string) string {
	parts := strings.Split(mmdd, "/")
	if len(parts) != 2 {
		return ""
	}

	month, _ := strconv.Atoi(parts[0])
	day, _ := strconv.Atoi(parts[1])

	year := p.statementYear
	if p.statementMonth == 12 && month == 1 {
		year++
	} else if p.statementMonth == 1 && month == 12 {
		year--
	}

	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}

// parseAmount converts "1,234.56" or "-1,234.56" to float64
func parseAmount(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "$", "")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func monthNameToNumber(name string) int {
	months := map[string]int{
		"jan": 1, "january": 1,
		"feb": 2, "february": 2,
		"mar": 3, "march": 3,
		"apr": 4, "april": 4,
		"may": 5,
		"jun": 6, "june": 6,
		"jul": 7, "july": 7,
		"aug": 8, "august": 8,
		"sep": 9, "september": 9,
		"oct": 10, "october": 10,
		"nov": 11, "november": 11,
		"dec": 12, "december": 12,
	}
	return months[strings.ToLower(name)]
}

// Vendor extraction patterns
var vendorPatterns = map[string]*regexp.Regexp{
	"Jetro":           regexp.MustCompile(`(?i)JETRO`),
	"Chef's Choice":   regexp.MustCompile(`(?i)CHEF.?S?\s*CHOICE`),
	"Cogent Waste":    regexp.MustCompile(`(?i)COGENT\s*WASTE`),
	"Con Edison":      regexp.MustCompile(`(?i)CON\s*ED`),
	"National Grid":   regexp.MustCompile(`(?i)NGRID|NATIONAL\s*GRID`),
	"Uber Eats":       regexp.MustCompile(`(?i)UBER`),
	"Grubhub":         regexp.MustCompile(`(?i)GRUBHUB`),
	"DoorDash":        regexp.MustCompile(`(?i)DOORDASH`),
	"Verizon":         regexp.MustCompile(`(?i)VERIZON`),
	"AT&T":            regexp.MustCompile(`(?i)\bATT\s|AT&T`),
	"Sampar's":        regexp.MustCompile(`(?i)SAMPARS?`),
	"Clover":          regexp.MustCompile(`(?i)CLOVER`),
	"Cintas":          regexp.MustCompile(`(?i)CINTAS`),
	"Dish Network":    regexp.MustCompile(`(?i)DISH\s*NETWORK`),
	"Bankcard":        regexp.MustCompile(`(?i)BANKCARD\s*MTOT`),
	"Gobwa Exotic":    regexp.MustCompile(`(?i)GOBWA\s*EXOTIC`),
	"Caribbean Depot": regexp.MustCompile(`(?i)CARIBBEAN\s*DEPOT`),
	"C&S Meats":       regexp.MustCompile(`(?i)C\s*AND\s*S\s*MEATS`),
	"Good Food":       regexp.MustCompile(`(?i)GOOD\s*FOOD\s*FOR\s*LESS`),
	"INP Foods":       regexp.MustCompile(`(?i)INP\s*FOODS`),
	"Wegmans":         regexp.MustCompile(`(?i)WEGMANS`),
}

func extractVendorHint(description string) string {
	for vendor, pattern := range vendorPatterns {
		if pattern.MatchString(description) {
			return vendor
		}
	}

	// Try BUSINESS NAME CITY * STATE pattern
	cityStatePattern := regexp.MustCompile(`([A-Z][A-Z0-9\s&']+?)\s+(?:[A-Z]+\s+)?\*\s*[A-Z]{2}`)
	if match := cityStatePattern.FindStringSubmatch(description); len(match) > 1 {
		vendor := strings.TrimSpace(match[1])
		vendor = regexp.MustCompile(`^(DBCRD|DEBIT|POS|AP|AUT|VISA|DDA|PUR)\s+`).ReplaceAllString(vendor, "")
		if len(vendor) > 3 {
			return vendor
		}
	}

	return ""
}

func categorizeTransaction(txnType, description string, amount float64) string {
	descUpper := strings.ToUpper(description)

	if amount > 0 {
		if strings.Contains(descUpper, "BANKCARD") || strings.Contains(descUpper, "MTOT DEP") {
			return "income_cards"
		}
		if strings.Contains(descUpper, "UBER") {
			return "income_delivery"
		}
		if strings.Contains(descUpper, "GRUBHUB") {
			return "income_delivery"
		}
		if strings.Contains(descUpper, "DOORDASH") {
			return "income_delivery"
		}
		if strings.Contains(descUpper, "REFUND") {
			return "refund"
		}
		if strings.Contains(descUpper, "OD GRACE") || strings.Contains(descUpper, "FEE REFUND") {
			return "refund"
		}
		return "income_other"
	}

	if txnType == "check" {
		return "expense_check"
	}
	if txnType == "fee" {
		return "fee"
	}
	if strings.Contains(descUpper, "OVERDRAFT") {
		return "fee"
	}
	if strings.Contains(descUpper, "TRANSFER") || strings.Contains(descUpper, "XFER") {
		return "transfer"
	}
	if strings.Contains(descUpper, "ATM") {
		return "atm"
	}

	return "expense"
}
