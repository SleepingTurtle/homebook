-- Seed data for payroll (past 4 weeks)

-- First, create the payroll weeks
INSERT INTO payroll_weeks (id, period_start, period_end) VALUES
(1, '2024-12-23', '2024-12-29'),
(2, '2024-12-30', '2025-01-05'),
(3, '2025-01-06', '2025-01-12'),
(4, '2025-01-13', '2025-01-19');

-- Week 1: Dec 23-29, 2024 (all paid)
INSERT INTO payroll (week_id, employee_id, total_hours, hourly_rate, payment_method, check_number, status, date_paid) VALUES
(1, 1, 40.0, 18.00, 'check', '1001', 'paid', '2024-12-30'),
(1, 2, 38.5, 16.50, 'check', '1002', 'paid', '2024-12-30'),
(1, 3, 35.0, 17.00, 'cash', '', 'paid', '2024-12-30'),
(1, 4, 32.0, 15.50, 'cash', '', 'paid', '2024-12-30'),
(1, 5, 42.0, 19.00, 'check', '1003', 'paid', '2024-12-30'),
(1, 6, 28.0, 16.00, 'cash', '', 'paid', '2024-12-30'),
(1, 8, 36.0, 17.50, 'check', '1004', 'paid', '2024-12-30');

-- Week 2: Dec 30 - Jan 5, 2025 (all paid)
INSERT INTO payroll (week_id, employee_id, total_hours, hourly_rate, payment_method, check_number, status, date_paid) VALUES
(2, 1, 38.0, 18.00, 'check', '1005', 'paid', '2025-01-06'),
(2, 2, 40.0, 16.50, 'check', '1006', 'paid', '2025-01-06'),
(2, 3, 36.0, 17.00, 'cash', '', 'paid', '2025-01-06'),
(2, 4, 30.0, 15.50, 'cash', '', 'paid', '2025-01-06'),
(2, 5, 44.0, 19.00, 'check', '1007', 'paid', '2025-01-06'),
(2, 6, 32.0, 16.00, 'cash', '', 'paid', '2025-01-06'),
(2, 8, 38.0, 17.50, 'check', '1008', 'paid', '2025-01-06');

-- Week 3: Jan 6-12, 2025 (all paid)
INSERT INTO payroll (week_id, employee_id, total_hours, hourly_rate, payment_method, check_number, status, date_paid) VALUES
(3, 1, 40.0, 18.00, 'check', '1009', 'paid', '2025-01-13'),
(3, 2, 36.0, 16.50, 'check', '1010', 'paid', '2025-01-13'),
(3, 3, 38.0, 17.00, 'cash', '', 'paid', '2025-01-13'),
(3, 4, 34.0, 15.50, 'cash', '', 'paid', '2025-01-13'),
(3, 5, 40.0, 19.00, 'check', '1011', 'paid', '2025-01-13'),
(3, 6, 30.0, 16.00, 'cash', '', 'paid', '2025-01-13'),
(3, 8, 36.0, 17.50, 'check', '1012', 'paid', '2025-01-13');

-- Week 4: Jan 13-19, 2025 (partially paid - some still unpaid)
INSERT INTO payroll (week_id, employee_id, total_hours, hourly_rate, payment_method, check_number, status, date_paid) VALUES
(4, 1, 40.0, 18.00, 'check', '1013', 'paid', '2025-01-20'),
(4, 2, 38.0, 16.50, 'check', '1014', 'paid', '2025-01-20'),
(4, 3, 35.0, 17.00, 'cash', '', 'not_paid', NULL),
(4, 4, 32.0, 15.50, 'cash', '', 'not_paid', NULL),
(4, 5, 40.0, 19.00, 'check', '', 'not_paid', NULL),
(4, 6, 28.0, 16.00, 'cash', '', 'not_paid', NULL),
(4, 8, 38.0, 17.50, 'check', '', 'not_paid', NULL);
