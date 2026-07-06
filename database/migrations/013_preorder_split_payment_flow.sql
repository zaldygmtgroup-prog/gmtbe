ALTER TABLE preorders
  MODIFY COLUMN payment_status ENUM('unpaid','pending','partial','paid','expired','failed','refund') NOT NULL DEFAULT 'unpaid',
  ADD COLUMN payment_mode ENUM('full','split') NOT NULL DEFAULT 'full' AFTER total_komisi,
  ADD COLUMN dp_proof VARCHAR(500) NULL AFTER payment_proof,
  ADD COLUMN remaining_proof VARCHAR(500) NULL AFTER dp_proof,
  ADD COLUMN last_payment_stage VARCHAR(20) NULL AFTER remaining_proof;
