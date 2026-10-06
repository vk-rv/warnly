CREATE TABLE event (
    pid SMALLINT UNSIGNED NOT NULL,
    event_id CHAR(32) NOT NULL,
    gid BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(6) NOT NULL,
    event_day DATE NOT NULL,
    event_hour DATETIME NOT NULL,
    expires_at DATETIME(6) NOT NULL,
    deleted TINYINT UNSIGNED NOT NULL,
    user_id TEXT NOT NULL,
    message LONGTEXT NOT NULL,
    title TEXT NOT NULL,
    payload LONGTEXT NOT NULL,
    PRIMARY KEY (pid, event_id)
) ENGINE=DuckDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_bin;

CREATE TABLE event_tag (
    pid SMALLINT UNSIGNED NOT NULL,
    event_id CHAR(32) NOT NULL,
    ordinal INT UNSIGNED NOT NULL,
    tag_key TEXT NOT NULL,
    tag_value LONGTEXT NOT NULL,
    PRIMARY KEY (pid, event_id, ordinal)
) ENGINE=DuckDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_bin;
