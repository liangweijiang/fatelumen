CREATE TABLE IF NOT EXISTS system_settings (
    `key` VARCHAR(64) NOT NULL,
    `value` VARCHAR(512) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
