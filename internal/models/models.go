package models

import "time"

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
	Category    string
	Description string
	CreatedAt   time.Time
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
	ID         int64
	Date       string // YYYY-MM-DD
	Shift      string // "breakfast", "lunch", "dinner"
	NetSales   float64
	Taxes      float64
	CreditCard float64
	CashReceipt float64
	CashOnHand float64
	Notes      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
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
	DatePaid      string // YYYY-MM-DD or empty
	Notes         string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Payroll struct {
	ID            int64
	EmployeeID    int64
	EmployeeName  string // populated by JOIN
	PeriodStart   string // YYYY-MM-DD
	PeriodEnd     string // YYYY-MM-DD
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

type Session struct {
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Dashboard aggregates
type DashboardData struct {
	UnpaidExpensesTotal  float64
	UnpaidExpensesCount  int
	UnpaidPayrollTotal   float64
	UnpaidPayrollCount   int
	RecentSalesGrouped   []DateGroup
	RecentSalesTotal     float64
	UnpaidExpenses       []Expense
	UnpaidPayroll        []Payroll
}

// Filter structs for list queries
type ExpenseFilter struct {
	StartDate string
	EndDate   string
	Status    string
	VendorID  int64
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
	Date      string          // Display date (MM-DD-YYYY)
	RawDate   string          // Raw date for sorting (YYYY-MM-DD)
	Total     float64         // Sum of NetSales for this date
	Sales     []DailySale     // Individual shift entries
	Delivery  *DeliverySales  // Delivery sales for this date (nil if none)
	Collapsed bool            // Whether this date row is collapsed
}

// SalesGroup represents a group of sales with a label and total
type SalesGroup struct {
	Label      string          // "Today", "This Week", "January 2025", "2024"
	Total      float64         // Sum of NetSales for the group
	Sales      []DailySale     // Individual entries (used for Today)
	DateGroups []DateGroup     // Grouped by date (used for non-Today sections)
	Delivery   *DeliverySales  // Delivery data for Today section
	Collapsed  bool            // Default collapsed state for UI
}

// GroupedSalesData organizes sales into time-based groups
type GroupedSalesData struct {
	Today      *SalesGroup  // Today's sales (shows individual shifts)
	ThisMonth  *SalesGroup  // Current month excluding today (grouped by date)
	PrevMonths []SalesGroup // Previous months in current year (grouped by date)
	PrevYears  []SalesGroup // Previous years (grouped by date)
}
