package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"homebooks/internal/auth"
	"homebooks/internal/database"
	"homebooks/internal/filestore"
	"homebooks/internal/handlers"
	"homebooks/internal/jobs"
	"homebooks/internal/logger"
	"homebooks/internal/version"
	"homebooks/web"
)

func main() {
	// Handle --version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("HomeBooks %s (built %s, commit %s)\n",
			version.Version, version.BuildTime, version.GitCommit)
		os.Exit(0)
	}

	// Initialize logger first
	logger.Init()
	log := logger.Default()

	// Get database path from env or use default
	dbPath := os.Getenv("HOMEBOOKS_DB_PATH")
	if dbPath == "" {
		dbPath = "./data/homebooks.db"
	}

	// Get port from env or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Open database
	db, err := database.Open(dbPath)
	if err != nil {
		log.Error("database_open_failed", "path", dbPath, "error", err.Error())
		os.Exit(1)
	}
	defer db.Close()

	// Initialize schema
	if err := db.Init(); err != nil {
		log.Error("database_init_failed", "error", err.Error())
		os.Exit(1)
	}

	// Parse templates
	tmpl, err := template.ParseFS(web.TemplatesFS, "templates/*.html")
	if err != nil {
		log.Error("template_parse_failed", "error", err.Error())
		os.Exit(1)
	}

	// Initialize auth
	a := auth.New(db.DB)

	// Clean expired sessions on startup
	a.CleanExpiredSessions()

	// Initialize filestore (in data/uploads directory alongside database)
	uploadsPath := filepath.Join(filepath.Dir(dbPath), "uploads")
	files, err := filestore.New(uploadsPath)
	if err != nil {
		log.Error("filestore_init_failed", "path", uploadsPath, "error", err.Error())
		os.Exit(1)
	}

	// Initialize and start job worker
	worker := jobs.NewWorker(db, log)
	worker.Register("parse_statement", jobs.ParseStatementHandler(uploadsPath))
	worker.Start()
	defer worker.Stop()

	// Initialize handlers
	h := handlers.New(db, a, tmpl, files)

	// Setup routes
	mux := http.NewServeMux()

	// Static files - use fs.Sub to get the static subdirectory
	staticFS, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		log.Error("static_fs_failed", "error", err.Error())
		os.Exit(1)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))

	// Auth routes (no auth required)
	mux.HandleFunc("GET /login", h.LoginPage)
	mux.HandleFunc("POST /login", h.LoginSubmit)
	mux.HandleFunc("POST /logout", h.Logout)

	// Protected routes
	mux.HandleFunc("GET /{$}", h.Dashboard)

	// Sales
	mux.HandleFunc("GET /sales", h.SalesList)
	mux.HandleFunc("GET /sales/new", h.SalesNew)
	mux.HandleFunc("POST /sales", h.SalesCreate)
	mux.HandleFunc("GET /sales/{id}/edit", h.SalesEdit)
	mux.HandleFunc("POST /sales/{id}", h.SalesUpdate)
	mux.HandleFunc("POST /sales/{id}/delete", h.SalesDelete)
	mux.HandleFunc("GET /api/sales/shifts", h.SalesShiftsAPI)

	// Delivery Sales
	mux.HandleFunc("GET /sales/delivery/new", h.DeliveryNew)
	mux.HandleFunc("GET /sales/delivery/{date}/edit", h.DeliveryEdit)
	mux.HandleFunc("POST /sales/delivery", h.DeliverySave)

	// Expenses
	mux.HandleFunc("GET /expenses", h.ExpensesList)
	mux.HandleFunc("GET /expenses/new", h.ExpensesNew)
	mux.HandleFunc("POST /expenses", h.ExpensesCreate)
	mux.HandleFunc("GET /expenses/{id}/edit", h.ExpensesEdit)
	mux.HandleFunc("POST /expenses/{id}", h.ExpensesUpdate)
	mux.HandleFunc("GET /expenses/{id}/pay", h.ExpensesPayForm)
	mux.HandleFunc("POST /expenses/{id}/pay", h.ExpensesPay)
	mux.HandleFunc("POST /expenses/{id}/delete", h.ExpensesDelete)
	mux.HandleFunc("GET /expenses/{id}/receipt", h.ExpensesDownloadReceipt)
	mux.HandleFunc("POST /expenses/{id}/receipt", h.ExpensesUploadReceipt)
	mux.HandleFunc("POST /expenses/{id}/receipt/delete", h.ExpensesDeleteReceipt)

	// Payroll
	mux.HandleFunc("GET /payroll", h.PayrollList)
	mux.HandleFunc("POST /payroll/save", h.PayrollSaveHours)
	mux.HandleFunc("GET /payroll/weeks/new", h.PayrollWeekNew)
	mux.HandleFunc("GET /payroll/weeks/{id}/edit", h.PayrollWeekEdit)
	mux.HandleFunc("GET /payroll/history/{id}", h.PayrollWeekDetail)
	mux.HandleFunc("GET /payroll/new", h.PayrollNew)
	mux.HandleFunc("POST /payroll", h.PayrollCreate)
	mux.HandleFunc("GET /payroll/entry/{id}/edit", h.PayrollEdit)
	mux.HandleFunc("POST /payroll/entry/{id}", h.PayrollUpdate)
	mux.HandleFunc("POST /payroll/entry/{id}/pay", h.PayrollPay)
	mux.HandleFunc("POST /payroll/entry/{id}/delete", h.PayrollDelete)

	// Bank Statements
	mux.HandleFunc("GET /bank-statements", h.ReconciliationsList)
	mux.HandleFunc("POST /bank-statements/upload", h.ReconciliationsUpload)
	mux.HandleFunc("GET /bank-statements/{id}", h.ReconciliationsReview)
	mux.HandleFunc("POST /bank-statements/{id}/reparse", h.ReconciliationsReparse)
	mux.HandleFunc("POST /bank-statements/{id}/complete", h.ReconciliationsComplete)
	mux.HandleFunc("POST /bank-statements/{id}/match", h.ReconciliationsMatch)
	mux.HandleFunc("POST /bank-statements/{id}/unmatch", h.ReconciliationsUnmatch)
	mux.HandleFunc("POST /bank-statements/{id}/ignore", h.ReconciliationsIgnore)
	mux.HandleFunc("POST /bank-statements/{id}/create-expense", h.ReconciliationsCreateExpense)
	mux.HandleFunc("POST /bank-statements/{id}/update-type", h.ReconciliationsUpdateType)
	mux.HandleFunc("POST /bank-statements/{id}/delete", h.ReconciliationsDelete)

	// Jobs API
	mux.HandleFunc("GET /api/jobs/{id}", h.JobStatus)

	// Version API
	mux.HandleFunc("GET /api/version", h.APIVersion)

	// Vendors
	mux.HandleFunc("GET /vendors", h.VendorsList)
	mux.HandleFunc("GET /vendors/new", h.VendorsNew)
	mux.HandleFunc("GET /vendors/{id}", h.VendorsShow)
	mux.HandleFunc("POST /vendors", h.VendorsCreate)
	mux.HandleFunc("GET /vendors/{id}/edit", h.VendorsEdit)
	mux.HandleFunc("POST /vendors/{id}", h.VendorsUpdate)
	mux.HandleFunc("POST /vendors/{id}/delete", h.VendorsDelete)

	// Employees
	mux.HandleFunc("GET /employees", h.EmployeesList)
	mux.HandleFunc("POST /employees", h.EmployeesCreate)
	mux.HandleFunc("POST /employees/{id}/deactivate", h.EmployeesDeactivate)
	mux.HandleFunc("POST /employees/{id}/reactivate", h.EmployeesReactivate)

	// Wrap with middleware: logging -> auth -> mux
	handler := logger.HTTPMiddleware(a.Middleware(mux))

	log.Info("server_starting", "port", port, "address", "http://localhost:"+port, "version", version.Version)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Error("server_failed", "error", err.Error())
		os.Exit(1)
	}
}
