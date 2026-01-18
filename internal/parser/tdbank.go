package parser

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// ParsedStatement represents a fully parsed bank statement
type ParsedStatement struct {
	AccountLastFour  string
	BeginningBalance float64
	EndingBalance    float64
	Transactions     []ParsedTransaction
}

// ParsedTransaction represents a single parsed transaction
type ParsedTransaction struct {
	PostingDate     string // YYYY-MM-DD
	Description     string
	Amount          float64 // negative for debits, positive for credits
	TransactionType string  // deposit, check, debit, ach, fee, transfer
	Category        string
	CheckNumber     string
	VendorHint      string
	ReferenceNumber string
}

// TDBankParser parses TD Bank PDF statements
type TDBankParser struct {
	statementYear  int
	statementMonth int
}

// NewTDBankParser creates a new TD Bank parser
func NewTDBankParser() *TDBankParser {
	return &TDBankParser{}
}

// Parse extracts data from a TD Bank PDF statement
func (p *TDBankParser) Parse(pdfPath string) (*ParsedStatement, error) {
	// Extract text using pdftotext
	text, err := p.extractText(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("extract text: %w", err)
	}

	// Parse header to get balances and account info
	stmt, err := p.parseHeader(text)
	if err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}

	// Split into sections and parse each
	sections := p.splitSections(text)

	// Parse each section type
	stmt.Transactions = append(stmt.Transactions, p.parseDeposits(sections["deposits"])...)
	stmt.Transactions = append(stmt.Transactions, p.parseCredits(sections["credits"])...)
	stmt.Transactions = append(stmt.Transactions, p.parseChecks(sections["checks"])...)
	stmt.Transactions = append(stmt.Transactions, p.parsePayments(sections["payments"])...)
	stmt.Transactions = append(stmt.Transactions, p.parseWithdrawals(sections["withdrawals"])...)
	stmt.Transactions = append(stmt.Transactions, p.parseFees(sections["fees"])...)

	return stmt, nil
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

// Regex patterns for parsing
var (
	// Account number pattern: "Account Number: XXXXXX1234"
	accountPattern = regexp.MustCompile(`Account\s+Number[:\s]+\w*(\d{4})\b`)

	// Statement period pattern: "Statement Period: Nov 01, 2024 to Nov 30, 2024"
	periodPattern = regexp.MustCompile(`Statement\s+Period[:\s]+(\w+)\s+\d+,?\s+(\d{4})`)

	// Balance patterns
	beginningBalancePattern = regexp.MustCompile(`Beginning\s+Balance\s+\$?([\d,]+\.\d{2})`)
	endingBalancePattern    = regexp.MustCompile(`Ending\s+Balance\s+\$?([\d,]+\.\d{2})`)

	// Transaction line pattern: "12/01 Description here 1,234.56"
	transactionPattern = regexp.MustCompile(`^(\d{2}/\d{2})\s+(.+?)\s+([\d,]+\.\d{2})\s*$`)

	// Check pattern: "12/01 2730 500.00" or with asterisk "12/01 2730* 500.00"
	checkPattern = regexp.MustCompile(`(\d{2}/\d{2})\s+(\d+)\*?\s+([\d,]+\.\d{2})`)

	// Continuation line (indented, no date)
	continuationPattern = regexp.MustCompile(`^\s{6,}(\S.*)$`)
)

// parseHeader extracts account info and balances
func (p *TDBankParser) parseHeader(text string) (*ParsedStatement, error) {
	stmt := &ParsedStatement{}

	// Extract account number (last 4 digits)
	if match := accountPattern.FindStringSubmatch(text); len(match) > 1 {
		stmt.AccountLastFour = match[1]
	}

	// Extract statement period to determine year
	if match := periodPattern.FindStringSubmatch(text); len(match) > 2 {
		year, _ := strconv.Atoi(match[2])
		p.statementYear = year
		// Parse month name
		p.statementMonth = monthNameToNumber(match[1])
	}

	// Extract beginning balance
	if match := beginningBalancePattern.FindStringSubmatch(text); len(match) > 1 {
		stmt.BeginningBalance = parseAmount(match[1])
	}

	// Extract ending balance
	if match := endingBalancePattern.FindStringSubmatch(text); len(match) > 1 {
		stmt.EndingBalance = parseAmount(match[1])
	}

	return stmt, nil
}

// Section markers for TD Bank statements
var sectionMarkers = map[string]struct {
	start string
	end   []string
}{
	"deposits":    {"Electronic Deposits", []string{"Other Credits", "Checks Paid", "Electronic Payments"}},
	"credits":     {"Other Credits", []string{"Checks Paid", "Electronic Payments"}},
	"checks":      {"Checks Paid", []string{"Electronic Payments", "Other Withdrawals"}},
	"payments":    {"Electronic Payments", []string{"Other Withdrawals", "Service Charges"}},
	"withdrawals": {"Other Withdrawals", []string{"Service Charges", "DAILY BALANCE SUMMARY"}},
	"fees":        {"Service Charges", []string{"DAILY BALANCE SUMMARY", "ACCOUNT MESSAGES"}},
}

// splitSections divides the text into named sections
func (p *TDBankParser) splitSections(text string) map[string]string {
	sections := make(map[string]string)

	for name, markers := range sectionMarkers {
		startIdx := strings.Index(text, markers.start)
		if startIdx == -1 {
			continue
		}

		// Find the earliest end marker
		endIdx := len(text)
		for _, endMarker := range markers.end {
			if idx := strings.Index(text[startIdx:], endMarker); idx != -1 {
				absIdx := startIdx + idx
				if absIdx < endIdx {
					endIdx = absIdx
				}
			}
		}

		sections[name] = text[startIdx:endIdx]
	}

	return sections
}

// parseDeposits parses the Electronic Deposits section
func (p *TDBankParser) parseDeposits(section string) []ParsedTransaction {
	if section == "" {
		return nil
	}

	var transactions []ParsedTransaction
	scanner := bufio.NewScanner(strings.NewReader(section))

	for scanner.Scan() {
		line := scanner.Text()
		if match := transactionPattern.FindStringSubmatch(line); len(match) > 3 {
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
	if section == "" {
		return nil
	}

	var transactions []ParsedTransaction
	scanner := bufio.NewScanner(strings.NewReader(section))

	for scanner.Scan() {
		line := scanner.Text()
		if match := transactionPattern.FindStringSubmatch(line); len(match) > 3 {
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

// parseChecks parses the Checks Paid section (table format, possibly 2 columns)
func (p *TDBankParser) parseChecks(section string) []ParsedTransaction {
	if section == "" {
		return nil
	}

	var transactions []ParsedTransaction

	// Find all check matches in the section
	matches := checkPattern.FindAllStringSubmatch(section, -1)
	for _, match := range matches {
		if len(match) > 3 {
			txn := ParsedTransaction{
				PostingDate:     p.formatDate(match[1]),
				Description:     "Check #" + match[2],
				Amount:          -parseAmount(match[3]), // Negative for checks
				TransactionType: "check",
				CheckNumber:     match[2],
				Category:        "expense_check",
			}
			transactions = append(transactions, txn)
		}
	}

	return transactions
}

// parsePayments parses the Electronic Payments section (multi-line entries)
func (p *TDBankParser) parsePayments(section string) []ParsedTransaction {
	if section == "" {
		return nil
	}

	var transactions []ParsedTransaction
	var currentTxn *ParsedTransaction

	scanner := bufio.NewScanner(strings.NewReader(section))
	for scanner.Scan() {
		line := scanner.Text()

		// Check for new transaction line
		if match := transactionPattern.FindStringSubmatch(line); len(match) > 3 {
			// Save previous transaction if exists
			if currentTxn != nil {
				currentTxn.Category = categorizeTransaction(currentTxn.TransactionType, currentTxn.Description, currentTxn.Amount)
				currentTxn.VendorHint = extractVendorHint(currentTxn.Description)
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
			if match := continuationPattern.FindStringSubmatch(line); len(match) > 1 {
				// Append to description
				currentTxn.Description += " " + strings.TrimSpace(match[1])
			}
		}
	}

	// Don't forget the last transaction
	if currentTxn != nil {
		currentTxn.Category = categorizeTransaction(currentTxn.TransactionType, currentTxn.Description, currentTxn.Amount)
		currentTxn.VendorHint = extractVendorHint(currentTxn.Description)
		transactions = append(transactions, *currentTxn)
	}

	return transactions
}

// parseWithdrawals parses the Other Withdrawals section
func (p *TDBankParser) parseWithdrawals(section string) []ParsedTransaction {
	if section == "" {
		return nil
	}

	var transactions []ParsedTransaction
	var currentTxn *ParsedTransaction

	scanner := bufio.NewScanner(strings.NewReader(section))
	for scanner.Scan() {
		line := scanner.Text()

		if match := transactionPattern.FindStringSubmatch(line); len(match) > 3 {
			if currentTxn != nil {
				currentTxn.Category = categorizeTransaction(currentTxn.TransactionType, currentTxn.Description, currentTxn.Amount)
				currentTxn.VendorHint = extractVendorHint(currentTxn.Description)
				transactions = append(transactions, *currentTxn)
			}

			currentTxn = &ParsedTransaction{
				PostingDate:     p.formatDate(match[1]),
				Description:     strings.TrimSpace(match[2]),
				Amount:          -parseAmount(match[3]), // Negative for withdrawals
				TransactionType: "withdrawal",
			}
		} else if currentTxn != nil {
			if match := continuationPattern.FindStringSubmatch(line); len(match) > 1 {
				currentTxn.Description += " " + strings.TrimSpace(match[1])
			}
		}
	}

	if currentTxn != nil {
		currentTxn.Category = categorizeTransaction(currentTxn.TransactionType, currentTxn.Description, currentTxn.Amount)
		currentTxn.VendorHint = extractVendorHint(currentTxn.Description)
		transactions = append(transactions, *currentTxn)
	}

	return transactions
}

// parseFees parses the Service Charges section
func (p *TDBankParser) parseFees(section string) []ParsedTransaction {
	if section == "" {
		return nil
	}

	var transactions []ParsedTransaction
	scanner := bufio.NewScanner(strings.NewReader(section))

	for scanner.Scan() {
		line := scanner.Text()
		if match := transactionPattern.FindStringSubmatch(line); len(match) > 3 {
			txn := ParsedTransaction{
				PostingDate:     p.formatDate(match[1]),
				Description:     strings.TrimSpace(match[2]),
				Amount:          -parseAmount(match[3]), // Negative for fees
				TransactionType: "fee",
				Category:        "fee",
			}
			txn.VendorHint = "Bank Fee"
			transactions = append(transactions, txn)
		}
	}

	return transactions
}

// formatDate converts MM/DD to YYYY-MM-DD using statement year/month
func (p *TDBankParser) formatDate(mmdd string) string {
	parts := strings.Split(mmdd, "/")
	if len(parts) != 2 {
		return ""
	}

	month, _ := strconv.Atoi(parts[0])
	day, _ := strconv.Atoi(parts[1])

	// Handle year boundary: if statement is for December but transaction is January
	year := p.statementYear
	if p.statementMonth == 12 && month == 1 {
		year++
	} else if p.statementMonth == 1 && month == 12 {
		year--
	}

	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}

// parseAmount converts string like "1,234.56" to float64
func parseAmount(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
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

// Vendor extraction patterns
var vendorPatterns = map[string]*regexp.Regexp{
	"Jetro":            regexp.MustCompile(`(?i)JETRO\s*CASH\s*CARRY`),
	"Chef's Choice":    regexp.MustCompile(`(?i)CHEF.?S?\s*CHOICE`),
	"Cogent Waste":     regexp.MustCompile(`(?i)COGENT\s*WASTE`),
	"Con Edison":       regexp.MustCompile(`(?i)CON\s*ED`),
	"National Grid":    regexp.MustCompile(`(?i)NGRID|NATIONAL\s*GRID`),
	"Uber Eats":        regexp.MustCompile(`(?i)UBER\s*(USA|EATS)?`),
	"Grubhub":          regexp.MustCompile(`(?i)GRUBHUB`),
	"DoorDash":         regexp.MustCompile(`(?i)DOORDASH`),
	"Verizon":          regexp.MustCompile(`(?i)VERIZON`),
	"AT&T":             regexp.MustCompile(`(?i)\bATT\b|AT&T`),
	"Sampar's":         regexp.MustCompile(`(?i)SAMPARS?`),
	"Clover":           regexp.MustCompile(`(?i)CLOVER`),
	"Cintas":           regexp.MustCompile(`(?i)CINTAS`),
	"Sysco":            regexp.MustCompile(`(?i)SYSCO`),
	"US Foods":         regexp.MustCompile(`(?i)US\s*FOODS`),
	"Restaurant Depot": regexp.MustCompile(`(?i)RESTAURANT\s*DEPOT`),
}

// extractVendorHint tries to identify the vendor from the description
func extractVendorHint(description string) string {
	for vendor, pattern := range vendorPatterns {
		if pattern.MatchString(description) {
			return vendor
		}
	}

	// Try to extract business name from common patterns
	// Pattern: "BUSINESS NAME CITY * STATE"
	if match := regexp.MustCompile(`([A-Z][A-Z\s&']+?)\s+[A-Z]{2}\s*\*?\s*[A-Z]{2}`).FindStringSubmatch(description); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}

	return ""
}

// categorizeTransaction determines the category based on type and description
func categorizeTransaction(txnType, description string, amount float64) string {
	descUpper := strings.ToUpper(description)

	if amount > 0 {
		// Credits
		if strings.Contains(descUpper, "BANKCARD") || strings.Contains(descUpper, "MTOT DEP") {
			return "income_cards"
		}
		if strings.Contains(descUpper, "UBER") || strings.Contains(descUpper, "GRUBHUB") || strings.Contains(descUpper, "DOORDASH") {
			return "income_delivery"
		}
		if strings.Contains(descUpper, "REFUND") {
			return "refund"
		}
		return "income_other"
	}

	// Debits
	if txnType == "check" {
		return "expense_check"
	}
	if txnType == "fee" {
		return "fee"
	}
	if strings.Contains(descUpper, "TRANSFER") || strings.Contains(descUpper, "XFER") {
		return "transfer"
	}
	return "expense"
}
