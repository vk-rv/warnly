CREATE TABLE IF NOT EXISTS `user` (
  `id` int NOT NULL AUTO_INCREMENT,
  `email` varchar(50) NOT NULL,
  `name` varchar(100) NOT NULL,
  `surname` varchar(100) NOT NULL,
  `username` varchar(39) NOT NULL UNIQUE,
  `password` char(60) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `email` (`email`)
);

CREATE TABLE IF NOT EXISTS `project` (
  `id` int NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL,
  `name` varchar(32) NOT NULL,
  `user_id` int NOT NULL,
  `team_id` int NOT NULL,
  `platform` tinyint NOT NULL,
  `project_key` varchar(7) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`,`team_id`),
  UNIQUE KEY `project_key` (`project_key`)
);

CREATE TABLE IF NOT EXISTS `issue` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `uuid` binary(16) NOT NULL,
  `first_seen` datetime NOT NULL,
  `last_seen` datetime NOT NULL,
  `hash` char(32) NOT NULL,
  `message` varchar(128) DEFAULT NULL,
  `view` varchar(255) DEFAULT NULL,
  `num_comments` int,
  `project_id` int NOT NULL,
  `priority` tinyint NOT NULL,
  `error_type` varchar(512) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_uuid` (`uuid`),
  KEY `idx_hash` (`hash`)
);

CREATE TABLE IF NOT EXISTS `team` (
  `id` int NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL,
  `name` varchar(255) NOT NULL,
  `owner_id` int NOT NULL,
  PRIMARY KEY (`id`)
);


INSERT INTO `team` (`created_at`, `name`, `owner_id`) VALUES
(NOW(), 'default', 1);


CREATE TABLE IF NOT EXISTS `team_relation` (
  `id` int NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL,
  `team_id` int NOT NULL,
  `user_id` int NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `team_id` (`team_id`,`user_id`)
);

INSERT INTO `team_relation` (`created_at`, `team_id`, `user_id`) VALUES
(NOW(), 1, 1);

CREATE TABLE IF NOT EXISTS `message` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `issue_id` BIGINT NOT NULL,
  `user_id` INT NOT NULL,
  `content` TEXT NOT NULL,
  `created_at` DATETIME NOT NULL DEFAULT NOW(),
  FOREIGN KEY (issue_id) REFERENCES issue(id),
  FOREIGN KEY (user_id) REFERENCES user(id),
  KEY `idx_created_at` (`created_at`)
);

CREATE TABLE IF NOT EXISTS `mention` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `message_id` INT NOT NULL,
  `mentioned_user_id` INT NOT NULL,
  `created_at` DATETIME NOT NULL DEFAULT NOW(),
  KEY `idx_message_id` (`message_id`),
  KEY `idx_mentioned_user_id` (`mentioned_user_id`),
  KEY `idx_created_at` (`created_at`)
);

CREATE TABLE IF NOT EXISTS `message_view` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `message_id` INT NOT NULL,
  `user_id` INT NOT NULL,
  `viewed_at` DATETIME NOT NULL DEFAULT NOW(),
  FOREIGN KEY (`message_id`) REFERENCES `message`(`id`),
  FOREIGN KEY (`user_id`) REFERENCES `user`(`id`),
  UNIQUE KEY `unique_message_view` (`message_id`, `user_id`)
);

CREATE TABLE IF NOT EXISTS `issue_assignment` (
  `issue_id` BIGINT NOT NULL PRIMARY KEY,
  `assigned_to_user_id` BIGINT DEFAULT NULL,
  `assigned_to_team_id` BIGINT DEFAULT NULL,
  `assigned_by_user_id` BIGINT NOT NULL,
  `assigned_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS `issue_assignment_history` (
  `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `issue_id` BIGINT NOT NULL,
  `old_assigned_to_user_id` BIGINT DEFAULT NULL,
  `old_assigned_to_team_id` BIGINT DEFAULT NULL,
  `new_assigned_to_user_id` BIGINT DEFAULT NULL,
  `new_assigned_to_team_id` BIGINT DEFAULT NULL,
  `assigned_by_user_id` BIGINT DEFAULT NULL, -- NULL for automatic assignments
  `assigned_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  KEY `idx_iah_issue_id` (`issue_id`),
  KEY `idx_iah_assigned_at` (`assigned_at`)
);
