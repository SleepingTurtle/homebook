-- Bank Reconciliation Seed Data
-- Run with: sqlite3 ./data/homebooks.db < scripts/seed_reconciliations.sql

-- Clear existing data first to ensure predictable IDs
DELETE FROM bank_transactions;
DELETE FROM bank_reconciliations;
DELETE FROM sqlite_sequence WHERE name='bank_transactions';
DELETE FROM sqlite_sequence WHERE name='bank_reconciliations';

-- ============================================
-- December 2025 - COMPLETED reconciliation (ID: 1)
-- ============================================

INSERT INTO bank_reconciliations (
    statement_date, starting_balance, ending_balance, status,
    file_path, account_last_four, parsed_at, reconciled_at,
    electronic_deposits, electronic_payments, checks_paid, service_fees
) VALUES (
    '2025-12-31', 15420.50, 18245.75, 'completed',
    '', '4521', '2026-01-05 10:30:00', '2026-01-05 14:15:00',
    42850.00, 28575.25, 11200.50, 249.00
);

-- December 2025 Transactions - ALL MATCHED/IGNORED/CREATED

-- Deposits (positive amounts)
INSERT INTO bank_transactions (reconciliation_id, posting_date, description, amount, transaction_type, match_status, notes) VALUES
(1, '2025-12-02', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 3245.67, 'deposit', 'ignored', 'Daily card settlement'),
(1, '2025-12-03', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 2876.43, 'deposit', 'ignored', 'Daily card settlement'),
(1, '2025-12-04', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 3521.89, 'deposit', 'ignored', 'Daily card settlement'),
(1, '2025-12-05', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 4102.55, 'deposit', 'ignored', 'Daily card settlement'),
(1, '2025-12-06', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 4890.21, 'deposit', 'ignored', 'Daily card settlement'),
(1, '2025-12-09', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 3156.78, 'deposit', 'ignored', 'Daily card settlement'),
(1, '2025-12-10', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 2945.32, 'deposit', 'ignored', 'Daily card settlement'),
(1, '2025-12-11', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 3678.90, 'deposit', 'ignored', 'Daily card settlement'),
(1, '2025-12-12', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 3412.56, 'deposit', 'ignored', 'Daily card settlement'),
(1, '2025-12-13', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 5234.88, 'deposit', 'ignored', 'Daily card settlement'),
(1, '2025-12-16', 'ACH CREDIT GRUBHUB PAYOUT', 1245.67, 'ach', 'ignored', 'Weekly delivery payout'),
(1, '2025-12-16', 'ACH CREDIT DOORDASH PAYOUT', 987.54, 'ach', 'ignored', 'Weekly delivery payout'),
(1, '2025-12-17', 'DEPOSIT CASH/CHECK DEPOSIT', 2500.00, 'deposit', 'ignored', 'Cash deposit'),
(1, '2025-12-23', 'ACH CREDIT GRUBHUB PAYOUT', 1052.60, 'ach', 'ignored', 'Weekly delivery payout');

-- Checks paid (negative amounts)
INSERT INTO bank_transactions (reconciliation_id, posting_date, description, amount, transaction_type, check_number, vendor_hint, match_status) VALUES
(1, '2025-12-03', 'CHECK 1205 SYSCO FOODS', -2450.75, 'check', '1205', 'SYSCO', 'matched'),
(1, '2025-12-05', 'CHECK 1206 US FOODS INC', -1875.50, 'check', '1206', 'US FOODS', 'matched'),
(1, '2025-12-10', 'CHECK 1207 SYSCO FOODS', -2890.25, 'check', '1207', 'SYSCO', 'matched'),
(1, '2025-12-12', 'CHECK 1208 PRODUCE JUNCTION', -645.80, 'check', '1208', 'PRODUCE JUNCTION', 'matched'),
(1, '2025-12-17', 'CHECK 1209 SYSCO FOODS', -2178.45, 'check', '1209', 'SYSCO', 'matched'),
(1, '2025-12-19', 'CHECK 1210 RESTAURANT DEPOT', -892.35, 'check', '1210', 'RESTAURANT DEPOT', 'matched'),
(1, '2025-12-23', 'CHECK 1211 US FOODS INC', -267.40, 'check', '1211', 'US FOODS', 'matched');

-- ACH/Electronic payments (negative amounts)
INSERT INTO bank_transactions (reconciliation_id, posting_date, description, amount, transaction_type, vendor_hint, match_status) VALUES
(1, '2025-12-01', 'ACH DEBIT VERIZON WIRELESS', -245.89, 'ach', 'VERIZON', 'matched'),
(1, '2025-12-05', 'ACH DEBIT PECO ENERGY', -1876.45, 'ach', 'PECO', 'matched'),
(1, '2025-12-10', 'ACH DEBIT STATE FARM INS', -650.00, 'ach', 'STATE FARM', 'matched'),
(1, '2025-12-15', 'ACH DEBIT PGW GAS', -425.67, 'ach', 'PGW', 'matched'),
(1, '2025-12-15', 'ACH DEBIT WASTE MGMT', -189.00, 'ach', 'WASTE MGMT', 'matched'),
(1, '2025-12-20', 'ACH DEBIT PHILADELPHIA WATER', -156.78, 'ach', 'PHILA WATER', 'matched'),
(1, '2025-12-28', 'ONLINE PAYMENT COMCAST BUSINESS', -275.99, 'debit', 'COMCAST', 'matched'),
(1, '2025-12-01', 'ACH DEBIT QUICKBOOKS PAYROLL', -4250.00, 'ach', 'QUICKBOOKS', 'created'),
(1, '2025-12-15', 'ACH DEBIT QUICKBOOKS PAYROLL', -4350.00, 'ach', 'QUICKBOOKS', 'created'),
(1, '2025-12-29', 'ACH DEBIT QUICKBOOKS PAYROLL', -4125.00, 'ach', 'QUICKBOOKS', 'created');

-- Debit card purchases (negative amounts)
INSERT INTO bank_transactions (reconciliation_id, posting_date, description, amount, transaction_type, vendor_hint, match_status) VALUES
(1, '2025-12-04', 'DEBIT CARD PURCHASE HOME DEPOT', -156.78, 'debit', 'HOME DEPOT', 'created'),
(1, '2025-12-08', 'DEBIT CARD PURCHASE STAPLES', -89.45, 'debit', 'STAPLES', 'created'),
(1, '2025-12-11', 'DEBIT CARD PURCHASE RESTAURANT DEPOT', -234.56, 'debit', 'RESTAURANT DEPOT', 'created'),
(1, '2025-12-18', 'DEBIT CARD PURCHASE COSTCO', -567.89, 'debit', 'COSTCO', 'created'),
(1, '2025-12-22', 'DEBIT CARD PURCHASE AMAZON', -124.99, 'debit', 'AMAZON', 'created');

-- Fees (negative amounts)
INSERT INTO bank_transactions (reconciliation_id, posting_date, description, amount, transaction_type, match_status, notes) VALUES
(1, '2025-12-31', 'SERVICE CHARGE', -25.00, 'fee', 'ignored', 'Monthly service fee'),
(1, '2025-12-31', 'ANALYSIS SERVICE CHARGE', -124.00, 'fee', 'ignored', 'Account analysis fee'),
(1, '2025-12-15', 'ACH RETURN FEE', -35.00, 'fee', 'ignored', 'ACH return'),
(1, '2025-12-20', 'WIRE TRANSFER FEE', -25.00, 'fee', 'ignored', 'Outgoing wire'),
(1, '2025-12-31', 'PAPER STATEMENT FEE', -5.00, 'fee', 'ignored', 'Paper statement'),
(1, '2025-12-31', 'OVERDRAFT FEE', -35.00, 'fee', 'ignored', 'Overdraft charge');


-- ============================================
-- January 2026 - IN PROGRESS reconciliation
-- ============================================

INSERT INTO bank_reconciliations (
    statement_date, starting_balance, ending_balance, status,
    file_path, account_last_four, parsed_at,
    electronic_deposits, electronic_payments, checks_paid, service_fees
) VALUES (
    '2026-01-15', 18245.75, 0, 'parsed',
    '', '4521', '2026-01-16 09:00:00',
    28450.00, 15230.50, 8450.00, 175.00
);

-- January 2026 Transactions - MIX OF MATCHED AND UNMATCHED

-- Deposits (some ignored, some unmatched)
INSERT INTO bank_transactions (reconciliation_id, posting_date, description, amount, transaction_type, match_status, notes) VALUES
(2, '2026-01-02', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 2156.78, 'deposit', 'ignored', 'Daily card settlement'),
(2, '2026-01-03', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 3421.90, 'deposit', 'ignored', 'Daily card settlement'),
(2, '2026-01-06', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 2987.65, 'deposit', 'unmatched', ''),
(2, '2026-01-07', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 3654.32, 'deposit', 'unmatched', ''),
(2, '2026-01-08', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 4125.89, 'deposit', 'unmatched', ''),
(2, '2026-01-09', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 3567.44, 'deposit', 'unmatched', ''),
(2, '2026-01-10', 'DEPOSIT CREDIT CARD SETTLEMENT SQUARE', 4890.12, 'deposit', 'unmatched', ''),
(2, '2026-01-13', 'ACH CREDIT GRUBHUB PAYOUT', 1456.78, 'ach', 'unmatched', ''),
(2, '2026-01-13', 'ACH CREDIT DOORDASH PAYOUT', 1189.12, 'ach', 'unmatched', ''),
(2, '2026-01-14', 'DEPOSIT CASH/CHECK DEPOSIT', 1000.00, 'deposit', 'unmatched', '');

-- Checks paid (some matched, some unmatched)
INSERT INTO bank_transactions (reconciliation_id, posting_date, description, amount, transaction_type, check_number, vendor_hint, match_status) VALUES
(2, '2026-01-03', 'CHECK 1212 SYSCO FOODS', -2678.45, 'check', '1212', 'SYSCO', 'matched'),
(2, '2026-01-07', 'CHECK 1213 US FOODS INC', -1956.80, 'check', '1213', 'US FOODS', 'matched'),
(2, '2026-01-08', 'CHECK 1214 PRODUCE JUNCTION', -534.25, 'check', '1214', 'PRODUCE JUNCTION', 'unmatched'),
(2, '2026-01-10', 'CHECK 1215 SYSCO FOODS', -2145.90, 'check', '1215', 'SYSCO', 'unmatched'),
(2, '2026-01-14', 'CHECK 1216 RESTAURANT DEPOT', -1134.60, 'check', '1216', 'RESTAURANT DEPOT', 'unmatched');

-- ACH/Electronic payments (some matched, some unmatched)
INSERT INTO bank_transactions (reconciliation_id, posting_date, description, amount, transaction_type, vendor_hint, match_status) VALUES
(2, '2026-01-02', 'ACH DEBIT VERIZON WIRELESS', -245.89, 'ach', 'VERIZON', 'matched'),
(2, '2026-01-05', 'ACH DEBIT PECO ENERGY', -2145.67, 'ach', 'PECO', 'unmatched'),
(2, '2026-01-10', 'ACH DEBIT STATE FARM INS', -650.00, 'ach', 'STATE FARM', 'unmatched'),
(2, '2026-01-13', 'ACH DEBIT QUICKBOOKS PAYROLL', -4450.00, 'ach', 'QUICKBOOKS', 'unmatched'),
(2, '2026-01-15', 'ACH DEBIT PGW GAS', -389.45, 'ach', 'PGW', 'unmatched'),
(2, '2026-01-15', 'ACH DEBIT WASTE MGMT', -189.00, 'ach', 'WASTE MGMT', 'unmatched');

-- Debit card purchases (all unmatched - need review)
INSERT INTO bank_transactions (reconciliation_id, posting_date, description, amount, transaction_type, vendor_hint, match_status) VALUES
(2, '2026-01-04', 'DEBIT CARD PURCHASE HOME DEPOT', -234.56, 'debit', 'HOME DEPOT', 'unmatched'),
(2, '2026-01-09', 'DEBIT CARD PURCHASE COSTCO', -445.67, 'debit', 'COSTCO', 'unmatched'),
(2, '2026-01-11', 'DEBIT CARD PURCHASE AMAZON', -89.99, 'debit', 'AMAZON', 'unmatched'),
(2, '2026-01-14', 'DEBIT CARD PURCHASE STAPLES', -156.78, 'debit', 'STAPLES', 'unmatched');

-- Fees (unmatched)
INSERT INTO bank_transactions (reconciliation_id, posting_date, description, amount, transaction_type, match_status) VALUES
(2, '2026-01-15', 'SERVICE CHARGE', -25.00, 'fee', 'unmatched'),
(2, '2026-01-15', 'ANALYSIS SERVICE CHARGE', -115.00, 'fee', 'unmatched'),
(2, '2026-01-08', 'ACH RETURN FEE', -35.00, 'fee', 'unmatched');
