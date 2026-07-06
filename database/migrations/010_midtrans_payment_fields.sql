ALTER TABLE preorders
  ADD COLUMN payment_status ENUM('unpaid','pending','partial','paid','expired','failed','refund') NOT NULL DEFAULT 'unpaid' AFTER status,
  ADD COLUMN payment_url VARCHAR(500) NULL AFTER payment_status,
  ADD COLUMN payment_token VARCHAR(255) NULL AFTER payment_url,
  ADD COLUMN midtrans_order_id VARCHAR(100) NULL AFTER payment_token,
  ADD COLUMN midtrans_transaction_id VARCHAR(100) NULL AFTER midtrans_order_id,
  ADD KEY idx_preorders_midtrans_order_id (midtrans_order_id);
