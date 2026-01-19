package models

import (
	"strings"
	"time"
)

// VendorCategories is the list of available vendor categories
var VendorCategories = []string{
	"Beverages",
	"Delivery",
	"Donation",
	"Equipment",
	"Finished Food",
	"Food",
	"Insurance",
	"Licenses",
	"Loan",
	"Marketing",
	"Meat",
	"Paper",
	"Payroll",
	"Rent",
	"Repairs/Reno",
	"Seafood",
	"Services",
	"Supplies",
	"Taxes",
	"Utilities",
}

type Vendor struct {
	ID          int64
	Name        string
	Category    string // comma-separated list of categories
	Description string
	CreatedAt   time.Time
}

// HasCategory checks if the vendor has a specific category
func (v Vendor) HasCategory(cat string) bool {
	if v.Category == "" {
		return false
	}
	for _, c := range strings.Split(v.Category, ",") {
		if c == cat {
			return true
		}
	}
	return false
}

// CategoryList returns categories as a slice
func (v Vendor) CategoryList() []string {
	if v.Category == "" {
		return nil
	}
	return strings.Split(v.Category, ",")
}

type Employee struct {
	ID            int64
	Name          string
	HourlyRate    float64
	PaymentMethod string // "cash" or "check"
	Active        bool
	CreatedAt     time.Time
}

type DailySale struct {
	ID          int64
	Date        string // YYYY-MM-DD
	Shift       string // "breakfast", "lunch", "dinner"
	NetSales    float64
	Taxes       float64
	CreditCard  float64
	CashReceipt float64
	CashOnHand  float64
	Notes       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Variance calculates Cash On Hand - Expected Cash
// Expected = Net Sales + Taxes - Credit Card
func (s DailySale) Variance() float64 {
	expected := s.NetSales + s.Taxes - s.CreditCard
	return s.CashOnHand - expected
}

// ExpectedCash returns the expected cash amount
func (s DailySale) ExpectedCash() float64 {
	return s.NetSales + s.Taxes - s.CreditCard
}

// DeliverySales represents delivery platform sales for a single day
type DeliverySales struct {
	ID               int64
	Date             string // YYYY-MM-DD
	GrubhubSubtotal  float64
	GrubhubNet       float64
	DoordashSubtotal float64
	DoordashNet      float64
	UberEatsEarnings float64
	UberEatsPayout   float64
	Notes            string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// TotalSubtotal returns the sum of all delivery subtotals/earnings
func (d DeliverySales) TotalSubtotal() float64 {
	return d.GrubhubSubtotal + d.DoordashSubtotal + d.UberEatsEarnings
}

// TotalNet returns the sum of all delivery net amounts/payouts
func (d DeliverySales) TotalNet() float64 {
	return d.GrubhubNet + d.DoordashNet + d.UberEatsPayout
}

// HasData returns true if any delivery data has been entered
func (d DeliverySales) HasData() bool {
	return d.GrubhubSubtotal > 0 || d.GrubhubNet > 0 ||
		d.DoordashSubtotal > 0 || d.DoordashNet > 0 ||
		d.UberEatsEarnings > 0 || d.UberEatsPayout > 0
}

type Expense struct {
	ID            int64
	Date          string // YYYY-MM-DD
	VendorID      int64
	VendorName    string // populated by JOIN
	Amount        float64
	InvoiceNumber string
	Status        string // "paid" or "not_paid"
	PaymentType   string // "cash", "check", "debit", "credit"
	CheckNumber   string
	DateOpened    string // YYYY-MM-DD or empty
	DueDate       string // YYYY-MM-DD or empty
	DatePaid      string // YYYY-MM-DD or empty
	Notes         string
	ReceiptPath   string // stored filename in filestore
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// PayrollWeek represents a payroll period (Monday-Sunday)
type PayrollWeek struct {
	ID          int64
	PeriodStart string // YYYY-MM-DD (Monday)
	PeriodEnd   string // YYYY-MM-DD (Sunday)
	CreatedAt   time.Time
}

type Payroll struct {
	ID            int64
	WeekID        int64 // references payroll_weeks.id
	EmployeeID    int64
	EmployeeName  string // populated by JOIN
	PeriodStart   string // YYYY-MM-DD - populated by JOIN with payroll_weeks
	PeriodEnd     string // YYYY-MM-DD - populated by JOIN with payroll_weeks
	TotalHours    float64
	HourlyRate    float64
	PaymentMethod string // "cash" or "check"
	CheckNumber   string
	Status        string // "paid" or "not_paid"
	DatePaid      string // YYYY-MM-DD or empty
	Notes         string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// TotalPay calculates hours * rate
func (p Payroll) TotalPay() float64 {
	return p.TotalHours * p.HourlyRate
}

// WeeklyPayrollEntry combines an employee with their payroll for a specific week
type WeeklyPayrollEntry struct {
	Employee Employee
	Payroll  *Payroll // nil if no payroll entry exists for this employee this week
}

// PayrollWeekSummary represents a summary of a payroll week
type PayrollWeekSummary struct {
	WeekID             int64
	PeriodStart        string
	PeriodEnd          string
	PeriodStartDisplay string
	PeriodEndDisplay   string
	EmployeeCount      int
	TotalHours         float64
	TotalPay           float64
	PaidCount          int
}

// AllPaid returns true if all employees are paid for this week
func (p PayrollWeekSummary) AllPaid() bool {
	return p.PaidCount == p.EmployeeCount
}

type Session struct {
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Dashboard aggregates
type DashboardData struct {
	TodaySalesTotal     float64
	TodayExpensesTotal  float64
	UnpaidExpensesTotal float64
	UnpaidExpensesCount int
	RecentSalesGrouped  []DateGroup
	RecentSalesTotal    float64
	UnpaidExpenses      []Expense
}

// Filter structs for list queries
type ExpenseFilter struct {
	StartDate  string
	EndDate    string
	Status     string
	VendorID   int64
	Categories []string // filter by vendor categories (multi-select)
}

// HasCategory checks if a category is in the filter
func (f ExpenseFilter) HasCategory(cat string) bool {
	for _, c := range f.Categories {
		if c == cat {
			return true
		}
	}
	return false
}

type PayrollFilter struct {
	StartDate  string
	EndDate    string
	Status     string
	EmployeeID int64
}

type SalesFilter struct {
	StartDate string
	EndDate   string
}

// DateGroup represents a single date's sales (for collapsing by date)
type DateGroup struct {
	Date      string         // Display date (MM-DD-YYYY)
	RawDate   string         // Raw date for sorting (YYYY-MM-DD)
	Total     float64        // Sum of NetSales for this date
	Sales     []DailySale    // Individual shift entries
	Delivery  *DeliverySales // Delivery sales for this date (nil if none)
	Collapsed bool           // Whether this date row is collapsed
}

// SalesGroup represents a group of sales with a label and total
type SalesGroup struct {
	Label      string         // "Today", "This Week", "January 2025", "2024"
	Total      float64        // Sum of NetSales for the group
	Sales      []DailySale    // Individual entries (used for Today)
	DateGroups []DateGroup    // Grouped by date (used for non-Today sections)
	Delivery   *DeliverySales // Delivery data for Today section
	Collapsed  bool           // Default collapsed state for UI
}

// GroupedSalesData organizes sales into time-based groups
type GroupedSalesData struct {
	Today      *SalesGroup  // Today's sales (shows individual shifts)
	ThisMonth  *SalesGroup  // Current month excluding today (grouped by date)
	PrevMonths []SalesGroup // Previous months in current year (grouped by date)
	PrevYears  []SalesGroup // Previous years (grouped by date)
}

// BankReconciliation represents a bank statement reconciliation
type BankReconciliation struct {
	ID                   int64
	StatementDate        string // YYYY-MM-DD
	StatementDateDisplay string // formatted for display
	StartingBalance      float64
	EndingBalance        float64
	Status               string // pending, parsing, parsed, reconciling, completed
	FilePath             string // stored filename in filestore
	AccountLastFour      string
	ParseJobID           *int64
	ParsedAt             *time.Time
	ReconciledAt         *time.Time
	Notes                string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	// Statement summary totals (from PDF)
	ElectronicDeposits float64
	ElectronicPayments float64
	ChecksPaid         float64
	ServiceFees        float64
}

// BankTransaction represents a single transaction from a bank statement
type BankTransaction struct {
	ID               int64
	ReconciliationID int64
	PostingDate      string // YYYY-MM-DD
	Description      string
	Amount           float64 // negative for debits, positive for credits
	TransactionType  string  // deposit, check, debit, ach, fee, transfer
	Category         string  // income_cards, income_delivery, expense, fee, transfer
	CheckNumber      string
	VendorHint       string // extracted vendor name
	ReferenceNumber  string
	MatchedExpenseID *int64
	MatchStatus      string // unmatched, matched, ignored, created
	MatchConfidence  string // auto_exact, auto_fuzzy, manual
	MatchedAt        *time.Time
	Notes            string
	CreatedAt        time.Time

	// Joined fields for display
	MatchedExpenseVendor string
	MatchedExpenseDate   string
}

// MonthOption represents a month available for reconciliation
type MonthOption struct {
	Value    string // YYYY-MM format
	Label    string // "January 2025" format
	Selected bool
}

// Job represents a background job in the queue
type Job struct {
	ID          int64
	JobType     string
	Payload     string // JSON payload
	Status      string // pending, running, completed, failed
	Progress    int    // 0-100
	Result      string // JSON result or error message
	Attempts    int
	MaxAttempts int
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
}
