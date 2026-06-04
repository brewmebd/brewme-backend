CREATE DATABASE IF NOT EXISTS `brewme`;
USE `brewme`;

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

-- 1. Categories
DROP TABLE IF EXISTS `categories`;
CREATE TABLE `categories` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(60) NOT NULL,
  `slug` varchar(60) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_categories_name` (`name`),
  UNIQUE KEY `uq_categories_slug` (`slug`)
) ENGINE=InnoDB AUTO_INCREMENT=13 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. Users
DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `full_name` varchar(120) NOT NULL,
  `username` varchar(60) NOT NULL,
  `email` varchar(190) NOT NULL,
  `password_hash` varchar(255) NOT NULL,
  `bio` text DEFAULT NULL,
  `category_id` int(10) unsigned DEFAULT NULL,
  `avatar_url` varchar(500) DEFAULT NULL,
  `email_verified_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_users_username` (`username`),
  UNIQUE KEY `uq_users_email` (`email`),
  KEY `idx_users_category` (`category_id`),
  CONSTRAINT `fk_users_category` FOREIGN KEY (`category_id`) REFERENCES `categories` (`id`) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=12 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. Supporters
DROP TABLE IF EXISTS `supporters`;
CREATE TABLE `supporters` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `display_name` varchar(120) DEFAULT NULL,
  `email` varchar(190) DEFAULT NULL,
  `is_anonymous` tinyint(1) NOT NULL DEFAULT 0,
  `first_supported_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_supporter_creator_email` (`user_id`,`email`),
  KEY `idx_supporters_user` (`user_id`),
  CONSTRAINT `fk_supporters_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 4. Membership Tiers
DROP TABLE IF EXISTS `membership_tiers`;
CREATE TABLE `membership_tiers` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `name` varchar(80) NOT NULL,
  `price` decimal(10,2) NOT NULL,
  `billing_period` enum('monthly','yearly') NOT NULL DEFAULT 'monthly',
  `color` varchar(40) DEFAULT NULL,
  `sort_order` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_tiers_user` (`user_id`,`sort_order`),
  CONSTRAINT `fk_tiers_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 5. Goals
DROP TABLE IF EXISTS `goals`;
CREATE TABLE `goals` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `label` varchar(200) NOT NULL,
  `target_amount` decimal(10,2) NOT NULL,
  `current_amount` decimal(10,2) NOT NULL DEFAULT 0.00,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_goals_user_active` (`user_id`,`is_active`),
  CONSTRAINT `fk_goals_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6. Donations
DROP TABLE IF EXISTS `donations`;
CREATE TABLE `donations` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `supporter_id` bigint(20) unsigned DEFAULT NULL,
  `display_name` varchar(120) DEFAULT NULL,
  `is_anonymous` tinyint(1) NOT NULL DEFAULT 0,
  `cups` smallint(5) unsigned NOT NULL DEFAULT 1,
  `amount` decimal(10,2) NOT NULL,
  `currency` char(3) NOT NULL DEFAULT 'USD',
  `message` text DEFAULT NULL,
  `status` enum('pending','succeeded','failed','refunded') NOT NULL DEFAULT 'succeeded',
  `stripe_charge_id` varchar(255) DEFAULT NULL,
  `reply_message` text DEFAULT NULL,
  `replied_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_donations_user_created` (`user_id`,`created_at`),
  KEY `idx_donations_supporter` (`supporter_id`),
  CONSTRAINT `fk_donations_supporter` FOREIGN KEY (`supporter_id`) REFERENCES `supporters` (`id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `fk_donations_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 7. Memberships
DROP TABLE IF EXISTS `memberships`;
CREATE TABLE `memberships` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `tier_id` bigint(20) unsigned NOT NULL,
  `supporter_id` bigint(20) unsigned DEFAULT NULL,
  `display_name` varchar(120) DEFAULT NULL,
  `amount` decimal(10,2) NOT NULL,
  `status` enum('active','past_due','canceled','paused') NOT NULL DEFAULT 'active',
  `stripe_subscription_id` varchar(255) DEFAULT NULL,
  `started_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `current_period_end` timestamp NULL DEFAULT NULL,
  `canceled_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_memberships_user` (`user_id`,`status`),
  KEY `idx_memberships_tier` (`tier_id`),
  KEY `idx_memberships_supporter` (`supporter_id`),
  CONSTRAINT `fk_memberships_supporter` FOREIGN KEY (`supporter_id`) REFERENCES `supporters` (`id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `fk_memberships_tier` FOREIGN KEY (`tier_id`) REFERENCES `membership_tiers` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_memberships_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 8. Notification Settings
DROP TABLE IF EXISTS `notification_settings`;
CREATE TABLE `notification_settings` (
  `user_id` bigint(20) unsigned NOT NULL,
  `new_supporter` tinyint(1) NOT NULL DEFAULT 1,
  `new_message` tinyint(1) NOT NULL DEFAULT 1,
  `weekly_report` tinyint(1) NOT NULL DEFAULT 0,
  `marketing_emails` tinyint(1) NOT NULL DEFAULT 0,
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`user_id`),
  CONSTRAINT `fk_notif_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 9. Payout Accounts
DROP TABLE IF EXISTS `payout_accounts`;
CREATE TABLE `payout_accounts` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `provider` enum('stripe','paypal') NOT NULL DEFAULT 'stripe',
  `external_account_id` varchar(255) DEFAULT NULL,
  `card_last4` char(4) DEFAULT NULL,
  `is_connected` tinyint(1) NOT NULL DEFAULT 0,
  `connected_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_payout_user_provider` (`user_id`,`provider`),
  CONSTRAINT `fk_payout_acct_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 10. Payouts
DROP TABLE IF EXISTS `payouts`;
CREATE TABLE `payouts` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `reference` varchar(20) NOT NULL,
  `amount` decimal(10,2) NOT NULL,
  `currency` char(3) NOT NULL DEFAULT 'USD',
  `method` enum('stripe','paypal','bank') NOT NULL DEFAULT 'stripe',
  `status` enum('pending','completed','failed') NOT NULL DEFAULT 'pending',
  `payout_date` date NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_payout_reference` (`reference`),
  KEY `idx_payouts_user_date` (`user_id`,`payout_date`),
  CONSTRAINT `fk_payouts_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 11. Posts
DROP TABLE IF EXISTS `posts`;
CREATE TABLE `posts` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `title` varchar(255) NOT NULL,
  `body` mediumtext DEFAULT NULL,
  `preview` varchar(500) DEFAULT NULL,
  `visibility` enum('public','members') NOT NULL DEFAULT 'public',
  `status` enum('draft','published') NOT NULL DEFAULT 'published',
  `likes_count` int(10) unsigned NOT NULL DEFAULT 0,
  `comments_count` int(10) unsigned NOT NULL DEFAULT 0,
  `published_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_posts_user_published` (`user_id`,`published_at`),
  KEY `idx_posts_visibility` (`visibility`),
  CONSTRAINT `fk_posts_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 12. Post Media
DROP TABLE IF EXISTS `post_media`;
CREATE TABLE `post_media` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `post_id` bigint(20) unsigned NOT NULL,
  `url` varchar(500) NOT NULL,
  `media_type` enum('image','video','file') NOT NULL DEFAULT 'image',
  `sort_order` tinyint(3) unsigned NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_media_post` (`post_id`,`sort_order`),
  CONSTRAINT `fk_media_post` FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 13. Post Comments
DROP TABLE IF EXISTS `post_comments`;
CREATE TABLE `post_comments` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `post_id` bigint(20) unsigned NOT NULL,
  `supporter_id` bigint(20) unsigned DEFAULT NULL,
  `author_name` varchar(120) NOT NULL,
  `body` text NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_comments_post` (`post_id`,`created_at`),
  KEY `fk_comments_supporter` (`supporter_id`),
  CONSTRAINT `fk_comments_post` FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_comments_supporter` FOREIGN KEY (`supporter_id`) REFERENCES `supporters` (`id`) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 14. Social Links
DROP TABLE IF EXISTS `social_links`;
CREATE TABLE `social_links` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `platform` enum('twitter','instagram','youtube','website','tiktok','other') NOT NULL,
  `url` varchar(500) NOT NULL,
  `sort_order` tinyint(3) unsigned NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_social_user_platform` (`user_id`,`platform`),
  CONSTRAINT `fk_social_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 15. Tier Perks
DROP TABLE IF EXISTS `tier_perks`;
CREATE TABLE `tier_perks` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `tier_id` bigint(20) unsigned NOT NULL,
  `perk_text` varchar(200) NOT NULL,
  `sort_order` tinyint(3) unsigned NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_perks_tier` (`tier_id`,`sort_order`),
  CONSTRAINT `fk_perks_tier` FOREIGN KEY (`tier_id`) REFERENCES `membership_tiers` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=13 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 16. Views
DROP VIEW IF EXISTS `creator_earnings`;
CREATE VIEW `creator_earnings` AS
select 
  `u`.`id` AS `user_id`,
  coalesce((select sum(`donations`.`amount`) from `donations` where `donations`.`user_id` = `u`.`id` and `donations`.`status` = 'succeeded'),0) + 
  coalesce((select sum(`memberships`.`amount`) from `memberships` where `memberships`.`user_id` = `u`.`id` and `memberships`.`status` = 'active'),0) AS `total_earned`,
  coalesce((select sum(`payouts`.`amount`) from `payouts` where `payouts`.`user_id` = `u`.`id` and `payouts`.`status` = 'completed'),0) AS `total_paid_out`,
  (select count(0) from `supporters` where `supporters`.`user_id` = `u`.`id`) AS `supporter_count` 
from `users` `u`;

DROP VIEW IF EXISTS `supporter_feed`;
CREATE VIEW `supporter_feed` AS
select 
  `d`.`user_id` AS `user_id`,
  'coffee' AS `support_type`,
  coalesce(`d`.`display_name`,'Anonymous') AS `display_name`,
  `d`.`message` AS `message`,
  `d`.`cups` AS `cups`,
  `d`.`amount` AS `amount`,
  `d`.`reply_message` is not null AS `replied`,
  `d`.`created_at` AS `created_at` 
from `donations` `d` 
where `d`.`status` = 'succeeded' 
union all 
select 
  `m`.`user_id` AS `user_id`,
  'membership' AS `support_type`,
  coalesce(`m`.`display_name`,'Anonymous') AS `display_name`,
  NULL AS `message`,
  0 AS `cups`,
  `m`.`amount` AS `amount`,
  0 AS `replied`,
  `m`.`started_at` AS `created_at` 
from `memberships` `m`;

-- 17. Seed Data
INSERT INTO `categories` (`id`, `name`, `slug`, `created_at`) VALUES
(1, 'Digital Art', 'digital-art', '2026-06-03 03:49:32'),
(2, 'Music', 'music', '2026-06-03 03:49:32'),
(3, 'Writing', 'writing', '2026-06-03 03:49:32'),
(4, 'Podcasting', 'podcasting', '2026-06-03 03:49:32'),
(5, 'Open Source', 'open-source', '2026-06-03 03:49:32'),
(6, 'Education', 'education', '2026-06-03 03:49:32'),
(7, 'Gaming', 'gaming', '2026-06-03 03:49:32'),
(8, 'Photography', 'photography', '2026-06-03 03:49:32'),
(9, 'Film', 'film', '2026-06-03 03:49:32'),
(10, 'Cooking', 'cooking', '2026-06-03 03:49:32'),
(11, 'Tech', 'tech', '2026-06-03 03:49:32'),
(12, 'Fitness', 'fitness', '2026-06-03 03:49:32');

INSERT INTO `users` (`id`, `full_name`, `username`, `email`, `password_hash`, `bio`, `category_id`, `avatar_url`, `email_verified_at`, `created_at`, `updated_at`) VALUES
(1, 'Sarah Chen', 'sarahchen', 'sarah@example.com', '$2y$10$REPLACE_WITH_REAL_HASH', 'Digital artist creating illustrations, tutorials, and design resources. I share weekly art process videos and exclusive assets for my supporters.', 1, NULL, NULL, '2026-06-03 03:49:32', '2026-06-03 03:49:32'),
(2, 'Alex Rivera', 'alexrivera', 'alex@example.com', '$2y$10$REPLACE_WITH_REAL_HASH', 'Indie musician crafting lo-fi beats and ambient soundscapes. Every coffee helps me produce my next album.', 2, NULL, NULL, '2026-06-03 03:49:32', '2026-06-03 03:49:32'),
(3, 'Jordan Park', 'jordanpark', 'jordan@example.com', '$2y$10$REPLACE_WITH_REAL_HASH', 'Fiction writer and poet. I publish weekly short stories and poetry for my supporters.', 3, NULL, NULL, '2026-06-03 03:49:32', '2026-06-03 03:49:32'),
(4, 'Maya Johnson', 'mayajohnson', 'maya@example.com', '$2y$10$REPLACE_WITH_REAL_HASH', 'Host of \"The Creative Hour\" — a weekly podcast interviewing artists, designers, and creative entrepreneurs.', 4, NULL, NULL, '2026-06-03 03:49:32', '2026-06-03 03:49:32'),
(5, 'Leo Tanaka', 'leotanaka', 'leo@example.com', '$2y$10$REPLACE_WITH_REAL_HASH', 'Full-stack developer maintaining open source tools.', 5, NULL, NULL, '2026-06-03 03:49:32', '2026-06-03 03:49:32'),
(6, 'Priya Sharma', 'priyasharma', 'priya@example.com', '$2y$10$REPLACE_WITH_REAL_HASH', 'Teaching math and science through visual explainers.', 6, NULL, NULL, '2026-06-03 03:49:32', '2026-06-03 03:49:32'),
(7, 'Chris Lee', 'chrislee', 'chris@example.com', '$2y$10$REPLACE_WITH_REAL_HASH', 'Retro game streamer and speedrun enthusiast.', 7, NULL, NULL, '2026-06-03 03:49:32', '2026-06-03 03:49:32'),
(8, 'Nina Costa', 'ninacosta', 'nina@example.com', '$2y$10$REPLACE_WITH_REAL_HASH', 'Street and travel photographer based in Lisbon.', 8, NULL, NULL, '2026-06-03 03:49:32', '2026-06-03 03:49:32'),
(11, 'Saikat Sikder', 'mdhsaikats', 'saikatsikder2911@gmail.com', '$2a$10$CEp0TPyjgyUYyrwsnEoa0O1j.B.B8miZ2EaoNCkr9YvJXG.axH58m', 'I am a gamer', 1, '/uploads/avatar/22c87521f496c5e30f8bb4234537a669.png', NULL, '2026-06-04 03:35:57', '2026-06-04 03:35:57');

INSERT INTO `supporters` (`id`, `user_id`, `display_name`, `email`, `is_anonymous`, `first_supported_at`) VALUES
(1, 1, 'Emily Rodriguez', 'emily@example.com', 0, '2026-06-03 03:49:33'),
(2, 1, 'Marcus Thompson', 'marcus@example.com', 0, '2026-06-03 03:49:33'),
(3, 1, NULL, NULL, 1, '2026-06-03 03:49:33'),
(4, 1, 'Lily Kim', 'lily@example.com', 0, '2026-06-03 03:49:33'),
(5, 1, 'James Wilson', 'james@example.com', 0, '2026-06-03 03:49:33');

INSERT INTO `donations` (`id`, `user_id`, `supporter_id`, `display_name`, `is_anonymous`, `cups`, `amount`, `currency`, `message`, `status`, `stripe_charge_id`, `reply_message`, `replied_at`, `created_at`) VALUES
(1, 1, 1, 'Emily R.', 0, 3, '15.00', 'USD', 'Love your work! Keep creating amazing art! 🎨', 'succeeded', NULL, NULL, NULL, '2026-06-03 03:49:33'),
(2, 1, 2, 'Marcus T.', 0, 1, '5.00', 'USD', 'Your tutorials saved my portfolio. Thank you!', 'succeeded', NULL, 'Thank you so much, Marcus!', '2026-06-03 03:49:33', '2026-06-03 03:49:33'),
(3, 1, 3, NULL, 1, 5, '25.00', 'USD', NULL, 'succeeded', NULL, NULL, NULL, '2026-06-03 03:49:33'),
(4, 1, 4, 'Lily K.', 0, 2, '10.00', 'USD', 'Supporting your journey! Can\'t wait for more content.', 'succeeded', NULL, NULL, NULL, '2026-06-03 03:49:33'),
(5, 1, 5, 'James W.', 0, 1, '5.00', 'USD', 'Incredible artist. Honored to support.', 'succeeded', NULL, 'Honored to have you!', '2026-06-03 03:49:33', '2026-06-03 03:49:33');

INSERT INTO `goals` (`id`, `user_id`, `label`, `target_amount`, `current_amount`, `is_active`, `created_at`) VALUES
(1, 1, 'New iPad Pro for drawing streams', '500.00', '340.00', 1, '2026-06-03 03:49:32');

INSERT INTO `membership_tiers` (`id`, `user_id`, `name`, `price`, `billing_period`, `color`, `sort_order`, `is_active`, `created_at`) VALUES
(1, 1, 'Coffee Supporter', '5.00', 'monthly', 'bg-brew-yellow-light', 0, 1, '2026-06-03 03:49:32'),
(2, 1, 'Gold Member', '15.00', 'monthly', 'bg-brew-yellow/10', 1, 1, '2026-06-03 03:49:32'),
(3, 1, 'Platinum Patron', '50.00', 'monthly', 'bg-brew-yellow/20', 2, 1, '2026-06-03 03:49:32');

INSERT INTO `memberships` (`id`, `user_id`, `tier_id`, `supporter_id`, `display_name`, `amount`, `status`, `stripe_subscription_id`, `started_at`, `current_period_end`, `canceled_at`) VALUES
(1, 1, 2, NULL, 'Aria Patel', '20.00', 'active', NULL, '2026-06-03 03:49:33', NULL, NULL);

INSERT INTO `notification_settings` (`user_id`, `new_supporter`, `new_message`, `weekly_report`, `marketing_emails`, `updated_at`) VALUES
(1, 1, 1, 0, 0, '2026-06-03 03:49:32');

INSERT INTO `payout_accounts` (`id`, `user_id`, `provider`, `external_account_id`, `card_last4`, `is_connected`, `connected_at`) VALUES
(1, 1, 'stripe', NULL, '4242', 1, '2026-06-03 03:49:32');

INSERT INTO `payouts` (`id`, `user_id`, `reference`, `amount`, `currency`, `method`, `status`, `payout_date`, `created_at`) VALUES
(1, 1, 'PO-001', '450.00', 'USD', 'stripe', 'completed', '2026-09-01', '2026-06-03 03:49:33'),
(2, 1, 'PO-002', '380.00', 'USD', 'stripe', 'completed', '2026-08-01', '2026-06-03 03:49:33'),
(3, 1, 'PO-003', '420.00', 'USD', 'stripe', 'completed', '2026-07-01', '2026-06-03 03:49:33'),
(4, 1, 'PO-004', '310.00', 'USD', 'stripe', 'completed', '2026-06-01', '2026-06-03 03:49:33'),
(5, 1, 'PO-005', '220.00', 'USD', 'stripe', 'completed', '2026-05-01', '2026-06-03 03:49:33');

INSERT INTO `posts` (`id`, `user_id`, `title`, `body`, `preview`, `visibility`, `status`, `likes_count`, `comments_count`, `published_at`, `created_at`, `updated_at`) VALUES
(1, 1, 'Behind the scenes: My latest illustration process', NULL, 'A deep dive into how I created the ocean sunset piece that went viral on Instagram...', 'public', 'published', 47, 12, '2026-04-22 10:00:00', '2026-06-03 03:49:33', '2026-06-03 03:49:33'),
(2, 1, 'Exclusive: Full PSD files for January collection', NULL, 'Download all 12 high-res illustration files including layered PSDs and brush presets...', 'members', 'published', 89, 23, '2026-04-18 10:00:00', '2026-06-03 03:49:33', '2026-06-03 03:49:33'),
(3, 1, 'Monthly Q&A Recap — Your questions answered', NULL, 'Thank you for all the amazing questions this month! Here are my answers to the top 20...', 'public', 'published', 34, 8, '2026-04-10 10:00:00', '2026-06-03 03:49:33', '2026-06-03 03:49:33'),
(4, 1, 'Brush pack v3.0 — Premium Procreate brushes', NULL, 'My custom brush pack updated with 15 new brushes optimized for iPad Pro and Apple Pencil...', 'members', 'published', 156, 45, '2026-04-05 10:00:00', '2026-06-03 03:49:33', '2026-06-03 03:49:33'),
(5, 1, 'New series announcement: Digital Landscapes', NULL, 'Excited to announce my new series focused on creating stunning digital landscapes...', 'public', 'published', 72, 19, '2026-03-28 10:00:00', '2026-06-03 03:49:33', '2026-06-03 03:49:33');

INSERT INTO `social_links` (`id`, `user_id`, `platform`, `url`, `sort_order`) VALUES
(1, 1, 'twitter', 'https://twitter.com/sarahchen', 0),
(2, 1, 'instagram', 'https://instagram.com/sarahchen', 1),
(3, 1, 'website', 'https://sarahchen.art', 2);

INSERT INTO `tier_perks` (`id`, `tier_id`, `perk_text`, `sort_order`) VALUES
(1, 1, 'Access to supporters feed', 0),
(2, 1, 'Name in supporter wall', 1),
(3, 1, 'Monthly newsletter', 2),
(4, 2, 'All Coffee Supporter perks', 0),
(5, 2, 'Exclusive posts & downloads', 1),
(6, 2, 'Monthly Q&A access', 2),
(7, 2, 'Early content access', 3),
(8, 3, 'All Gold Member perks', 0),
(9, 3, '1-on-1 monthly call', 1),
(10, 3, 'Custom illustration request', 2),
(11, 3, 'Behind-the-scenes access', 3),
(12, 3, 'Credits in all works', 4);

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;