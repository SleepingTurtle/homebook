package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"homebooks/internal/auth"
	"homebooks/internal/database"
	"homebooks/internal/models"
)

type Handler struct {
	db   *database.DB
	auth *auth.Auth
	tmpl *template.Template
}

func New(db *database.DB, a *auth.Auth, tmpl *template.Template) *Handler {
	return &Handler{
		db:   db,
		auth: a,
		tmpl: tmpl,
	}
}

func (h *Handler) render(w http.ResponseWriter, name string, data map[string]interface{}) {
	err := h.tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Login handlers
func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	token := h.auth.GetSessionFromRequest(r)
	if token != "" && h.auth.ValidateSession(token) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	h.render(w, "login.html", map[string]interface{}{"Error": ""})
}

func (h *Handler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")

	if !h.auth.CheckPassword(password) {
		h.render(w, "login.html", map[string]interface{}{"Error": "Invalid password"})
		return
	}

	token, err := h.auth.CreateSession()
	if err != nil {
		h.render(w, "login.html", map[string]interface{}{"Error": "Failed to create session"})
		return
	}

	h.auth.SetSessionCookie(w, token)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := h.auth.GetSessionFromRequest(r)
	if token != "" {
		h.auth.DeleteSession(token)
	}
	h.auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

// Dashboard
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	unpaidExpenses, expenseTotal, _ := h.db.ListUnpaidExpenses()
	unpaidPayroll, payrollTotal, _ := h.db.ListUnpaidPayroll()
	recentSalesGrouped, recentSalesTotal, _ := h.db.ListRecentSalesGrouped(7)

	data := models.DashboardData{
		UnpaidExpensesTotal:  expenseTotal,
		UnpaidExpensesCount:  len(unpaidExpenses),
		UnpaidPayrollTotal:   payrollTotal,
		UnpaidPayrollCount:   len(unpaidPayroll),
		RecentSalesGrouped:   recentSalesGrouped,
		RecentSalesTotal:     recentSalesTotal,
		UnpaidExpenses:       unpaidExpenses,
		UnpaidPayroll:        unpaidPayroll,
	}

	h.render(w, "dashboard.html", map[string]interface{}{
		"Title":  "Dashboard",
		"Active": "dashboard",
		"Data":   data,
	})
}

// Vendors handlers
func (h *Handler) VendorsList(w http.ResponseWriter, r *http.Request) {
	vendors, _ := h.db.ListVendors()
	h.render(w, "vendors_list.html", map[string]interface{}{
		"Title":   "Vendors",
		"Active":  "vendors",
		"Vendors": vendors,
	})
}

func (h *Handler) VendorsNew(w http.ResponseWriter, r *http.Request) {
	h.render(w, "vendors_form.html", map[string]interface{}{
		"Title":      "New Vendor",
		"Active":     "vendors",
		"Vendor":     models.Vendor{},
		"Categories": models.VendorCategories,
	})
}

func (h *Handler) VendorsCreate(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	category := r.FormValue("category")
	description := r.FormValue("description")

	if name == "" {
		h.render(w, "vendors_form.html", map[string]interface{}{
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
		h.render(w, "vendors_form.html", map[string]interface{}{
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
	h.render(w, "vendors_form.html", map[string]interface{}{
		"Title":      "Edit Vendor",
		"Active":     "vendors",
		"Vendor":     vendor,
		"Categories": models.VendorCategories,
	})
}

func (h *Handler) VendorsUpdate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	name := r.FormValue("name")
	category := r.FormValue("category")
	description := r.FormValue("description")

	if name == "" {
		h.render(w, "vendors_form.html", map[string]interface{}{
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
		h.render(w, "vendors_form.html", map[string]interface{}{
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
	h.render(w, "employees_list.html", map[string]interface{}{
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
		h.render(w, "employees_list.html", map[string]interface{}{
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
		h.render(w, "employees_list.html", map[string]interface{}{
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
	h.render(w, "sales_list.html", map[string]any{
		"Title":     "Daily Sales",
		"Active":    "sales",
		"Grouped":   grouped,
		"TodayDate": time.Now().Format("2006-01-02"),
	})
}

func (h *Handler) SalesNew(w http.ResponseWriter, r *http.Request) {
	sale := models.DailySale{Date: time.Now().Format("2006-01-02")}
	h.render(w, "sales_form.html", map[string]interface{}{
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
		h.render(w, "sales_form.html", map[string]interface{}{
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
	h.render(w, "sales_form.html", map[string]interface{}{
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
		h.render(w, "sales_form.html", map[string]interface{}{
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
	h.render(w, "delivery_form.html", map[string]any{
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
	h.render(w, "delivery_form.html", map[string]any{
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
		h.render(w, "delivery_form.html", map[string]any{
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
		StartDate: r.URL.Query().Get("start_date"),
		EndDate:   r.URL.Query().Get("end_date"),
		Status:    r.URL.Query().Get("status"),
		VendorID:  vendorID,
	}
	expenses, total, _ := h.db.ListExpenses(filter)
	vendors, _ := h.db.ListVendors()
	h.render(w, "expenses_list.html", map[string]interface{}{
		"Title":    "Expenses",
		"Active":   "expenses",
		"Expenses": expenses,
		"Total":    total,
		"Vendors":  vendors,
		"Filter":   filter,
	})
}

func (h *Handler) ExpensesNew(w http.ResponseWriter, r *http.Request) {
	vendors, _ := h.db.ListVendors()
	lastCheck, _ := h.db.GetLastExpenseCheckNumber()
	h.render(w, "expenses_form.html", map[string]interface{}{
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
		DatePaid:      r.FormValue("date_paid"),
		Notes:         r.FormValue("notes"),
	}

	_, err := h.db.CreateExpense(expense)
	if err != nil {
		vendors, _ := h.db.ListVendors()
		lastCheck, _ := h.db.GetLastExpenseCheckNumber()
		h.render(w, "expenses_form.html", map[string]interface{}{
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
	h.render(w, "expenses_form.html", map[string]interface{}{
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
		DatePaid:      r.FormValue("date_paid"),
		Notes:         r.FormValue("notes"),
	}

	err := h.db.UpdateExpense(expense)
	if err != nil {
		vendors, _ := h.db.ListVendors()
		lastCheck, _ := h.db.GetLastExpenseCheckNumber()
		h.render(w, "expenses_form.html", map[string]interface{}{
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

func (h *Handler) ExpensesPay(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	h.db.MarkExpensePaid(id)
	referer := r.Header.Get("Referer")
	if referer != "" && referer == "/" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/expenses", http.StatusFound)
}

func (h *Handler) ExpensesDelete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	h.db.DeleteExpense(id)
	http.Redirect(w, r, "/expenses", http.StatusFound)
}

// Payroll handlers
func (h *Handler) PayrollList(w http.ResponseWriter, r *http.Request) {
	employeeID, _ := strconv.ParseInt(r.URL.Query().Get("employee_id"), 10, 64)
	filter := models.PayrollFilter{
		StartDate:  r.URL.Query().Get("start_date"),
		EndDate:    r.URL.Query().Get("end_date"),
		Status:     r.URL.Query().Get("status"),
		EmployeeID: employeeID,
	}
	payrolls, total, _ := h.db.ListPayroll(filter)
	employees, _ := h.db.ListEmployees(true)
	h.render(w, "payroll_list.html", map[string]interface{}{
		"Title":     "Payroll",
		"Active":    "payroll",
		"Payrolls":  payrolls,
		"Total":     total,
		"Employees": employees,
		"Filter":    filter,
	})
}

func (h *Handler) PayrollNew(w http.ResponseWriter, r *http.Request) {
	employees, _ := h.db.ListEmployees(true)
	lastCheck, _ := h.db.GetLastPayrollCheckNumber()
	h.render(w, "payroll_form.html", map[string]interface{}{
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
		h.render(w, "payroll_form.html", map[string]interface{}{
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
	h.render(w, "payroll_form.html", map[string]interface{}{
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
		h.render(w, "payroll_form.html", map[string]interface{}{
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
	h.db.MarkPayrollPaid(id)
	referer := r.Header.Get("Referer")
	if referer != "" && referer == "/" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/payroll", http.StatusFound)
}

func (h *Handler) PayrollDelete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	h.db.DeletePayroll(id)
	http.Redirect(w, r, "/payroll", http.StatusFound)
}
