ALTER TABLE preorders
  ADD COLUMN payment_proof VARCHAR(500) NULL AFTER payment_status;
