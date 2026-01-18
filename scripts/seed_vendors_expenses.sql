-- Seed data for vendors and expenses
-- Realistic restaurant vendors and recurring expenses
-- Categories use comma-separated format for multi-select

-- Vendors
INSERT INTO vendors (name, category, description) VALUES
-- Food suppliers (often supply multiple categories)
('Sysco Foods', 'Food,Paper,Supplies', 'Primary food distributor'),
('US Foods', 'Food,Supplies', 'Secondary food distributor'),
('Restaurant Depot', 'Food,Supplies,Paper', 'Cash and carry wholesale'),
('Local Produce Co', 'Food', 'Fresh local vegetables and fruits'),

-- Meat & Seafood
('Prime Meats Inc', 'Meat,Food', 'Premium beef and pork supplier'),
('Atlantic Seafood', 'Seafood,Food', 'Fresh fish and shellfish'),

-- Beverages
('Coca-Cola Bottling', 'Beverages', 'Soft drinks and fountain supplies'),
('Premium Coffee Roasters', 'Beverages', 'Coffee and tea supplier'),
('Valley Dairy', 'Beverages,Food', 'Milk, cream, and dairy products'),

-- Paper & Supplies
('WebstaurantStore', 'Paper,Supplies,Equipment', 'Disposables and smallwares'),
('Restaurant Supply Co', 'Supplies,Equipment', 'Kitchen supplies and equipment'),

-- Utilities
('City Electric', 'Utilities', 'Electricity provider'),
('City Gas & Water', 'Utilities', 'Gas and water services'),
('Verizon Business', 'Utilities,Services', 'Phone and internet'),
('Waste Management', 'Utilities,Services', 'Trash and recycling'),

-- Services
('ABC Pest Control', 'Services', 'Monthly pest control'),
('Hood Cleaning Pros', 'Services', 'Quarterly hood cleaning'),
('Linen Service Co', 'Services', 'Weekly linen and uniform service'),
('POS Systems Inc', 'Services,Equipment', 'Point of sale support'),

-- Insurance & Licenses
('Restaurant Insurance Group', 'Insurance', 'Business liability insurance'),
('State Liquor Authority', 'Licenses', 'Liquor license renewal'),
('Health Department', 'Licenses', 'Health permit'),

-- Equipment & Repairs
('Commercial Kitchen Repair', 'Repairs/Reno,Services,Equipment', 'Equipment repair service'),
('HVAC Solutions', 'Repairs/Reno,Services', 'Heating and cooling maintenance'),

-- Rent & Loans
('Main Street Properties', 'Rent', 'Building lease'),
('First National Bank', 'Loan', 'Business loan payment'),

-- Marketing
('Local Newspaper', 'Marketing', 'Print advertising'),
('Social Media Ads', 'Marketing', 'Facebook and Instagram ads'),

-- Delivery Services
('DoorDash Merchant', 'Delivery', 'Delivery service fees'),
('Grubhub Merchant', 'Delivery', 'Delivery service fees'),
('UberEats Merchant', 'Delivery', 'Delivery service fees'),

-- Taxes
('State Tax Authority', 'Taxes', 'Sales tax payments'),
('IRS', 'Taxes', 'Federal tax payments');

-- Expenses
-- Pattern: Regular recurring expenses + variable food costs
-- Food orders typically 2-3x per week, utilities monthly, rent monthly, etc.
-- Due dates: Food suppliers net-7, utilities net-15, rent due on 1st

INSERT INTO expenses (date, vendor_id, amount, invoice_number, status, payment_type, check_number, date_opened, due_date, date_paid, notes) VALUES
-- October 2024
('2024-10-01', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 2847.50, 'SYS-100124-001', 'paid', 'check', '1001', '2024-10-01', '2024-10-08', '2024-10-08', ''),
('2024-10-01', (SELECT id FROM vendors WHERE name = 'Main Street Properties'), 4500.00, 'RENT-1024', 'paid', 'check', '1002', '2024-10-01', '2024-10-01', '2024-10-01', 'October rent'),
('2024-10-03', (SELECT id FROM vendors WHERE name = 'Prime Meats Inc'), 1234.80, 'PM-8834', 'paid', 'check', '1003', '2024-10-03', '2024-10-10', '2024-10-10', ''),
('2024-10-04', (SELECT id FROM vendors WHERE name = 'US Foods'), 1567.25, 'USF-445521', 'paid', 'check', '1004', '2024-10-04', '2024-10-11', '2024-10-11', ''),
('2024-10-05', (SELECT id FROM vendors WHERE name = 'Atlantic Seafood'), 876.40, 'AS-2241', 'paid', 'check', '1005', '2024-10-05', '2024-10-12', '2024-10-12', ''),
('2024-10-07', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 445.60, 'LPC-1007', 'paid', 'cash', '', '2024-10-07', '2024-10-07', '2024-10-07', ''),
('2024-10-08', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 3124.80, 'SYS-100824-001', 'paid', 'check', '1006', '2024-10-08', '2024-10-15', '2024-10-15', ''),
('2024-10-10', (SELECT id FROM vendors WHERE name = 'Coca-Cola Bottling'), 567.80, 'CC-78456', 'paid', 'check', '1007', '2024-10-10', '2024-10-17', '2024-10-17', ''),
('2024-10-11', (SELECT id FROM vendors WHERE name = 'Prime Meats Inc'), 1456.20, 'PM-8901', 'paid', 'check', '1008', '2024-10-11', '2024-10-18', '2024-10-18', ''),
('2024-10-12', (SELECT id FROM vendors WHERE name = 'WebstaurantStore'), 345.90, 'WRS-445566', 'paid', 'credit', '', '2024-10-12', '2024-10-12', '2024-10-12', ''),
('2024-10-14', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 398.40, 'LPC-1014', 'paid', 'cash', '', '2024-10-14', '2024-10-14', '2024-10-14', ''),
('2024-10-15', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 2956.70, 'SYS-101524-001', 'paid', 'check', '1009', '2024-10-15', '2024-10-22', '2024-10-22', ''),
('2024-10-15', (SELECT id FROM vendors WHERE name = 'City Electric'), 1245.80, 'ELEC-1024', 'paid', 'check', '1010', '2024-10-15', '2024-10-30', '2024-10-15', 'October electric'),
('2024-10-15', (SELECT id FROM vendors WHERE name = 'City Gas & Water'), 456.70, 'GW-1024', 'paid', 'check', '1011', '2024-10-15', '2024-10-30', '2024-10-15', 'October gas/water'),
('2024-10-16', (SELECT id FROM vendors WHERE name = 'ABC Pest Control'), 125.00, 'PC-1024', 'paid', 'check', '1012', '2024-10-16', '2024-10-23', '2024-10-23', 'Monthly service'),
('2024-10-17', (SELECT id FROM vendors WHERE name = 'Atlantic Seafood'), 945.60, 'AS-2298', 'paid', 'check', '1013', '2024-10-17', '2024-10-24', '2024-10-24', ''),
('2024-10-18', (SELECT id FROM vendors WHERE name = 'US Foods'), 1823.40, 'USF-446892', 'paid', 'check', '1014', '2024-10-18', '2024-10-25', '2024-10-25', ''),
('2024-10-20', (SELECT id FROM vendors WHERE name = 'Linen Service Co'), 178.50, 'LSC-1020', 'paid', 'check', '1015', '2024-10-20', '2024-10-27', '2024-10-27', 'Weekly linen'),
('2024-10-21', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 467.80, 'LPC-1021', 'paid', 'cash', '', '2024-10-21', '2024-10-21', '2024-10-21', ''),
('2024-10-22', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 3245.90, 'SYS-102224-001', 'paid', 'check', '1016', '2024-10-22', '2024-10-29', '2024-10-29', ''),
('2024-10-23', (SELECT id FROM vendors WHERE name = 'Premium Coffee Roasters'), 289.40, 'PCR-5567', 'paid', 'check', '1017', '2024-10-23', '2024-10-30', '2024-10-30', ''),
('2024-10-24', (SELECT id FROM vendors WHERE name = 'Prime Meats Inc'), 1567.30, 'PM-9023', 'paid', 'check', '1018', '2024-10-24', '2024-10-31', '2024-10-31', ''),
('2024-10-25', (SELECT id FROM vendors WHERE name = 'Verizon Business'), 189.99, 'VZ-1024', 'paid', 'debit', '', '2024-10-25', '2024-11-10', '2024-10-25', 'Phone/internet'),
('2024-10-27', (SELECT id FROM vendors WHERE name = 'Linen Service Co'), 178.50, 'LSC-1027', 'paid', 'check', '1019', '2024-10-27', '2024-11-03', '2024-11-03', 'Weekly linen'),
('2024-10-28', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 512.30, 'LPC-1028', 'paid', 'cash', '', '2024-10-28', '2024-10-28', '2024-10-28', ''),
('2024-10-29', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 2789.60, 'SYS-102924-001', 'paid', 'check', '1020', '2024-10-29', '2024-11-05', '2024-11-05', ''),
('2024-10-30', (SELECT id FROM vendors WHERE name = 'Waste Management'), 345.00, 'WM-1024', 'paid', 'check', '1021', '2024-10-30', '2024-11-15', '2024-11-06', 'October trash'),
('2024-10-31', (SELECT id FROM vendors WHERE name = 'Atlantic Seafood'), 1123.40, 'AS-2356', 'paid', 'check', '1022', '2024-10-31', '2024-11-07', '2024-11-07', ''),

-- November 2024
('2024-11-01', (SELECT id FROM vendors WHERE name = 'Main Street Properties'), 4500.00, 'RENT-1124', 'paid', 'check', '1023', '2024-11-01', '2024-11-01', '2024-11-01', 'November rent'),
('2024-11-01', (SELECT id FROM vendors WHERE name = 'First National Bank'), 1250.00, 'LOAN-1124', 'paid', 'check', '1024', '2024-11-01', '2024-11-01', '2024-11-01', 'Loan payment'),
('2024-11-01', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 3156.80, 'SYS-110124-001', 'paid', 'check', '1025', '2024-11-01', '2024-11-08', '2024-11-08', ''),
('2024-11-03', (SELECT id FROM vendors WHERE name = 'Linen Service Co'), 178.50, 'LSC-1103', 'paid', 'check', '1026', '2024-11-03', '2024-11-10', '2024-11-10', 'Weekly linen'),
('2024-11-04', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 478.90, 'LPC-1104', 'paid', 'cash', '', '2024-11-04', '2024-11-04', '2024-11-04', ''),
('2024-11-05', (SELECT id FROM vendors WHERE name = 'Prime Meats Inc'), 1345.60, 'PM-9134', 'paid', 'check', '1027', '2024-11-05', '2024-11-12', '2024-11-12', ''),
('2024-11-06', (SELECT id FROM vendors WHERE name = 'US Foods'), 1678.90, 'USF-448234', 'paid', 'check', '1028', '2024-11-06', '2024-11-13', '2024-11-13', ''),
('2024-11-07', (SELECT id FROM vendors WHERE name = 'Atlantic Seafood'), 867.40, 'AS-2401', 'paid', 'check', '1029', '2024-11-07', '2024-11-14', '2024-11-14', ''),
('2024-11-08', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 2934.50, 'SYS-110824-001', 'paid', 'check', '1030', '2024-11-08', '2024-11-15', '2024-11-15', ''),
('2024-11-10', (SELECT id FROM vendors WHERE name = 'Linen Service Co'), 178.50, 'LSC-1110', 'paid', 'check', '1031', '2024-11-10', '2024-11-17', '2024-11-17', 'Weekly linen'),
('2024-11-11', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 534.60, 'LPC-1111', 'paid', 'cash', '', '2024-11-11', '2024-11-11', '2024-11-11', ''),
('2024-11-12', (SELECT id FROM vendors WHERE name = 'Coca-Cola Bottling'), 612.40, 'CC-79012', 'paid', 'check', '1032', '2024-11-12', '2024-11-19', '2024-11-19', ''),
('2024-11-13', (SELECT id FROM vendors WHERE name = 'Prime Meats Inc'), 1456.80, 'PM-9201', 'paid', 'check', '1033', '2024-11-13', '2024-11-20', '2024-11-20', ''),
('2024-11-14', (SELECT id FROM vendors WHERE name = 'ABC Pest Control'), 125.00, 'PC-1124', 'paid', 'check', '1034', '2024-11-14', '2024-11-21', '2024-11-21', 'Monthly service'),
('2024-11-15', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 3267.40, 'SYS-111524-001', 'paid', 'check', '1035', '2024-11-15', '2024-11-22', '2024-11-22', ''),
('2024-11-15', (SELECT id FROM vendors WHERE name = 'City Electric'), 1312.50, 'ELEC-1124', 'paid', 'check', '1036', '2024-11-15', '2024-11-30', '2024-11-15', 'November electric'),
('2024-11-15', (SELECT id FROM vendors WHERE name = 'City Gas & Water'), 523.40, 'GW-1124', 'paid', 'check', '1037', '2024-11-15', '2024-11-30', '2024-11-15', 'November gas/water'),
('2024-11-17', (SELECT id FROM vendors WHERE name = 'Linen Service Co'), 178.50, 'LSC-1117', 'paid', 'check', '1038', '2024-11-17', '2024-11-24', '2024-11-24', 'Weekly linen'),
('2024-11-18', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 489.70, 'LPC-1118', 'paid', 'cash', '', '2024-11-18', '2024-11-18', '2024-11-18', ''),
('2024-11-19', (SELECT id FROM vendors WHERE name = 'US Foods'), 1923.60, 'USF-449567', 'paid', 'check', '1039', '2024-11-19', '2024-11-26', '2024-11-26', ''),
('2024-11-20', (SELECT id FROM vendors WHERE name = 'Atlantic Seafood'), 978.30, 'AS-2467', 'paid', 'check', '1040', '2024-11-20', '2024-11-27', '2024-11-27', ''),
('2024-11-21', (SELECT id FROM vendors WHERE name = 'Prime Meats Inc'), 1678.90, 'PM-9278', 'paid', 'check', '1041', '2024-11-21', '2024-11-28', '2024-11-28', 'Extra order for Thanksgiving'),
('2024-11-22', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 4123.50, 'SYS-112224-001', 'paid', 'check', '1042', '2024-11-22', '2024-11-29', '2024-11-29', 'Thanksgiving stock'),
('2024-11-24', (SELECT id FROM vendors WHERE name = 'Linen Service Co'), 245.00, 'LSC-1124', 'paid', 'check', '1043', '2024-11-24', '2024-12-01', '2024-12-01', 'Extra linens for holiday'),
('2024-11-25', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 678.40, 'LPC-1125', 'paid', 'cash', '', '2024-11-25', '2024-11-25', '2024-11-25', 'Thanksgiving produce'),
('2024-11-25', (SELECT id FROM vendors WHERE name = 'Verizon Business'), 189.99, 'VZ-1124', 'paid', 'debit', '', '2024-11-25', '2024-12-10', '2024-11-25', 'Phone/internet'),
('2024-11-29', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 2567.80, 'SYS-112924-001', 'paid', 'check', '1044', '2024-11-29', '2024-12-06', '2024-12-06', ''),
('2024-11-30', (SELECT id FROM vendors WHERE name = 'Waste Management'), 345.00, 'WM-1124', 'paid', 'check', '1045', '2024-11-30', '2024-12-15', '2024-12-07', 'November trash'),
('2024-11-30', (SELECT id FROM vendors WHERE name = 'Hood Cleaning Pros'), 475.00, 'HCP-Q424', 'paid', 'check', '1046', '2024-11-30', '2024-12-07', '2024-12-07', 'Quarterly hood cleaning'),

-- December 2024
('2024-12-01', (SELECT id FROM vendors WHERE name = 'Main Street Properties'), 4500.00, 'RENT-1224', 'paid', 'check', '1047', '2024-12-01', '2024-12-01', '2024-12-01', 'December rent'),
('2024-12-01', (SELECT id FROM vendors WHERE name = 'First National Bank'), 1250.00, 'LOAN-1224', 'paid', 'check', '1048', '2024-12-01', '2024-12-01', '2024-12-01', 'Loan payment'),
('2024-12-02', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 3234.60, 'SYS-120224-001', 'paid', 'check', '1049', '2024-12-02', '2024-12-09', '2024-12-09', ''),
('2024-12-03', (SELECT id FROM vendors WHERE name = 'Prime Meats Inc'), 1567.40, 'PM-9345', 'paid', 'check', '1050', '2024-12-03', '2024-12-10', '2024-12-10', ''),
('2024-12-04', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 512.80, 'LPC-1204', 'paid', 'cash', '', '2024-12-04', '2024-12-04', '2024-12-04', ''),
('2024-12-05', (SELECT id FROM vendors WHERE name = 'Atlantic Seafood'), 1045.60, 'AS-2534', 'paid', 'check', '1051', '2024-12-05', '2024-12-12', '2024-12-12', ''),
('2024-12-06', (SELECT id FROM vendors WHERE name = 'US Foods'), 1834.70, 'USF-450912', 'paid', 'check', '1052', '2024-12-06', '2024-12-13', '2024-12-13', ''),
('2024-12-08', (SELECT id FROM vendors WHERE name = 'Linen Service Co'), 178.50, 'LSC-1208', 'paid', 'check', '1053', '2024-12-08', '2024-12-15', '2024-12-15', 'Weekly linen'),
('2024-12-09', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 3456.80, 'SYS-120924-001', 'paid', 'check', '1054', '2024-12-09', '2024-12-16', '2024-12-16', ''),
('2024-12-10', (SELECT id FROM vendors WHERE name = 'Premium Coffee Roasters'), 312.60, 'PCR-5689', 'paid', 'check', '1055', '2024-12-10', '2024-12-17', '2024-12-17', ''),
('2024-12-11', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 478.90, 'LPC-1211', 'paid', 'cash', '', '2024-12-11', '2024-12-11', '2024-12-11', ''),
('2024-12-12', (SELECT id FROM vendors WHERE name = 'Prime Meats Inc'), 1789.30, 'PM-9412', 'paid', 'check', '1056', '2024-12-12', '2024-12-19', '2024-12-19', 'Holiday orders'),
('2024-12-13', (SELECT id FROM vendors WHERE name = 'ABC Pest Control'), 125.00, 'PC-1224', 'paid', 'check', '1057', '2024-12-13', '2024-12-20', '2024-12-20', 'Monthly service'),
('2024-12-14', (SELECT id FROM vendors WHERE name = 'Coca-Cola Bottling'), 678.90, 'CC-79456', 'paid', 'check', '1058', '2024-12-14', '2024-12-21', '2024-12-21', ''),
('2024-12-15', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 4567.80, 'SYS-121524-001', 'paid', 'check', '1059', '2024-12-15', '2024-12-22', '2024-12-22', 'Holiday stock'),
('2024-12-15', (SELECT id FROM vendors WHERE name = 'City Electric'), 1456.70, 'ELEC-1224', 'paid', 'check', '1060', '2024-12-15', '2024-12-31', '2024-12-15', 'December electric'),
('2024-12-15', (SELECT id FROM vendors WHERE name = 'City Gas & Water'), 612.30, 'GW-1224', 'paid', 'check', '1061', '2024-12-15', '2024-12-31', '2024-12-15', 'December gas/water'),
('2024-12-15', (SELECT id FROM vendors WHERE name = 'Linen Service Co'), 245.00, 'LSC-1215', 'paid', 'check', '1062', '2024-12-15', '2024-12-22', '2024-12-22', 'Extra holiday linens'),
('2024-12-16', (SELECT id FROM vendors WHERE name = 'Restaurant Insurance Group'), 2450.00, 'RIG-Q125', 'paid', 'check', '1063', '2024-12-16', '2024-12-16', '2024-12-16', 'Quarterly insurance'),
('2024-12-17', (SELECT id FROM vendors WHERE name = 'Atlantic Seafood'), 1234.50, 'AS-2601', 'paid', 'check', '1064', '2024-12-17', '2024-12-24', '2024-12-24', ''),
('2024-12-18', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 567.40, 'LPC-1218', 'paid', 'cash', '', '2024-12-18', '2024-12-18', '2024-12-18', ''),
('2024-12-19', (SELECT id FROM vendors WHERE name = 'US Foods'), 2134.60, 'USF-452345', 'paid', 'check', '1065', '2024-12-19', '2024-12-26', '2024-12-26', ''),
('2024-12-20', (SELECT id FROM vendors WHERE name = 'Prime Meats Inc'), 2145.80, 'PM-9489', 'paid', 'check', '1066', '2024-12-20', '2024-12-27', '2024-12-27', 'Christmas orders'),
('2024-12-21', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 3789.40, 'SYS-122124-001', 'paid', 'check', '1067', '2024-12-21', '2024-12-28', '2024-12-28', ''),
('2024-12-22', (SELECT id FROM vendors WHERE name = 'Linen Service Co'), 178.50, 'LSC-1222', 'paid', 'check', '1068', '2024-12-22', '2024-12-29', '2024-12-29', 'Weekly linen'),
('2024-12-23', (SELECT id FROM vendors WHERE name = 'WebstaurantStore'), 456.70, 'WRS-447823', 'paid', 'credit', '', '2024-12-23', '2024-12-23', '2024-12-23', 'Holiday supplies'),
('2024-12-25', (SELECT id FROM vendors WHERE name = 'Verizon Business'), 189.99, 'VZ-1224', 'paid', 'debit', '', '2024-12-25', '2025-01-10', '2024-12-25', 'Phone/internet'),
('2024-12-27', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 2678.90, 'SYS-122724-001', 'paid', 'check', '1069', '2024-12-27', '2025-01-03', '2025-01-03', ''),
('2024-12-28', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 489.60, 'LPC-1228', 'paid', 'cash', '', '2024-12-28', '2024-12-28', '2024-12-28', ''),
('2024-12-30', (SELECT id FROM vendors WHERE name = 'Atlantic Seafood'), 987.60, 'AS-2678', 'paid', 'check', '1070', '2024-12-30', '2025-01-06', '2025-01-06', ''),
('2024-12-31', (SELECT id FROM vendors WHERE name = 'Waste Management'), 345.00, 'WM-1224', 'paid', 'check', '1071', '2024-12-31', '2025-01-15', '2025-01-07', 'December trash'),

-- January 2025
('2025-01-01', (SELECT id FROM vendors WHERE name = 'Main Street Properties'), 4500.00, 'RENT-0125', 'paid', 'check', '1072', '2025-01-01', '2025-01-01', '2025-01-01', 'January rent'),
('2025-01-01', (SELECT id FROM vendors WHERE name = 'First National Bank'), 1250.00, 'LOAN-0125', 'paid', 'check', '1073', '2025-01-01', '2025-01-01', '2025-01-01', 'Loan payment'),
('2025-01-02', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 2567.80, 'SYS-010225-001', 'paid', 'check', '1074', '2025-01-02', '2025-01-09', '2025-01-09', ''),
('2025-01-03', (SELECT id FROM vendors WHERE name = 'Prime Meats Inc'), 1234.60, 'PM-9556', 'paid', 'check', '1075', '2025-01-03', '2025-01-10', '2025-01-10', ''),
('2025-01-05', (SELECT id FROM vendors WHERE name = 'Linen Service Co'), 178.50, 'LSC-0105', 'paid', 'check', '1076', '2025-01-05', '2025-01-12', '2025-01-12', 'Weekly linen'),
('2025-01-06', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 398.70, 'LPC-0106', 'paid', 'cash', '', '2025-01-06', '2025-01-06', '2025-01-06', ''),
('2025-01-07', (SELECT id FROM vendors WHERE name = 'US Foods'), 1567.40, 'USF-453678', 'paid', 'check', '1077', '2025-01-07', '2025-01-14', '2025-01-14', ''),
('2025-01-08', (SELECT id FROM vendors WHERE name = 'Atlantic Seafood'), 756.80, 'AS-2745', 'paid', 'check', '1078', '2025-01-08', '2025-01-15', '2025-01-15', ''),
('2025-01-09', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 2890.40, 'SYS-010925-001', 'paid', 'check', '1079', '2025-01-09', '2025-01-16', '2025-01-16', ''),
('2025-01-10', (SELECT id FROM vendors WHERE name = 'ABC Pest Control'), 125.00, 'PC-0125', 'paid', 'check', '1080', '2025-01-10', '2025-01-17', '2025-01-17', 'Monthly service'),
('2025-01-12', (SELECT id FROM vendors WHERE name = 'Linen Service Co'), 178.50, 'LSC-0112', 'paid', 'check', '1081', '2025-01-12', '2025-01-19', '2025-01-19', 'Weekly linen'),
('2025-01-13', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 445.60, 'LPC-0113', 'paid', 'cash', '', '2025-01-13', '2025-01-13', '2025-01-13', ''),
('2025-01-14', (SELECT id FROM vendors WHERE name = 'Prime Meats Inc'), 1345.70, 'PM-9623', 'paid', 'check', '1082', '2025-01-14', '2025-01-21', '2025-01-21', ''),
('2025-01-15', (SELECT id FROM vendors WHERE name = 'City Electric'), 1189.40, 'ELEC-0125', 'paid', 'check', '1083', '2025-01-15', '2025-01-31', '2025-01-15', 'January electric'),
('2025-01-15', (SELECT id FROM vendors WHERE name = 'City Gas & Water'), 589.60, 'GW-0125', 'paid', 'check', '1084', '2025-01-15', '2025-01-31', '2025-01-15', 'January gas/water'),

-- December 2025 (partial month to match other seed data)
('2025-12-01', (SELECT id FROM vendors WHERE name = 'Main Street Properties'), 4750.00, 'RENT-1225', 'paid', 'check', '1201', '2025-12-01', '2025-12-01', '2025-12-01', 'December rent'),
('2025-12-01', (SELECT id FROM vendors WHERE name = 'First National Bank'), 1250.00, 'LOAN-1225', 'paid', 'check', '1202', '2025-12-01', '2025-12-01', '2025-12-01', 'Loan payment'),
('2025-12-02', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 3456.70, 'SYS-120225-001', 'paid', 'check', '1203', '2025-12-02', '2025-12-09', '2025-12-09', ''),
('2025-12-03', (SELECT id FROM vendors WHERE name = 'Prime Meats Inc'), 1678.90, 'PM-10234', 'paid', 'check', '1204', '2025-12-03', '2025-12-10', '2025-12-10', ''),
('2025-12-05', (SELECT id FROM vendors WHERE name = 'Atlantic Seafood'), 1123.40, 'AS-3012', 'paid', 'check', '1205', '2025-12-05', '2025-12-12', '2025-12-12', ''),
('2025-12-08', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 3789.60, 'SYS-120825-001', 'paid', 'check', '1206', '2025-12-08', '2025-12-15', '2025-12-15', ''),
('2025-12-10', (SELECT id FROM vendors WHERE name = 'US Foods'), 2134.50, 'USF-478901', 'paid', 'check', '1207', '2025-12-10', '2025-12-17', '2025-12-17', ''),
('2025-12-12', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 567.80, 'LPC-1212', 'paid', 'cash', '', '2025-12-12', '2025-12-12', '2025-12-12', ''),
('2025-12-15', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 4234.60, 'SYS-121525-001', 'paid', 'check', '1208', '2025-12-15', '2025-12-22', '2025-12-22', 'Holiday stock'),
('2025-12-15', (SELECT id FROM vendors WHERE name = 'City Electric'), 1534.80, 'ELEC-1225', 'paid', 'check', '1209', '2025-12-15', '2025-12-31', '2025-12-15', 'December electric'),
('2025-12-15', (SELECT id FROM vendors WHERE name = 'City Gas & Water'), 678.90, 'GW-1225', 'paid', 'check', '1210', '2025-12-15', '2025-12-31', '2025-12-15', 'December gas/water'),
('2025-12-18', (SELECT id FROM vendors WHERE name = 'Prime Meats Inc'), 2345.60, 'PM-10345', 'paid', 'check', '1211', '2025-12-18', '2025-12-25', '2025-12-25', 'Christmas orders'),
('2025-12-20', (SELECT id FROM vendors WHERE name = 'Atlantic Seafood'), 1567.80, 'AS-3089', 'paid', 'check', '1212', '2025-12-20', '2025-12-27', '2025-12-27', ''),
('2025-12-22', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 3567.40, 'SYS-122225-001', 'paid', 'check', '1213', '2025-12-22', '2025-12-29', '2025-12-29', ''),
('2025-12-23', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 678.90, 'LPC-1223', 'paid', 'cash', '', '2025-12-23', '2025-12-23', '2025-12-23', ''),
('2025-12-28', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 2890.30, 'SYS-122825-001', 'paid', 'check', '1214', '2025-12-28', '2026-01-04', '2026-01-04', ''),
('2025-12-30', (SELECT id FROM vendors WHERE name = 'Waste Management'), 365.00, 'WM-1225', 'paid', 'check', '1215', '2025-12-30', '2026-01-15', '2026-01-06', 'December trash'),

-- January 2026
('2026-01-01', (SELECT id FROM vendors WHERE name = 'Main Street Properties'), 4750.00, 'RENT-0126', 'paid', 'check', '1216', '2026-01-01', '2026-01-01', '2026-01-01', 'January rent'),
('2026-01-01', (SELECT id FROM vendors WHERE name = 'First National Bank'), 1250.00, 'LOAN-0126', 'paid', 'check', '1217', '2026-01-01', '2026-01-01', '2026-01-01', 'Loan payment'),
('2026-01-02', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 2678.90, 'SYS-010226-001', 'paid', 'check', '1218', '2026-01-02', '2026-01-09', '2026-01-09', ''),
('2026-01-03', (SELECT id FROM vendors WHERE name = 'Prime Meats Inc'), 1345.60, 'PM-10412', 'paid', 'check', '1219', '2026-01-03', '2026-01-10', '2026-01-10', ''),
('2026-01-05', (SELECT id FROM vendors WHERE name = 'Linen Service Co'), 185.00, 'LSC-0105-26', 'paid', 'check', '1220', '2026-01-05', '2026-01-12', '2026-01-12', 'Weekly linen'),
('2026-01-06', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 423.70, 'LPC-0106-26', 'paid', 'cash', '', '2026-01-06', '2026-01-06', '2026-01-06', ''),
('2026-01-07', (SELECT id FROM vendors WHERE name = 'Atlantic Seafood'), 867.40, 'AS-3156', 'paid', 'check', '1221', '2026-01-07', '2026-01-14', '2026-01-14', ''),
('2026-01-08', (SELECT id FROM vendors WHERE name = 'US Foods'), 1678.90, 'USF-480234', 'paid', 'check', '1222', '2026-01-08', '2026-01-15', '2026-01-15', ''),
-- Unpaid expenses with realistic due dates
('2026-01-09', (SELECT id FROM vendors WHERE name = 'Sysco Foods'), 2956.80, 'SYS-010926-001', 'not_paid', '', '', '2026-01-09', '2026-01-16', NULL, ''),
('2026-01-10', (SELECT id FROM vendors WHERE name = 'ABC Pest Control'), 130.00, 'PC-0126', 'not_paid', '', '', '2026-01-10', '2026-01-17', NULL, 'Monthly service'),
('2026-01-12', (SELECT id FROM vendors WHERE name = 'Linen Service Co'), 185.00, 'LSC-0112-26', 'not_paid', '', '', '2026-01-12', '2026-01-19', NULL, 'Weekly linen'),
('2026-01-13', (SELECT id FROM vendors WHERE name = 'Local Produce Co'), 478.60, 'LPC-0113-26', 'not_paid', '', '', '2026-01-13', '2026-01-13', NULL, ''),
('2026-01-14', (SELECT id FROM vendors WHERE name = 'Prime Meats Inc'), 1456.70, 'PM-10489', 'not_paid', '', '', '2026-01-14', '2026-01-21', NULL, ''),
('2026-01-15', (SELECT id FROM vendors WHERE name = 'City Electric'), 1267.90, 'ELEC-0126', 'not_paid', '', '', '2026-01-15', '2026-01-31', NULL, 'January electric'),
('2026-01-15', (SELECT id FROM vendors WHERE name = 'City Gas & Water'), 612.40, 'GW-0126', 'not_paid', '', '', '2026-01-15', '2026-01-31', NULL, 'January gas/water');
