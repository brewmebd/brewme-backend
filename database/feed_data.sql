INSERT INTO "public"."categories" ("id", "name", "slug", "created_at") VALUES
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
INSERT INTO "public"."users" ("id", "full_name", "username", "email", "password_hash", "bio", "category_id", "avatar_url", "email_verified_at", "created_at", "updated_at") VALUES
(1, 'Sarah Chen', 'sarahchen', 'sarah@example.com', '$2y$10$REPLACE_WITH_REAL_HASH', 'Digital artist creating illustrations, tutorials, and design resources. I share weekly art process videos and exclusive assets for my supporters.', 1, NULL, NULL, '2026-06-03 03:49:32', '2026-06-03 03:49:32'),
(2, 'Alex Rivera', 'alexrivera', 'alex@example.com', '$2y$10$REPLACE_WITH_REAL_HASH', 'Indie musician crafting lo-fi beats and ambient soundscapes. Every coffee helps me produce my next album.', 2, NULL, NULL, '2026-06-03 03:49:32', '2026-06-03 03:49:32'),
(3, 'Jordan Park', 'jordanpark', 'jordan@example.com', '$2y$10$REPLACE_WITH_REAL_HASH', 'Fiction writer and poet. I publish weekly short stories and poetry for my supporters.', 3, NULL, NULL, '2026-06-03 03:49:32', '2026-06-03 03:49:32'),
(4, 'Maya Johnson', 'mayajohnson', 'maya@example.com', '$2y$10$REPLACE_WITH_REAL_HASH', 'Host of \"The Creative Hour\" — a weekly podcast interviewing artists, designers, and creative entrepreneurs.', 4, NULL, NULL, '2026-06-03 03:49:32', '2026-06-03 03:49:32'),
(5, 'Leo Tanaka', 'leotanaka', 'leo@example.com', '$2y$10$REPLACE_WITH_REAL_HASH', 'Full-stack developer maintaining open source tools.', 5, NULL, NULL, '2026-06-03 03:49:32', '2026-06-03 03:49:32'),
(6, 'Priya Sharma', 'priyasharma', 'priya@example.com', '$2y$10$REPLACE_WITH_REAL_HASH', 'Teaching math and science through visual explainers.', 6, NULL, NULL, '2026-06-03 03:49:32', '2026-06-03 03:49:32'),
(7, 'Chris Lee', 'chrislee', 'chris@example.com', '$2y$10$REPLACE_WITH_REAL_HASH', 'Retro game streamer and speedrun enthusiast.', 7, NULL, NULL, '2026-06-03 03:49:32', '2026-06-03 03:49:32'),
(8, 'Nina Costa', 'ninacosta', 'nina@example.com', '$2y$10$REPLACE_WITH_REAL_HASH', 'Street and travel photographer based in Lisbon.', 8, NULL, NULL, '2026-06-03 03:49:32', '2026-06-03 03:49:32'),
(11, 'Saikat Sikder', 'mdhsaikats', 'saikatsikder2911@gmail.com', '$2a$10$CEp0TPyjgyUYyrwsnEoa0O1j.B.B8miZ2EaoNCkr9YvJXG.axH58m', 'I am a gamer', 1, '/uploads/avatar/22c87521f496c5e30f8bb4234537a669.png', NULL, '2026-06-04 03:35:57', '2026-06-04 03:35:57');
INSERT INTO "public"."supporters" ("id", "user_id", "display_name", "email", "is_anonymous", "first_supported_at") VALUES
(1, 1, 'Emily Rodriguez', 'emily@example.com', 'f', '2026-06-03 03:49:33'),
(2, 1, 'Marcus Thompson', 'marcus@example.com', 'f', '2026-06-03 03:49:33'),
(3, 1, NULL, NULL, 't', '2026-06-03 03:49:33'),
(4, 1, 'Lily Kim', 'lily@example.com', 'f', '2026-06-03 03:49:33'),
(5, 1, 'James Wilson', 'james@example.com', 'f', '2026-06-03 03:49:33');
INSERT INTO "public"."membership_tiers" ("id", "user_id", "name", "price", "billing_period", "color", "sort_order", "is_active", "created_at") VALUES
(1, 1, 'Coffee Supporter', 5.00, 'monthly', 'bg-brew-yellow-light', 0, 't', '2026-06-03 03:49:32'),
(2, 1, 'Gold Member', 15.00, 'monthly', 'bg-brew-yellow/10', 1, 't', '2026-06-03 03:49:32'),
(3, 1, 'Platinum Patron', 50.00, 'monthly', 'bg-brew-yellow/20', 2, 't', '2026-06-03 03:49:32');
INSERT INTO "public"."goals" ("id", "user_id", "label", "target_amount", "current_amount", "is_active", "created_at") VALUES
(1, 1, 'New iPad Pro for drawing streams', 500.00, 340.00, 't', '2026-06-03 03:49:32');
INSERT INTO "public"."donations" ("id", "user_id", "supporter_id", "display_name", "is_anonymous", "cups", "amount", "currency", "message", "status", "stripe_charge_id", "reply_message", "replied_at", "created_at") VALUES
(1, 1, 1, 'Emily R.', 'f', 3, 15.00, 'USD', 'Love your work! Keep creating amazing art! 🎨', 'succeeded', NULL, NULL, NULL, '2026-06-03 03:49:33'),
(2, 1, 2, 'Marcus T.', 'f', 1, 5.00, 'USD', 'Your tutorials saved my portfolio. Thank you!', 'succeeded', NULL, 'Thank you so much, Marcus!', '2026-06-03 03:49:33', '2026-06-03 03:49:33'),
(3, 1, 3, NULL, 't', 5, 25.00, 'USD', NULL, 'succeeded', NULL, NULL, NULL, '2026-06-03 03:49:33'),
(4, 1, 4, 'Lily K.', 'f', 2, 10.00, 'USD', 'Supporting your journey! Can''t wait for more content.', 'succeeded', NULL, NULL, NULL, '2026-06-03 03:49:33'),
(5, 1, 5, 'James W.', 'f', 1, 5.00, 'USD', 'Incredible artist. Honored to support.', 'succeeded', NULL, 'Honored to have you!', '2026-06-03 03:49:33', '2026-06-03 03:49:33');
INSERT INTO "public"."memberships" ("id", "user_id", "tier_id", "supporter_id", "display_name", "amount", "status", "stripe_subscription_id", "reply_message", "replied_at", "started_at", "current_period_end", "canceled_at") VALUES
(1, 1, 2, NULL, 'Aria Patel', 20.00, 'active', NULL, NULL, NULL, '2026-06-03 03:49:33', NULL, NULL);
INSERT INTO "public"."notification_settings" ("user_id", "new_supporter", "new_message", "weekly_report", "marketing_emails", "updated_at") VALUES
(1, 't', 't', 'f', 'f', '2026-06-03 03:49:32');
INSERT INTO "public"."payout_accounts" ("id", "user_id", "provider", "external_account_id", "card_last4", "is_connected", "connected_at") VALUES
(1, 1, 'stripe', NULL, '4242', 't', '2026-06-03 03:49:32');
INSERT INTO "public"."payouts" ("id", "user_id", "reference", "amount", "currency", "method", "status", "payout_date", "created_at") VALUES
(1, 1, 'PO-001', 450.00, 'USD', 'stripe', 'completed', '2026-09-01', '2026-06-03 03:49:33'),
(2, 1, 'PO-002', 380.00, 'USD', 'stripe', 'completed', '2026-08-01', '2026-06-03 03:49:33'),
(3, 1, 'PO-003', 420.00, 'USD', 'stripe', 'completed', '2026-07-01', '2026-06-03 03:49:33'),
(4, 1, 'PO-004', 310.00, 'USD', 'stripe', 'completed', '2026-06-01', '2026-06-03 03:49:33'),
(5, 1, 'PO-005', 220.00, 'USD', 'stripe', 'completed', '2026-05-01', '2026-06-03 03:49:33');
INSERT INTO "public"."posts" ("id", "user_id", "title", "body", "preview", "image_url", "visibility", "status", "likes_count", "comments_count", "published_at", "created_at", "updated_at") VALUES
(1, 1, 'Behind the scenes: My latest illustration process', NULL, 'A deep dive into how I created the ocean sunset piece that went viral on Instagram...', NULL, 'public', 'published', 47, 12, '2026-04-22 10:00:00', '2026-06-03 03:49:33', '2026-06-03 03:49:33'),
(2, 1, 'Exclusive: Full PSD files for January collection', NULL, 'Download all 12 high-res illustration files including layered PSDs and brush presets...', NULL, 'members', 'published', 89, 23, '2026-04-18 10:00:00', '2026-06-03 03:49:33', '2026-06-03 03:49:33'),
(3, 1, 'Monthly Q&A Recap — Your questions answered', NULL, 'Thank you for all the amazing questions this month! Here are my answers to the top 20...', NULL, 'public', 'published', 34, 8, '2026-04-10 10:00:00', '2026-06-03 03:49:33', '2026-06-03 03:49:33'),
(4, 1, 'Brush pack v3.0 — Premium Procreate brushes', NULL, 'My custom brush pack updated with 15 new brushes optimized for iPad Pro and Apple Pencil...', NULL, 'members', 'published', 156, 45, '2026-04-05 10:00:00', '2026-06-03 03:49:33', '2026-06-03 03:49:33'),
(5, 1, 'New series announcement: Digital Landscapes', NULL, 'Excited to announce my new series focused on creating stunning digital landscapes...', NULL, 'public', 'published', 72, 19, '2026-03-28 10:00:00', '2026-06-03 03:49:33', '2026-06-03 03:49:33');


INSERT INTO "public"."social_links" ("id", "user_id", "platform", "url", "sort_order") VALUES
(1, 1, 'twitter', 'https://twitter.com/sarahchen', 0),
(2, 1, 'instagram', 'https://instagram.com/sarahchen', 1),
(3, 1, 'website', 'https://sarahchen.art', 2);
INSERT INTO "public"."tier_perks" ("id", "tier_id", "perk_text", "sort_order") VALUES
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


