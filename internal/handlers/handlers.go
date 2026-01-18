package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"homebooks/internal/auth"
	"homebooks/internal/database"
	"homebooks/internal/filestore"
	"homebooks/internal/logger"
	"homebooks/internal/models"
)

type Handler struct {
	db    *database.DB
	auth  *auth.Auth
	tmpl  *template.Template
	files *filestore.Store
}

func New(db *database.DB, a *auth.Auth, tmpl *template.Template, files *filestore.Store) *Handler {
	return &Handler{
		db:    db,
		auth:  a,
		tmpl:  tmpl,
		files: files,
	}
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, name string, data map[string]interface{}) {
	err := h.tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		l := logger.FromContext(r.Context())
		l.Error("template_render_error", "template", name, "error", err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Login handlers
func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := h.auth.GetSessionFromRequest(r)
	if token != "" && h.auth.ValidateSession(ctx, token) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	h.render(w, r, "login.html", map[string]interface{}{"Error": ""})
}

func (h *Handler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	password := r.FormValue("password")

	if !h.auth.CheckPassword(ctx, password) {
		h.render(w, r, "login.html", map[string]interface{}{"Error": "Invalid password"})
		return
	}

	token, err := h.auth.CreateSession(ctx)
	if err != nil {
		h.render(w, r, "login.html", map[string]interface{}{"Error": "Failed to create session"})
		return
	}

	h.auth.SetSessionCookie(w, token)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := h.auth.GetSessionFromRequest(r)
	if token != "" {
		h.auth.DeleteSession(ctx, token)
	}
	h.auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

// Dashboard
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	unpaidExpenses, expenseTotal, _ := h.db.ListUnpaidExpenses()
	recentSalesGrouped, recentSalesTotal, _ := h.db.ListRecentSalesGrouped(7)
	todaySalesTotal, _ := h.db.GetTodaySalesTotal()
	todayExpensesTotal, _ := h.db.GetTodayExpensesTotal()

	data := models.DashboardData{
		TodaySalesTotal:      todaySalesTotal,
		TodayExpensesTotal:   todayExpensesTotal,
		UnpaidExpensesTotal:  expenseTotal,
		UnpaidExpensesCount:  len(unpaidExpenses),
		RecentSalesGrouped:   recentSalesGrouped,
		RecentSalesTotal:     recentSalesTotal,
		UnpaidExpenses:       unpaidExpenses,
	}

	h.render(w, r, "dashboard.html", map[string]interface{}{
		"Title":  "Dashboard",
		"Active": "dashboard",
		"Data":   data,
	})
}

// Vendors handlers
func (h *Handler) VendorsList(w http.ResponseWriter, r *http.Request) {
	vendors, err := h.db.ListVendors()
	if err != nil {
		logger.FromContext(r.Context()).Error("vendor_list_error", "error", err.Error())
	}
	h.render(w, r, "vendors_list.html", map[string]interface{}{
		"Title":   "Vendors",
		"Active":  "vendors",
		"Vendors": vendors,
	})
}

func (h *Handler) VendorsShow(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	vendor, err := h.db.GetVendor(id)
	if err != nil {
		http.Redirect(w, r, "/vendors", http.StatusFound)
		return
	}
	expenses, total, _ := h.db.ListExpenses(models.ExpenseFilter{VendorID: id})
	h.render(w, r, "vendors_show.html", map[string]interface{}{
		"Title":    vendor.Name,
		"Active":   "vendors",
		"Vendor":   vendor,
		"Expenses": expenses,
		"Total":    total,
	})
}

func (h *Handler) VendorsNew(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "vendors_form.html", map[string]interface{}{
		"Title":      "New Vendor",
		"Active":     "vendors",
		"Vendor":     models.Vendor{},
		"Categories": models.VendorCategories,
	})
}

func (h *Handler) VendorsCreate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := r.FormValue("name")
	categories := r.Form["category"]
	category := strings.Join(categories, ",")
	description := r.FormValue("description")

	if name == "" {
		h.render(w, r, "vendors_form.html", map[string]interface{}{
			"Title":      "New Vendor",
			"Active":     "vendors",
			"Vendor":     models.Vendor{Name: name, Category: category, Description: description},
			"Categories": models.VendorCategories,
			"Error":      "Name is required",
		})
		return
	}

	_, err := h.db.CreateVendor(name, category, description)
	if err != nil {
		h.render(w, r, "vendors_form.html", map[string]interface{}{
			"Title":      "New Vendor",
			"Active":     "vendors",
			"Vendor":     models.Vendor{Name: name, Category: category, Description: description},
			"Categories": models.VendorCategories,
			"Error":      "Vendor already exists or error occurred",
		})
		return
	}

	http.Redirect(w, r, "/vendors", http.StatusFound)
}

func (h *Handler) VendorsEdit(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	vendor, err := h.db.GetVendor(id)
	if err != nil {
		http.Redirect(w, r, "/vendors", http.StatusFound)
		return
	}
	h.render(w, r, "vendors_form.html", map[string]interface{}{
		"Title":      "Edit Vendor",
		"Active":     "vendors",
		"Vendor":     vendor,
		"Categories": models.VendorCategories,
	})
}

func (h *Handler) VendorsUpdate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	name := r.FormValue("name")
	categories := r.Form["category"]
	category := strings.Join(categories, ",")
	description := r.FormValue("description")

	if name == "" {
		h.render(w, r, "vendors_form.html", map[string]interface{}{
			"Title":      "Edit Vendor",
			"Active":     "vendors",
			"Vendor":     models.Vendor{ID: id, Name: name, Category: category, Description: description},
			"Categories": models.VendorCategories,
			"Error":      "Name is required",
		})
		return
	}

	err := h.db.UpdateVendor(id, name, category, description)
	if err != nil {
		h.render(w, r, "vendors_form.html", map[string]interface{}{
			"Title":      "Edit Vendor",
			"Active":     "vendors",
			"Vendor":     models.Vendor{ID: id, Name: name, Category: category, Description: description},
			"Categories": models.VendorCategories,
			"Error":      "Error updating vendor",
		})
		return
	}

	http.Redirect(w, r, "/vendors", http.StatusFound)
}

func (h *Handler) VendorsDelete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	h.db.DeleteVendor(id)
	http.Redirect(w, r, "/vendors", http.StatusFound)
}

// Employees handlers
func (h *Handler) EmployeesList(w http.ResponseWriter, r *http.Request) {
	employees, _ := h.db.ListEmployees(false)
	h.render(w, r, "employees_list.html", map[string]interface{}{
		"Title":     "Employees",
		"Active":    "employees",
		"Employees": employees,
	})
}

func (h *Handler) EmployeesCreate(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	hourlyRate, _ := strconv.ParseFloat(r.FormValue("hourly_rate"), 64)
	paymentMethod := r.FormValue("payment_method")

	if name == "" || hourlyRate <= 0 {
		employees, _ := h.db.ListEmployees(false)
		h.render(w, r, "employees_list.html", map[string]interface{}{
			"Title":     "Employees",
			"Active":    "employees",
			"Employees": employees,
			"Error":     "Name and valid hourly rate are required",
		})
		return
	}

	_, err := h.db.CreateEmployee(name, hourlyRate, paymentMethod)
	if err != nil {
		employees, _ := h.db.ListEmployees(false)
		h.render(w, r, "employees_list.html", map[string]interface{}{
			"Title":     "Employees",
			"Active":    "employees",
			"Employees": employees,
			"Error":     "Error creating employee",
		})
		return
	}

	http.Redirect(w, r, "/employees", http.StatusFound)
}

func (h *Handler) EmployeesDeactivate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	h.db.DeactivateEmployee(id)
	http.Redirect(w, r, "/employees", http.StatusFound)
}

func (h *Handler) EmployeesReactivate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	h.db.ReactivateEmployee(id)
	http.Redirect(w, r, "/employees", http.StatusFound)
}

// Sales handlers
func (h *Handler) SalesList(w http.ResponseWriter, r *http.Request) {
	grouped, _ := h.db.ListSalesGrouped()
	h.render(w, r, "sales_list.html", map[string]any{
		"Title":     "Daily Sales",
		"Active":    "sales",
		"Grouped":   grouped,
		"TodayDate": time.Now().Format("2006-01-02"),
	})
}

func (h *Handler) SalesNew(w http.ResponseWriter, r *http.Request) {
	sale := models.DailySale{Date: time.Now().Format("2006-01-02")}
	h.render(w, r, "sales_form.html", map[string]interface{}{
		"Title":  "New Sale",
		"Active": "sales",
		"Sale":   sale,
	})
}

func (h *Handler) SalesCreate(w http.ResponseWriter, r *http.Request) {
	sale := models.DailySale{
		Date:  r.FormValue("date"),
		Shift: r.FormValue("shift"),
		Notes: r.FormValue("notes"),
	}
	sale.NetSales, _ = strconv.ParseFloat(r.FormValue("net_sales"), 64)
	sale.Taxes, _ = strconv.ParseFloat(r.FormValue("taxes"), 64)
	sale.CreditCard, _ = strconv.ParseFloat(r.FormValue("credit_card"), 64)
	sale.CashReceipt, _ = strconv.ParseFloat(r.FormValue("cash_receipt"), 64)
	sale.CashOnHand, _ = strconv.ParseFloat(r.FormValue("cash_on_hand"), 64)

	_, err := h.db.UpsertSale(sale)
	if err != nil {
		h.render(w, r, "sales_form.html", map[string]interface{}{
			"Title":  "New Sale",
			"Active": "sales",
			"Sale":   sale,
			"Error":  err.Error(),
		})
		return
	}
	http.Redirect(w, r, "/sales", http.StatusFound)
}

func (h *Handler) SalesEdit(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	sale, err := h.db.GetSale(id)
	if err != nil {
		http.Redirect(w, r, "/sales", http.StatusFound)
		return
	}
	h.render(w, r, "sales_form.html", map[string]interface{}{
		"Title":  "Edit Sale",
		"Active": "sales",
		"Sale":   sale,
	})
}

func (h *Handler) SalesUpdate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	sale := models.DailySale{
		ID:    id,
		Date:  r.FormValue("date"),
		Shift: r.FormValue("shift"),
		Notes: r.FormValue("notes"),
	}
	sale.NetSales, _ = strconv.ParseFloat(r.FormValue("net_sales"), 64)
	sale.Taxes, _ = strconv.ParseFloat(r.FormValue("taxes"), 64)
	sale.CreditCard, _ = strconv.ParseFloat(r.FormValue("credit_card"), 64)
	sale.CashReceipt, _ = strconv.ParseFloat(r.FormValue("cash_receipt"), 64)
	sale.CashOnHand, _ = strconv.ParseFloat(r.FormValue("cash_on_hand"), 64)

	err := h.db.UpdateSale(sale)
	if err != nil {
		h.render(w, r, "sales_form.html", map[string]interface{}{
			"Title":  "Edit Sale",
			"Active": "sales",
			"Sale":   sale,
			"Error":  err.Error(),
		})
		return
	}
	http.Redirect(w, r, "/sales", http.StatusFound)
}

func (h *Handler) SalesDelete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	h.db.DeleteSale(id)
	http.Redirect(w, r, "/sales", http.StatusFound)
}

// SalesShiftsAPI returns existing shifts for a given date as JSON
func (h *Handler) SalesShiftsAPI(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	shifts, err := h.db.GetShiftsForDate(date)
	if err != nil {
		shifts = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{"shifts": shifts})
}

// Delivery Sales handlers
func (h *Handler) DeliveryNew(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	delivery := models.DeliverySales{Date: date}
	h.render(w, r, "delivery_form.html", map[string]any{
		"Title":    "Add Delivery Sales",
		"Active":   "sales",
		"Delivery": delivery,
	})
}

func (h *Handler) DeliveryEdit(w http.ResponseWriter, r *http.Request) {
	date := r.PathValue("date")
	delivery, err := h.db.GetDeliverySalesForDate(date)
	if err != nil {
		http.Error(w, "Error fetching delivery sales", http.StatusInternalServerError)
		return
	}
	if delivery == nil {
		delivery = &models.DeliverySales{Date: date}
	}
	h.render(w, r, "delivery_form.html", map[string]any{
		"Title":    "Edit Delivery Sales",
		"Active":   "sales",
		"Delivery": *delivery,
	})
}

func (h *Handler) DeliverySave(w http.ResponseWriter, r *http.Request) {
	grubhubSubtotal, _ := strconv.ParseFloat(r.FormValue("grubhub_subtotal"), 64)
	grubhubNet, _ := strconv.ParseFloat(r.FormValue("grubhub_net"), 64)
	doordashSubtotal, _ := strconv.ParseFloat(r.FormValue("doordash_subtotal"), 64)
	doordashNet, _ := strconv.ParseFloat(r.FormValue("doordash_net"), 64)
	ubereatsEarnings, _ := strconv.ParseFloat(r.FormValue("ubereats_earnings"), 64)
	ubereatsPayout, _ := strconv.ParseFloat(r.FormValue("ubereats_payout"), 64)

	delivery := models.DeliverySales{
		Date:             r.FormValue("date"),
		GrubhubSubtotal:  grubhubSubtotal,
		GrubhubNet:       grubhubNet,
		DoordashSubtotal: doordashSubtotal,
		DoordashNet:      doordashNet,
		UberEatsEarnings: ubereatsEarnings,
		UberEatsPayout:   ubereatsPayout,
		Notes:            r.FormValue("notes"),
	}

	err := h.db.UpsertDeliverySales(delivery)
	if err != nil {
		h.render(w, r, "delivery_form.html", map[string]any{
			"Title":    "Edit Delivery Sales",
			"Active":   "sales",
			"Delivery": delivery,
			"Error":    err.Error(),
		})
		return
	}

	http.Redirect(w, r, "/sales", http.StatusFound)
}

// Expenses handlers
func (h *Handler) ExpensesList(w http.ResponseWriter, r *http.Request) {
	vendorID, _ := strconv.ParseInt(r.URL.Query().Get("vendor_id"), 10, 64)
	filter := models.ExpenseFilter{
		StartDate:  r.URL.Query().Get("start_date"),
		EndDate:    r.URL.Query().Get("end_date"),
		Status:     r.URL.Query().Get("status"),
		VendorID:   vendorID,
		Categories: r.URL.Query()["category"],
	}
	expenses, total, _ := h.db.ListExpenses(filter)
	vendors, _ := h.db.ListVendors()
	h.render(w, r, "expenses_list.html", map[string]interface{}{
		"Title":      "Expenses",
		"Active":     "expenses",
		"Expenses":   expenses,
		"Total":      total,
		"Vendors":    vendors,
		"Filter":     filter,
		"Categories": models.VendorCategories,
	})
}

func (h *Handler) ExpensesNew(w http.ResponseWriter, r *http.Request) {
	vendors, _ := h.db.ListVendors()
	lastCheck, _ := h.db.GetLastExpenseCheckNumber()
	h.render(w, r, "expenses_form.html", map[string]interface{}{
		"Title":           "New Expense",
		"Active":          "expenses",
		"Expense":         models.Expense{Date: time.Now().Format("2006-01-02")},
		"Vendors":         vendors,
		"LastCheckNumber": lastCheck,
	})
}

func (h *Handler) ExpensesCreate(w http.ResponseWriter, r *http.Request) {
	vendorID, _ := strconv.ParseInt(r.FormValue("vendor_id"), 10, 64)
	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)

	expense := models.Expense{
		Date:          r.FormValue("date"),
		VendorID:      vendorID,
		Amount:        amount,
		InvoiceNumber: r.FormValue("invoice_number"),
		Status:        r.FormValue("status"),
		PaymentType:   r.FormValue("payment_type"),
		CheckNumber:   r.FormValue("check_number"),
		DateOpened:    r.FormValue("date_opened"),
		DueDate:       r.FormValue("due_date"),
		DatePaid:      r.FormValue("date_paid"),
		Notes:         r.FormValue("notes"),
	}

	_, err := h.db.CreateExpense(expense)
	if err != nil {
		vendors, _ := h.db.ListVendors()
		lastCheck, _ := h.db.GetLastExpenseCheckNumber()
		h.render(w, r, "expenses_form.html", map[string]interface{}{
			"Title":           "New Expense",
			"Active":          "expenses",
			"Expense":         expense,
			"Vendors":         vendors,
			"LastCheckNumber": lastCheck,
			"Error":           err.Error(),
		})
		return
	}
	http.Redirect(w, r, "/expenses", http.StatusFound)
}

func (h *Handler) ExpensesEdit(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	expense, err := h.db.GetExpense(id)
	if err != nil {
		http.Redirect(w, r, "/expenses", http.StatusFound)
		return
	}
	vendors, _ := h.db.ListVendors()
	lastCheck, _ := h.db.GetLastExpenseCheckNumber()
	h.render(w, r, "expenses_form.html", map[string]interface{}{
		"Title":           "Edit Expense",
		"Active":          "expenses",
		"Expense":         expense,
		"Vendors":         vendors,
		"LastCheckNumber": lastCheck,
	})
}

func (h *Handler) ExpensesUpdate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	vendorID, _ := strconv.ParseInt(r.FormValue("vendor_id"), 10, 64)
	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)

	expense := models.Expense{
		ID:            id,
		Date:          r.FormValue("date"),
		VendorID:      vendorID,
		Amount:        amount,
		InvoiceNumber: r.FormValue("invoice_number"),
		Status:        r.FormValue("status"),
		PaymentType:   r.FormValue("payment_type"),
		CheckNumber:   r.FormValue("check_number"),
		DateOpened:    r.FormValue("date_opened"),
		DueDate:       r.FormValue("due_date"),
		DatePaid:      r.FormValue("date_paid"),
		Notes:         r.FormValue("notes"),
	}

	err := h.db.UpdateExpense(expense)
	if err != nil {
		vendors, _ := h.db.ListVendors()
		lastCheck, _ := h.db.GetLastExpenseCheckNumber()
		h.render(w, r, "expenses_form.html", map[string]interface{}{
			"Title":           "Edit Expense",
			"Active":          "expenses",
			"Expense":         expense,
			"Vendors":         vendors,
			"LastCheckNumber": lastCheck,
			"Error":           err.Error(),
		})
		return
	}
	http.Redirect(w, r, "/expenses", http.StatusFound)
}

func (h *Handler) ExpensesPayForm(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	expense, err := h.db.GetExpense(id)
	if err != nil {
		http.Redirect(w, r, "/expenses", http.StatusFound)
		return
	}
	lastCheck, _ := h.db.GetLastExpenseCheckNumber()
	h.render(w, r, "expenses_pay.html", map[string]interface{}{
		"Title":           "Mark Expense Paid",
		"Active":          "expenses",
		"Expense":         expense,
		"LastCheckNumber": lastCheck,
	})
}

func (h *Handler) ExpensesPay(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	paymentType := r.FormValue("payment_type")
	checkNumber := r.FormValue("check_number")
	h.db.MarkExpensePaid(id, paymentType, checkNumber)
	http.Redirect(w, r, "/expenses", http.StatusFound)
}

func (h *Handler) ExpensesDelete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	h.db.DeleteExpense(id)
	http.Redirect(w, r, "/expenses", http.StatusFound)
}

// Payroll handlers

// getWeekBounds calculates the Monday and Sunday for a given date
func getWeekBounds(date time.Time) (string, string) {
	// Find Monday of the week
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	monday := date.AddDate(0, 0, -(weekday - 1))
	sunday := monday.AddDate(0, 0, 6)
	return monday.Format("2006-01-02"), sunday.Format("2006-01-02")
}

func (h *Handler) PayrollList(w http.ResponseWriter, r *http.Request) {
	weeks, err := h.db.ListPayrollWeeks(50)
	if err != nil {
		logger.FromContext(r.Context()).Error("payroll_weeks_error", "error", err.Error())
	}

	h.render(w, r, "payroll_list.html", map[string]any{
		"Title":  "Payroll",
		"Active": "payroll",
		"Weeks":  weeks,
	})
}

func (h *Handler) PayrollWeekNew(w http.ResponseWriter, r *http.Request) {
	// Default to current week
	weekStart, weekEnd := getWeekBounds(time.Now())

	entries, total, _ := h.db.GetWeeklyPayroll(weekStart, weekEnd)
	lastCheck, _ := h.db.GetLastPayrollCheckNumber()

	// Format dates for display
	weekStartDate, _ := time.Parse("2006-01-02", weekStart)
	weekEndDate, _ := time.Parse("2006-01-02", weekEnd)
	weekStartDisplay := weekStartDate.Format("Jan 2")
	weekEndDisplay := weekEndDate.Format("Jan 2, 2006")

	h.render(w, r, "payroll_week_edit.html", map[string]any{
		"Title":           "New Payroll Week",
		"Active":          "payroll",
		"Entries":         entries,
		"Total":           total,
		"WeekStart":       weekStart,
		"WeekEnd":         weekEnd,
		"WeekDisplay":     weekStartDisplay + " - " + weekEndDisplay,
		"LastCheckNumber": lastCheck,
	})
}

func (h *Handler) PayrollWeekEdit(w http.ResponseWriter, r *http.Request) {
	weekIDStr := r.PathValue("id")
	weekID, err := strconv.ParseInt(weekIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid week ID", http.StatusBadRequest)
		return
	}

	// Get the week details
	week, err := h.db.GetPayrollWeek(weekID)
	if err != nil {
		http.Error(w, "Week not found", http.StatusNotFound)
		return
	}

	entries, total, _ := h.db.GetWeeklyPayrollByWeekID(weekID)
	lastCheck, _ := h.db.GetLastPayrollCheckNumber()

	// Format dates for display
	weekStartDate, _ := time.Parse("2006-01-02", week.PeriodStart)
	weekEndDate, _ := time.Parse("2006-01-02", week.PeriodEnd)
	weekStartDisplay := weekStartDate.Format("Jan 2")
	weekEndDisplay := weekEndDate.Format("Jan 2, 2006")

	h.render(w, r, "payroll_week_edit.html", map[string]any{
		"Title":           "Edit Payroll - " + weekEndDisplay,
		"Active":          "payroll",
		"Entries":         entries,
		"Total":           total,
		"WeekID":          weekID,
		"WeekStart":       week.PeriodStart,
		"WeekEnd":         week.PeriodEnd,
		"WeekDisplay":     weekStartDisplay + " - " + weekEndDisplay,
		"LastCheckNumber": lastCheck,
	})
}

func (h *Handler) PayrollWeekDetail(w http.ResponseWriter, r *http.Request) {
	weekIDStr := r.PathValue("id")
	weekID, err := strconv.ParseInt(weekIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid week ID", http.StatusBadRequest)
		return
	}

	// Get the week details
	week, err := h.db.GetPayrollWeek(weekID)
	if err != nil {
		http.Error(w, "Week not found", http.StatusNotFound)
		return
	}

	entries, total, _ := h.db.GetWeeklyPayrollByWeekID(weekID)
	lastCheck, _ := h.db.GetLastPayrollCheckNumber()

	// Format dates for display (handle both "2006-01-02" and "2006-01-02T15:04:05Z" formats)
	weekStartDate, err := time.Parse("2006-01-02", week.PeriodStart)
	if err != nil {
		weekStartDate, err = time.Parse(time.RFC3339, week.PeriodStart)
		if err != nil {
			http.Error(w, "Invalid period start date: "+week.PeriodStart, http.StatusInternalServerError)
			return
		}
	}
	weekEndDate, err := time.Parse("2006-01-02", week.PeriodEnd)
	if err != nil {
		weekEndDate, err = time.Parse(time.RFC3339, week.PeriodEnd)
		if err != nil {
			http.Error(w, "Invalid period end date: "+week.PeriodEnd, http.StatusInternalServerError)
			return
		}
	}
	weekStartDisplay := weekStartDate.Format("Jan 2")
	weekEndDisplay := weekEndDate.Format("Jan 2, 2006")

	h.render(w, r, "payroll_detail.html", map[string]interface{}{
		"Title":           "Payroll - " + weekStartDisplay + " to " + weekEndDisplay,
		"Active":          "payroll",
		"Entries":         entries,
		"Total":           total,
		"WeekID":          weekID,
		"WeekStart":       week.PeriodStart,
		"WeekEnd":         week.PeriodEnd,
		"WeekDisplay":     weekStartDisplay + " - " + weekEndDisplay,
		"LastCheckNumber": lastCheck,
	})
}

func (h *Handler) PayrollNew(w http.ResponseWriter, r *http.Request) {
	employees, _ := h.db.ListEmployees(true)
	lastCheck, _ := h.db.GetLastPayrollCheckNumber()
	h.render(w, r, "payroll_form.html", map[string]interface{}{
		"Title":           "New Payroll",
		"Active":          "payroll",
		"Payroll":         models.Payroll{},
		"Employees":       employees,
		"LastCheckNumber": lastCheck,
	})
}

func (h *Handler) PayrollCreate(w http.ResponseWriter, r *http.Request) {
	employeeID, _ := strconv.ParseInt(r.FormValue("employee_id"), 10, 64)
	totalHours, _ := strconv.ParseFloat(r.FormValue("total_hours"), 64)
	hourlyRate, _ := strconv.ParseFloat(r.FormValue("hourly_rate"), 64)

	payroll := models.Payroll{
		EmployeeID:    employeeID,
		PeriodStart:   r.FormValue("period_start"),
		PeriodEnd:     r.FormValue("period_end"),
		TotalHours:    totalHours,
		HourlyRate:    hourlyRate,
		PaymentMethod: r.FormValue("payment_method"),
		CheckNumber:   r.FormValue("check_number"),
		Status:        r.FormValue("status"),
		DatePaid:      r.FormValue("date_paid"),
		Notes:         r.FormValue("notes"),
	}

	_, err := h.db.CreatePayroll(payroll)
	if err != nil {
		employees, _ := h.db.ListEmployees(true)
		lastCheck, _ := h.db.GetLastPayrollCheckNumber()
		h.render(w, r, "payroll_form.html", map[string]interface{}{
			"Title":           "New Payroll",
			"Active":          "payroll",
			"Payroll":         payroll,
			"Employees":       employees,
			"LastCheckNumber": lastCheck,
			"Error":           err.Error(),
		})
		return
	}
	http.Redirect(w, r, "/payroll", http.StatusFound)
}

func (h *Handler) PayrollEdit(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	payroll, err := h.db.GetPayroll(id)
	if err != nil {
		http.Redirect(w, r, "/payroll", http.StatusFound)
		return
	}
	employees, _ := h.db.ListEmployees(true)
	lastCheck, _ := h.db.GetLastPayrollCheckNumber()
	h.render(w, r, "payroll_form.html", map[string]interface{}{
		"Title":           "Edit Payroll",
		"Active":          "payroll",
		"Payroll":         payroll,
		"Employees":       employees,
		"LastCheckNumber": lastCheck,
	})
}

func (h *Handler) PayrollUpdate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	employeeID, _ := strconv.ParseInt(r.FormValue("employee_id"), 10, 64)
	totalHours, _ := strconv.ParseFloat(r.FormValue("total_hours"), 64)
	hourlyRate, _ := strconv.ParseFloat(r.FormValue("hourly_rate"), 64)

	payroll := models.Payroll{
		ID:            id,
		EmployeeID:    employeeID,
		PeriodStart:   r.FormValue("period_start"),
		PeriodEnd:     r.FormValue("period_end"),
		TotalHours:    totalHours,
		HourlyRate:    hourlyRate,
		PaymentMethod: r.FormValue("payment_method"),
		CheckNumber:   r.FormValue("check_number"),
		Status:        r.FormValue("status"),
		DatePaid:      r.FormValue("date_paid"),
		Notes:         r.FormValue("notes"),
	}

	err := h.db.UpdatePayroll(payroll)
	if err != nil {
		employees, _ := h.db.ListEmployees(true)
		lastCheck, _ := h.db.GetLastPayrollCheckNumber()
		h.render(w, r, "payroll_form.html", map[string]interface{}{
			"Title":           "Edit Payroll",
			"Active":          "payroll",
			"Payroll":         payroll,
			"Employees":       employees,
			"LastCheckNumber": lastCheck,
			"Error":           err.Error(),
		})
		return
	}
	http.Redirect(w, r, "/payroll", http.StatusFound)
}

func (h *Handler) PayrollPay(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	paymentMethod := r.FormValue("payment_method")
	checkNumber := r.FormValue("check_number")
	week := r.FormValue("week")

	h.db.MarkPayrollPaidWithDetails(id, paymentMethod, checkNumber)

	if week != "" {
		http.Redirect(w, r, "/payroll?week="+week, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/payroll", http.StatusFound)
}

func (h *Handler) PayrollSaveHours(w http.ResponseWriter, r *http.Request) {
	weekStart := r.FormValue("week_start")
	weekEnd := r.FormValue("week_end")

	// Get all active employees to process their hours
	employees, _ := h.db.ListEmployees(true)
	for _, emp := range employees {
		hoursStr := r.FormValue(fmt.Sprintf("hours_%d", emp.ID))
		hours, _ := strconv.ParseFloat(hoursStr, 64)
		if hours > 0 {
			h.db.UpsertWeeklyPayroll(emp.ID, weekStart, weekEnd, hours, emp.HourlyRate, emp.PaymentMethod)
		}
	}

	http.Redirect(w, r, "/payroll", http.StatusFound)
}

func (h *Handler) PayrollDelete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	week := r.FormValue("week")
	h.db.DeletePayroll(id)
	if week != "" {
		http.Redirect(w, r, "/payroll?week="+week, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/payroll", http.StatusFound)
}

// Reconciliations handlers

func (h *Handler) ReconciliationsList(w http.ResponseWriter, r *http.Request) {
	reconciliations, err := h.db.ListReconciliations()
	if err != nil {
		logger.FromContext(r.Context()).Error("reconciliations_list_error", "error", err.Error())
	}

	// Get months that already have reconciliations
	reconciledMonths, err := h.db.GetReconciledMonths()
	if err != nil {
		logger.FromContext(r.Context()).Error("reconciled_months_error", "error", err.Error())
		reconciledMonths = make(map[string]bool)
	}

	// Generate available months (last 12 months, excluding already reconciled)
	availableMonths := []models.MonthOption{}
	now := time.Now()
	// Start from previous month
	current := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.Local)

	for i := 0; i < 12; i++ {
		monthValue := current.Format("2006-01")
		if !reconciledMonths[monthValue] {
			availableMonths = append(availableMonths, models.MonthOption{
				Value:    monthValue,
				Label:    current.Format("January 2006"),
				Selected: len(availableMonths) == 0, // Select first available
			})
		}
		current = current.AddDate(0, -1, 0)
	}

	h.render(w, r, "reconciliations_list.html", map[string]any{
		"Title":           "Bank Statements",
		"Active":          "expenses",
		"Reconciliations": reconciliations,
		"AvailableMonths": availableMonths,
	})
}

func (h *Handler) ReconciliationsUpload(w http.ResponseWriter, r *http.Request) {
	l := logger.FromContext(r.Context())

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		l.Error("reconciliation_upload_parse_error", "error", err.Error())
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get selected month (YYYY-MM format)
	statementMonth := r.FormValue("statement_month")
	if statementMonth == "" {
		http.Error(w, "Statement month is required", http.StatusBadRequest)
		return
	}

	// Parse month to get last day of month for statement date
	monthTime, err := time.Parse("2006-01", statementMonth)
	if err != nil {
		l.Error("reconciliation_upload_month_parse", "error", err.Error())
		http.Error(w, "Invalid month format", http.StatusBadRequest)
		return
	}
	// Get last day of the month
	statementDate := monthTime.AddDate(0, 1, -1).Format("2006-01-02")

	// Get uploaded file
	file, header, err := r.FormFile("statement_file")
	if err != nil {
		l.Error("reconciliation_upload_file_error", "error", err.Error())
		http.Error(w, "Failed to get uploaded file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	l.Info("reconciliation_upload", "month", statementMonth, "filename", header.Filename, "size", header.Size)

	// Save file to filestore
	filePath, err := h.files.Save(header.Filename, file)
	if err != nil {
		l.Error("reconciliation_file_save_error", "error", err.Error())
		http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}

	// Create reconciliation record
	recon := models.BankReconciliation{
		StatementDate:   statementDate,
		StartingBalance: 0,
		EndingBalance:   0,
		Status:          "pending",
		FilePath:        filePath,
		Notes:           fmt.Sprintf("Uploaded: %s", header.Filename),
	}

	reconID, err := h.db.CreateReconciliation(recon)
	if err != nil {
		// Clean up saved file on error
		h.files.Delete(filePath)
		l.Error("reconciliation_create_error", "error", err.Error())
		http.Error(w, "Failed to create reconciliation", http.StatusInternalServerError)
		return
	}

	// Queue parse job
	jobPayload := map[string]any{
		"reconciliation_id": reconID,
		"file_path":         filePath,
	}
	jobID, err := h.db.CreateJob("parse_statement", jobPayload)
	if err != nil {
		l.Error("reconciliation_job_create_error", "error", err.Error())
		http.Error(w, "Failed to queue parse job", http.StatusInternalServerError)
		return
	}

	// Update reconciliation with job ID
	if err := h.db.UpdateReconciliationParseJob(reconID, jobID); err != nil {
		l.Error("reconciliation_update_job_error", "error", err.Error())
	}

	l.Info("reconciliation_job_queued", "reconciliation_id", reconID, "job_id", jobID)

	// Return JSON response for frontend polling
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"reconciliation_id": reconID,
		"job_id":            jobID,
	})
}

// ReconciliationsDetail shows a read-only view of a completed reconciliation
func (h *Handler) ReconciliationsDetail(w http.ResponseWriter, r *http.Request) {
	l := logger.FromContext(r.Context())

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/bank-statements", http.StatusFound)
		return
	}

	recon, err := h.db.GetReconciliation(id)
	if err != nil {
		l.Error("reconciliation_detail_get_error", "id", id, "error", err.Error())
		http.Redirect(w, r, "/bank-statements", http.StatusFound)
		return
	}

	transactions, err := h.db.GetBankTransactions(id)
	if err != nil {
		l.Error("reconciliation_detail_transactions_error", "id", id, "error", err.Error())
	}

	stats, err := h.db.GetReconciliationStats(id)
	if err != nil {
		l.Error("reconciliation_detail_stats_error", "id", id, "error", err.Error())
	}

	h.render(w, r, "reconciliation_detail.html", map[string]any{
		"Title":          "Bank Statement Details",
		"Active":         "expenses",
		"Reconciliation": recon,
		"Transactions":   transactions,
		"Stats":          stats,
	})
}

// ReconciliationsComplete marks a reconciliation as completed
func (h *Handler) ReconciliationsComplete(w http.ResponseWriter, r *http.Request) {
	l := logger.FromContext(r.Context())

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/bank-statements", http.StatusFound)
		return
	}

	if err := h.db.UpdateReconciliationStatus(id, "completed"); err != nil {
		l.Error("reconciliation_complete_error", "id", id, "error", err.Error())
	} else {
		l.Info("reconciliation_completed", "id", id)
	}

	http.Redirect(w, r, fmt.Sprintf("/bank-statements/%d", id), http.StatusFound)
}

// ReconciliationsReparse queues a new parse job for an existing reconciliation
func (h *Handler) ReconciliationsReparse(w http.ResponseWriter, r *http.Request) {
	l := logger.FromContext(r.Context())

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/bank-statements", http.StatusFound)
		return
	}

	recon, err := h.db.GetReconciliation(id)
	if err != nil {
		l.Error("reconciliation_reparse_get_error", "id", id, "error", err.Error())
		http.Redirect(w, r, "/bank-statements", http.StatusFound)
		return
	}

	// Reset status to pending
	if err := h.db.UpdateReconciliationStatus(id, "pending"); err != nil {
		l.Error("reconciliation_reparse_status_error", "id", id, "error", err.Error())
	}

	// Queue new parse job
	jobPayload := map[string]any{
		"reconciliation_id": id,
		"file_path":         recon.FilePath,
	}
	jobID, err := h.db.CreateJob("parse_statement", jobPayload)
	if err != nil {
		l.Error("reconciliation_reparse_job_error", "id", id, "error", err.Error())
		http.Error(w, "Failed to queue parse job", http.StatusInternalServerError)
		return
	}

	// Update reconciliation with new job ID
	if err := h.db.UpdateReconciliationParseJob(id, jobID); err != nil {
		l.Error("reconciliation_reparse_update_error", "id", id, "error", err.Error())
	}

	l.Info("reconciliation_reparse_queued", "reconciliation_id", id, "job_id", jobID)

	http.Redirect(w, r, "/bank-statements", http.StatusFound)
}

// ReconciliationsReview shows the reconciliation review page
func (h *Handler) ReconciliationsReview(w http.ResponseWriter, r *http.Request) {
	l := logger.FromContext(r.Context())

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/bank-statements", http.StatusFound)
		return
	}

	recon, err := h.db.GetReconciliation(id)
	if err != nil {
		l.Error("reconciliation_get_error", "id", id, "error", err.Error())
		http.Redirect(w, r, "/bank-statements", http.StatusFound)
		return
	}

	transactions, err := h.db.GetBankTransactions(id)
	if err != nil {
		l.Error("reconciliation_transactions_error", "id", id, "error", err.Error())
	}

	stats, err := h.db.GetReconciliationStats(id)
	if err != nil {
		l.Error("reconciliation_stats_error", "id", id, "error", err.Error())
	}

	// Get expenses for matching (paid expenses from around the statement period)
	vendors, _ := h.db.ListVendors()
	expenses, _, _ := h.db.ListExpenses(models.ExpenseFilter{Status: "paid"})

	h.render(w, r, "reconciliation_edit.html", map[string]any{
		"Title":          "Review Reconciliation",
		"Active":         "expenses",
		"Reconciliation": recon,
		"Transactions":   transactions,
		"Stats":          stats,
		"Expenses":       expenses,
		"Vendors":        vendors,
	})
}

// ReconciliationsMatch manually matches a bank transaction to an expense
func (h *Handler) ReconciliationsMatch(w http.ResponseWriter, r *http.Request) {
	l := logger.FromContext(r.Context())

	reconID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/bank-statements", http.StatusFound)
		return
	}

	txnID, err := strconv.ParseInt(r.FormValue("transaction_id"), 10, 64)
	if err != nil {
		l.Error("match_invalid_txn_id", "error", err.Error())
		http.Redirect(w, r, fmt.Sprintf("/bank-statements/%d/review", reconID), http.StatusFound)
		return
	}

	expenseID, err := strconv.ParseInt(r.FormValue("expense_id"), 10, 64)
	if err != nil {
		l.Error("match_invalid_expense_id", "error", err.Error())
		http.Redirect(w, r, fmt.Sprintf("/bank-statements/%d/review", reconID), http.StatusFound)
		return
	}

	if err := h.db.MatchBankTransaction(txnID, expenseID, "manual"); err != nil {
		l.Error("match_error", "txn_id", txnID, "expense_id", expenseID, "error", err.Error())
	} else {
		l.Info("transaction_matched", "txn_id", txnID, "expense_id", expenseID)
	}

	http.Redirect(w, r, fmt.Sprintf("/bank-statements/%d/review", reconID), http.StatusFound)
}

// ReconciliationsUnmatch removes a match from a bank transaction
func (h *Handler) ReconciliationsUnmatch(w http.ResponseWriter, r *http.Request) {
	l := logger.FromContext(r.Context())

	reconID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/bank-statements", http.StatusFound)
		return
	}

	txnID, err := strconv.ParseInt(r.FormValue("transaction_id"), 10, 64)
	if err != nil {
		l.Error("unmatch_invalid_txn_id", "error", err.Error())
		http.Redirect(w, r, fmt.Sprintf("/bank-statements/%d/review", reconID), http.StatusFound)
		return
	}

	// Get the transaction to check if it was a "created" expense
	txn, err := h.db.GetBankTransaction(txnID)
	if err != nil {
		l.Error("unmatch_get_txn_error", "txn_id", txnID, "error", err.Error())
		http.Redirect(w, r, fmt.Sprintf("/bank-statements/%d/review", reconID), http.StatusFound)
		return
	}

	// If this was a created expense, delete it
	if txn.MatchStatus == "created" && txn.MatchedExpenseID != nil {
		if err := h.db.DeleteExpense(*txn.MatchedExpenseID); err != nil {
			l.Error("unmatch_delete_expense_error", "expense_id", *txn.MatchedExpenseID, "error", err.Error())
		} else {
			l.Info("created_expense_deleted", "txn_id", txnID, "expense_id", *txn.MatchedExpenseID)
		}
	}

	if err := h.db.UnmatchBankTransaction(txnID); err != nil {
		l.Error("unmatch_error", "txn_id", txnID, "error", err.Error())
	} else {
		l.Info("transaction_unmatched", "txn_id", txnID)
	}

	http.Redirect(w, r, fmt.Sprintf("/bank-statements/%d/review", reconID), http.StatusFound)
}

// ReconciliationsIgnore marks a bank transaction as ignored
func (h *Handler) ReconciliationsIgnore(w http.ResponseWriter, r *http.Request) {
	l := logger.FromContext(r.Context())

	reconID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/bank-statements", http.StatusFound)
		return
	}

	txnID, err := strconv.ParseInt(r.FormValue("transaction_id"), 10, 64)
	if err != nil {
		l.Error("ignore_invalid_txn_id", "error", err.Error())
		http.Redirect(w, r, fmt.Sprintf("/bank-statements/%d/review", reconID), http.StatusFound)
		return
	}

	reason := r.FormValue("reason")
	if reason == "" {
		reason = "Manually ignored"
	}

	if err := h.db.IgnoreBankTransaction(txnID, reason); err != nil {
		l.Error("ignore_error", "txn_id", txnID, "error", err.Error())
	} else {
		l.Info("transaction_ignored", "txn_id", txnID, "reason", reason)
	}

	http.Redirect(w, r, fmt.Sprintf("/bank-statements/%d/review", reconID), http.StatusFound)
}

// ReconciliationsCreateExpense creates a new expense from a bank transaction
func (h *Handler) ReconciliationsCreateExpense(w http.ResponseWriter, r *http.Request) {
	l := logger.FromContext(r.Context())

	reconID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/bank-statements", http.StatusFound)
		return
	}

	txnID, err := strconv.ParseInt(r.FormValue("transaction_id"), 10, 64)
	if err != nil {
		l.Error("create_expense_invalid_txn_id", "error", err.Error())
		http.Redirect(w, r, fmt.Sprintf("/bank-statements/%d/review", reconID), http.StatusFound)
		return
	}

	vendorID, err := strconv.ParseInt(r.FormValue("vendor_id"), 10, 64)
	if err != nil {
		l.Error("create_expense_invalid_vendor_id", "error", err.Error())
		http.Redirect(w, r, fmt.Sprintf("/bank-statements/%d/review", reconID), http.StatusFound)
		return
	}

	// Get the bank transaction
	txn, err := h.db.GetBankTransaction(txnID)
	if err != nil {
		l.Error("create_expense_get_txn_error", "txn_id", txnID, "error", err.Error())
		http.Redirect(w, r, fmt.Sprintf("/bank-statements/%d/review", reconID), http.StatusFound)
		return
	}

	// Create the expense (amount is negative in bank txn, so we use absolute value)
	amount := txn.Amount
	if amount < 0 {
		amount = -amount
	}

	// Map bank transaction type to expense payment type
	paymentType := ""
	switch txn.TransactionType {
	case "check":
		paymentType = "check"
	case "debit", "ach", "electronic":
		paymentType = "debit"
	case "credit":
		paymentType = "credit"
	default:
		paymentType = "debit" // default for unknown types
	}

	expense := models.Expense{
		Date:        txn.PostingDate,
		VendorID:    vendorID,
		Amount:      amount,
		Status:      "paid",
		PaymentType: paymentType,
		CheckNumber: txn.CheckNumber,
		DatePaid:    txn.PostingDate,
		Notes:       fmt.Sprintf("Created from bank statement: %s", txn.Description),
	}

	expenseID, err := h.db.CreateExpense(expense)
	if err != nil {
		l.Error("create_expense_error", "error", err.Error())
		http.Redirect(w, r, fmt.Sprintf("/bank-statements/%d/review", reconID), http.StatusFound)
		return
	}

	// Mark the transaction as created and link it
	if err := h.db.MarkBankTransactionCreated(txnID, expenseID); err != nil {
		l.Error("mark_txn_created_error", "txn_id", txnID, "expense_id", expenseID, "error", err.Error())
	} else {
		l.Info("expense_created_from_txn", "txn_id", txnID, "expense_id", expenseID)
	}

	http.Redirect(w, r, fmt.Sprintf("/bank-statements/%d/review", reconID), http.StatusFound)
}

// ReconciliationsUpdateType updates the transaction type for a bank transaction
func (h *Handler) ReconciliationsUpdateType(w http.ResponseWriter, r *http.Request) {
	l := logger.FromContext(r.Context())

	reconID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/bank-statements", http.StatusFound)
		return
	}

	txnID, err := strconv.ParseInt(r.FormValue("transaction_id"), 10, 64)
	if err != nil {
		l.Error("update_type_invalid_txn_id", "error", err.Error())
		http.Redirect(w, r, fmt.Sprintf("/bank-statements/%d/review", reconID), http.StatusFound)
		return
	}

	txnType := r.FormValue("transaction_type")

	// Determine if amount should be positive or negative based on type
	// Credits (positive): deposit, ach, refund
	// Debits (negative): check, debit, transfer, fee, other
	shouldBePositive := txnType == "deposit" || txnType == "ach" || txnType == "refund"

	if err := h.db.UpdateBankTransactionTypeAndSign(txnID, txnType, shouldBePositive); err != nil {
		l.Error("update_type_error", "txn_id", txnID, "error", err.Error())
	} else {
		l.Info("transaction_type_updated", "txn_id", txnID, "type", txnType)
	}

	http.Redirect(w, r, fmt.Sprintf("/bank-statements/%d/review", reconID), http.StatusFound)
}

// JobStatus returns the status of a background job as JSON (for polling)
func (h *Handler) JobStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	job, err := h.db.GetJob(id)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":       job.ID,
		"status":   job.Status,
		"progress": job.Progress,
		"result":   job.Result,
	})
}
