-- 002_posts_image.sql
-- Add an optional image URL to posts so creators can attach a cover image
-- when publishing from the dashboard.

ALTER TABLE `posts`
  ADD COLUMN `image_url` varchar(1000) DEFAULT NULL AFTER `preview`;
