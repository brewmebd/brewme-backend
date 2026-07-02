-- 001_membership_replies.sql
-- Allow creators to reply to memberships (previously only donations were replyable).
-- Adds reply storage to the memberships table and teaches the supporter_feed view
-- to report a membership's real reply state.

ALTER TABLE `memberships`
  ADD COLUMN `reply_message` text DEFAULT NULL,
  ADD COLUMN `replied_at` timestamp NULL DEFAULT NULL;

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
  `m`.`reply_message` is not null AS `replied`,
  `m`.`started_at` AS `created_at`
from `memberships` `m`;
