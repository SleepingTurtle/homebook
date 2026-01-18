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
	ElectronicDeposits float64
	ElectronicPayments float64
	ChecksPaid         float64
	ServiceFees        float64
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

	// Extract subtotals from the full text (sections end at "Subtotal:" so we can't get it from section content)
	// Use section headers with newlines to avoid matching summary lines at top of statement
	stmt.ElectronicDeposits = p.extractSectionSubtotal(text, "Electronic Deposits\n")
	stmt.ElectronicPayments = p.extractSectionSubtotal(text, "Electronic Payments\n")
	stmt.ChecksPaid = p.extractSectionSubtotal(text, "Checks Paid\n")
	stmt.ServiceFees = p.extractSectionSubtotal(text, "Service Charges\n")

	p.debugLog("Subtotals: Deposits=%.2f, Payments=%.2f, Checks=%.2f, Fees=%.2f",
		stmt.ElectronicDeposits, stmt.ElectronicPayments, stmt.ChecksPaid, stmt.ServiceFees)

	return stmt, nil
}

// extractSectionSubtotal finds the subtotal for a specific section in the full text
// It looks for the section header, then finds "Subtotal:" before the next section starts
func (p *TDBankParser) extractSectionSubtotal(text, sectionName string) float64 {
	// Find the section header
	idx := strings.Index(text, sectionName)
	if idx == -1 {
		return 0
	}

	// Look for "Subtotal:" after this section header but before the next section
	afterSection := text[idx+len(sectionName):]

	// Find where the next section starts (to limit our search)
	nextSectionMarkers := []string{
		"Electronic Deposits\n",
		"Other Credits\n",
		"Checks Paid\n",
		"Electronic Payments\n",
		"Other Withdrawals\n",
		"Service Charges\n",
		"DAILY BALANCE SUMMARY",
	}

	endIdx := len(afterSection)
	for _, marker := range nextSectionMarkers {
		if markerIdx := strings.Index(afterSection, marker); markerIdx != -1 && markerIdx < endIdx {
			endIdx = markerIdx
		}
	}

	// Search for subtotal only within this section's bounds
	sectionContent := afterSection[:endIdx]

	// Find the subtotal line - it appears after the transactions
	subtotalRe := regexp.MustCompile(`Subtotal:\s*\$?([\d,]+\.\d{2})`)
	if match := subtotalRe.FindStringSubmatch(sectionContent); len(match) > 1 {
		return parseAmount(match[1])
	}
	return 0
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
// These pages have format like "#2730     12/01                $500.00"
func (p *TDBankParser) truncateAtCheckImages(text string) string {
	// Find "DAILY BALANCE SUMMARY" - everything after this section is check images
	// The balance summary is followed by date/balance pairs, then check images start
	balanceSummaryIdx := strings.LastIndex(text, "DAILY BALANCE SUMMARY")
	if balanceSummaryIdx == -1 {
		return text
	}

	// Find the end of the balance summary section
	// Look for the page footer after the balance table
	afterSummary := text[balanceSummaryIdx:]
	footerIdx := strings.Index(afterSummary, "Call 1-800-937-2000")
	if footerIdx != -1 {
		// Include up to the footer
		return text[:balanceSummaryIdx+footerIdx]
	}

	return text
}

// Regex patterns for parsing
var (
	// Statement period: "Dec 01 2025-Dec 31 2025" or "Statement Period: Dec 01 2025-Dec 31 2025"
	periodPattern = regexp.MustCompile(`Statement Period:\s+(\w+)\s+\d+\s+(\d{4})`)

	// Account number: "Account # 428-0712609" or "Primary Account #: 428-0712609"
	accountPattern = regexp.MustCompile(`Account\s*#[:\s]+[\d-]*(\d{4})`)

	// Balance patterns - handle both with and without dollar sign
	beginningBalancePattern = regexp.MustCompile(`Beginning\s+Balance\s+\$?([\d,]+\.\d{2})`)
	endingBalancePattern    = regexp.MustCompile(`Ending\s+Balance\s+\$?(-?[\d,]+\.\d{2})`)
)

// parseHeader extracts account info and balances
func (p *TDBankParser) parseHeader(text string) (*ParsedStatement, error) {
	stmt := &ParsedStatement{}

	// Extract account number (last 4 digits)
	if match := accountPattern.FindStringSubmatch(text); len(match) > 1 {
		stmt.AccountLastFour = match[1]
	}

	// Extract statement period to determine year and month
	if match := periodPattern.FindStringSubmatch(text); len(match) > 2 {
		year, _ := strconv.Atoi(match[2])
		p.statementYear = year
		p.statementMonth = monthNameToNumber(match[1])
		stmt.StatementMonth = fmt.Sprintf("%d-%02d", year, p.statementMonth)
	}

	// Extract beginning balance
	if match := beginningBalancePattern.FindStringSubmatch(text); len(match) > 1 {
		stmt.BeginningBalance = parseAmount(match[1])
	}

	// Extract ending balance (can be negative)
	if match := endingBalancePattern.FindStringSubmatch(text); len(match) > 1 {
		stmt.EndingBalance = parseAmount(match[1])
	}

	return stmt, nil
}

// splitSections divides the text into named sections, handling "(continued)" headers
func (p *TDBankParser) splitSections(text string) map[string]string {
	sections := make(map[string]string)

	// Define section headers (including continued variations)
	type sectionDef struct {
		headers []string
		ends    []string
	}

	sectionDefs := map[string]sectionDef{
		"deposits": {
			headers: []string{"Electronic Deposits\n", "Electronic Deposits (continued)"},
			ends:    []string{"Other Credits", "Checks Paid", "Subtotal:"},
		},
		"credits": {
			headers: []string{"Other Credits\n"},
			ends:    []string{"Checks Paid", "Electronic Payments", "Subtotal:"},
		},
		"checks": {
			headers: []string{"Checks Paid"},
			ends:    []string{"Electronic Payments", "Subtotal:"},
		},
		"payments": {
			headers: []string{"Electronic Payments\n", "Electronic Payments (continued)"},
			ends:    []string{"Other Withdrawals", "Subtotal:"},
		},
		"withdrawals": {
			headers: []string{"Other Withdrawals\n"},
			ends:    []string{"Service Charges", "Subtotal:"},
		},
		"fees": {
			headers: []string{"Service Charges\n"},
			ends:    []string{"DAILY BALANCE SUMMARY", "Subtotal:"},
		},
	}

	// Page markers to skip
	pageMarkers := []string{
		"Call 1-800-937-2000",
		"Bank Deposits FDIC Insured",
		"STATEMENT OF ACCOUNT",
		"TRINI BREAKFAST SHED",
		"Page:",
		"Statement Period:",
		"DAILY ACCOUNT ACTIVITY",
		"POSTING DATE",
	}

	for sectionName, def := range sectionDefs {
		var allContent strings.Builder

		for _, header := range def.headers {
			// Find all occurrences of this header
			searchStart := 0
			for {
				idx := strings.Index(text[searchStart:], header)
				if idx == -1 {
					break
				}
				absIdx := searchStart + idx

				// Start after the header
				contentStart := absIdx + len(header)

				// Skip the column header line (POSTING DATE DESCRIPTION AMOUNT)
				remaining := text[contentStart:]
				if strings.HasPrefix(strings.TrimSpace(remaining), "POSTING DATE") {
					if nlIdx := strings.Index(remaining, "\n"); nlIdx != -1 {
						contentStart += nlIdx + 1
					}
				}

				// For checks section, skip the "No. Checks:" line and column headers
				if sectionName == "checks" {
					remaining = text[contentStart:]
					// Skip until we see the first date pattern
					lines := strings.Split(remaining, "\n")
					for i, line := range lines {
						if regexp.MustCompile(`^\d{2}/\d{2}\s`).MatchString(strings.TrimSpace(line)) {
							// Found first transaction line
							skipLen := 0
							for j := 0; j < i; j++ {
								skipLen += len(lines[j]) + 1
							}
							contentStart += skipLen
							break
						}
					}
				}

				// Find the end of this section chunk
				contentEnd := len(text)
				remaining = text[contentStart:]

				for _, endMarker := range def.ends {
					if endIdx := strings.Index(remaining, endMarker); endIdx != -1 {
						if contentStart+endIdx < contentEnd {
							contentEnd = contentStart + endIdx
						}
					}
				}

				// Also end at page breaks
				for _, pageMarker := range pageMarkers {
					if endIdx := strings.Index(remaining, pageMarker); endIdx != -1 {
						if contentStart+endIdx < contentEnd {
							contentEnd = contentStart + endIdx
						}
					}
				}

				// Extract content
				chunk := text[contentStart:contentEnd]

				// Clean up the chunk - remove page footers/headers that might be embedded
				chunk = p.cleanSectionContent(chunk)

				allContent.WriteString(chunk)
				allContent.WriteString("\n")

				// Move search forward
				searchStart = absIdx + len(header)
			}
		}

		sections[sectionName] = allContent.String()
	}

	return sections
}

// cleanSectionContent removes page headers/footers from section content
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
// Handles varying whitespace between description and amount
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
	// The asterisk indicates a break in serial sequence
	checkPattern := regexp.MustCompile(`(\d{2}/\d{2})\s+(\d+)\*?\s+([\d,]+\.\d{2})`)

	matches := checkPattern.FindAllStringSubmatch(section, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			checkNum := match[2]

			// Skip duplicates (same check appears in both columns sometimes)
			if seen[checkNum] {
				continue
			}
			seen[checkNum] = true

			txn := ParsedTransaction{
				PostingDate:     p.formatDate(match[1]),
				Description:     "Check #" + checkNum,
				Amount:          -parseAmount(match[3]), // Negative for checks (money out)
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
// These entries can span multiple lines:
// 12/02  DEBIT POS AP, AUT 120225 DDA PURCHASE AP                    1,525.50
//
//	JETRO CASH CARRY                 BROOKLYN             * NY
//	4085404039877380
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
			// Save previous transaction if exists
			if currentTxn != nil {
				p.finalizeTransaction(currentTxn)
				transactions = append(transactions, *currentTxn)
			}

			currentTxn = &ParsedTransaction{
				PostingDate:     p.formatDate(match[1]),
				Description:     strings.TrimSpace(match[2]),
				Amount:          -parseAmount(match[3]), // Negative for payments (money out)
				TransactionType: "debit",
			}
		} else if currentTxn != nil {
			// Check for continuation line (indented, no date)
			if match := continuationLinePattern.FindStringSubmatch(line); len(match) > 1 {
				// Append to description, but skip card numbers (16 digits)
				continuation := strings.TrimSpace(match[1])
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

// finalizeTransaction sets category and vendor hint for a transaction
func (p *TDBankParser) finalizeTransaction(txn *ParsedTransaction) {
	txn.Category = categorizeTransaction(txn.TransactionType, txn.Description, txn.Amount)
	txn.VendorHint = extractVendorHint(txn.Description)
}

// formatDate converts MM/DD to YYYY-MM-DD using statement year/month
func (p *TDBankParser) formatDate(mmdd string) string {
	parts := strings.Split(mmdd, "/")
	if len(parts) != 2 {
		return ""
	}

	month, _ := strconv.Atoi(parts[0])
	day, _ := strconv.Atoi(parts[1])

	// Handle year boundary
	year := p.statementYear
	if p.statementMonth == 12 && month == 1 {
		year++
	} else if p.statementMonth == 1 && month == 12 {
		year--
	}

	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}

// parseAmount converts string like "1,234.56" or "-1,234.56" to float64
func parseAmount(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "$", "")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// monthNameToNumber converts month name to number
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

// Vendor extraction patterns - map known bank descriptions to clean vendor names
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

// extractVendorHint tries to identify the vendor from the description
func extractVendorHint(description string) string {
	// Check known vendor patterns
	for vendor, pattern := range vendorPatterns {
		if pattern.MatchString(description) {
			return vendor
		}
	}

	// Try to extract business name from "BUSINESS NAME CITY * STATE" pattern
	// This is common in debit card transactions
	cityStatePattern := regexp.MustCompile(`([A-Z][A-Z0-9\s&']+?)\s+(?:[A-Z]+\s+)?\*\s*[A-Z]{2}`)
	if match := cityStatePattern.FindStringSubmatch(description); len(match) > 1 {
		vendor := strings.TrimSpace(match[1])
		// Clean up common prefixes
		vendor = regexp.MustCompile(`^(DBCRD|DEBIT|POS|AP|AUT|VISA|DDA|PUR)\s+`).ReplaceAllString(vendor, "")
		if len(vendor) > 3 {
			return vendor
		}
	}

	return ""
}

// categorizeTransaction determines the category based on type and description
func categorizeTransaction(txnType, description string, amount float64) string {
	descUpper := strings.ToUpper(description)

	if amount > 0 {
		// Credits/Deposits
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

	// Debits/Withdrawals
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
