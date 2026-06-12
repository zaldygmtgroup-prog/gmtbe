ALTER TABLE detail_users
  ADD COLUMN agent_program_type VARCHAR(50) NULL,
  ADD COLUMN agent_motivation TEXT NULL,
  ADD COLUMN referral_source VARCHAR(80) NULL,
  ADD COLUMN referral_name VARCHAR(120) NULL,
  ADD COLUMN referral_other VARCHAR(255) NULL,
  ADD COLUMN target_product VARCHAR(255) NULL;
