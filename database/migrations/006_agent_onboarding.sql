CREATE TABLE IF NOT EXISTS agent_onboarding_videos (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  slug VARCHAR(100) NOT NULL,
  title VARCHAR(255) NOT NULL,
  description TEXT NULL,
  video_url VARCHAR(255) NOT NULL,
  duration_seconds INT NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  is_required TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE KEY idx_agent_onboarding_videos_slug (slug)
);

CREATE TABLE IF NOT EXISTS agent_onboarding_progress (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  video_id BIGINT UNSIGNED NOT NULL,
  status ENUM('not_started','in_progress','completed') NOT NULL DEFAULT 'not_started',
  watched_seconds INT NOT NULL DEFAULT 0,
  completed_at DATETIME(3) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE KEY idx_user_video (user_id, video_id),
  CONSTRAINT fk_agent_onboarding_progress_user
    FOREIGN KEY (user_id) REFERENCES users(id)
    ON DELETE CASCADE,
  CONSTRAINT fk_agent_onboarding_progress_video
    FOREIGN KEY (video_id) REFERENCES agent_onboarding_videos(id)
    ON DELETE CASCADE
);
