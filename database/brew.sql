-- ============================================================================
--  BrewMe — Database Schema
--  A "Buy Me a Coffee" / membership platform where creators receive one-time
--  coffee tips and recurring membership support from their audience.
--
--  Engine : MySQL 8.0+ / MariaDB 10.4+
--  Charset: utf8mb4 (full Unicode incl. emoji used throughout the UI ☕)
--
--  Derived from the React frontend (pages/*, pages/dashboard/*).
--  Money is stored as DECIMAL(10,2). All timestamps are UTC.
-- ============================================================================

DROP DATABASE IF EXISTS brewme;
CREATE DATABASE brewme
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;
USE brewme;

SET FOREIGN_KEY_CHECKS = 0;

-- ============================================================================
--  REFERENCE DATA
-- ============================================================================

-- Creator categories shown in the Explore page filter pills.
CREATE TABLE categories (
  id          INT UNSIGNED NOT NULL AUTO_INCREMENT,
  name        VARCHAR(60)  NOT NULL,
  slug        VARCHAR(60)  NOT NULL,
  created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_categories_name (name),
  UNIQUE KEY uq_categories_slug (slug)
) ENGINE=InnoDB;

-- ============================================================================
--  USERS (CREATORS)
--  A user is a creator account. Their public page lives at brewme.com/{username}.
--  Supporters can tip WITHOUT an account, so supporters are modelled separately
--  (see `supporters`) and are not required to be users.
-- ============================================================================

CREATE TABLE users (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  full_name      VARCHAR(120)    NOT NULL,                 -- SignUp "Full Name" / Settings "Display Name"
  username       VARCHAR(60)      NOT NULL,                -- brewme.com/{username}  (the page URL slug)
  email          VARCHAR(190)     NOT NULL,
  password_hash  VARCHAR(255)     NOT NULL,                -- never store plaintext
  bio            TEXT             NULL,
  category_id    INT UNSIGNED     NULL,                    -- single primary category
  avatar_url     VARCHAR(500)     NULL,                    -- profile photo (JPG/PNG/GIF, max 2MB)
  email_verified_at TIMESTAMP     NULL,
  created_at     TIMESTAMP        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at     TIMESTAMP        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_users_username (username),
  UNIQUE KEY uq_users_email (email),
  KEY idx_users_category (category_id),
  CONSTRAINT fk_users_category
    FOREIGN KEY (category_id) REFERENCES categories (id)
    ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB;

-- Social links rendered on the creator profile header (twitter/instagram/youtube/website).
CREATE TABLE social_links (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id     BIGINT UNSIGNED NOT NULL,
  platform    ENUM('twitter','instagram','youtube','website','tiktok','other') NOT NULL,
  url         VARCHAR(500)    NOT NULL,
  sort_order  TINYINT UNSIGNED NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  UNIQUE KEY uq_social_user_platform (user_id, platform),
  CONSTRAINT fk_social_user
    FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB;

-- Per-creator email/notification preferences (Settings > Notifications toggles).
CREATE TABLE notification_settings (
  user_id           BIGINT UNSIGNED NOT NULL,
  new_supporter     BOOLEAN NOT NULL DEFAULT TRUE,
  new_message       BOOLEAN NOT NULL DEFAULT TRUE,
  weekly_report     BOOLEAN NOT NULL DEFAULT FALSE,
  marketing_emails  BOOLEAN NOT NULL DEFAULT FALSE,
  updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id),
  CONSTRAINT fk_notif_user
    FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB;

-- Stripe Connect / payout destination (Settings > Payouts: "Stripe Connected •••• 4242").
CREATE TABLE payout_accounts (
  id                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id               BIGINT UNSIGNED NOT NULL,
  provider              ENUM('stripe','paypal') NOT NULL DEFAULT 'stripe',
  external_account_id   VARCHAR(255) NULL,                 -- e.g. Stripe acct_xxx
  card_last4            CHAR(4)      NULL,
  is_connected          BOOLEAN      NOT NULL DEFAULT FALSE,
  connected_at          TIMESTAMP    NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_payout_user_provider (user_id, provider),
  CONSTRAINT fk_payout_acct_user
    FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB;

-- ============================================================================
--  GOALS
--  Optional funding goal with a progress bar on the creator profile.
--  `current_amount` is a cached running total; `target_amount` is the goal.
-- ============================================================================

CREATE TABLE goals (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id         BIGINT UNSIGNED NOT NULL,
  label           VARCHAR(200)    NOT NULL,                -- "New iPad Pro for drawing streams"
  target_amount   DECIMAL(10,2)   NOT NULL,
  current_amount  DECIMAL(10,2)   NOT NULL DEFAULT 0.00,
  is_active       BOOLEAN         NOT NULL DEFAULT TRUE,
  created_at      TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_goals_user_active (user_id, is_active),
  CONSTRAINT fk_goals_user
    FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB;

-- ============================================================================
--  MEMBERSHIP TIERS
--  Up to 3 recurring subscription tiers per creator (Memberships dashboard).
-- ============================================================================

CREATE TABLE membership_tiers (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id         BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(80)     NOT NULL,                -- "Coffee Supporter", "Gold Member"
  price           DECIMAL(10,2)   NOT NULL,                -- monthly price
  billing_period  ENUM('monthly','yearly') NOT NULL DEFAULT 'monthly',
  color           VARCHAR(40)     NULL,                    -- UI accent class (e.g. bg-brew-yellow-light)
  sort_order      TINYINT UNSIGNED NOT NULL DEFAULT 0,
  is_active       BOOLEAN         NOT NULL DEFAULT TRUE,
  created_at      TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_tiers_user (user_id, sort_order),
  CONSTRAINT fk_tiers_user
    FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB;

-- Ordered list of perks shown under each tier card.
CREATE TABLE tier_perks (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tier_id     BIGINT UNSIGNED NOT NULL,
  perk_text   VARCHAR(200)    NOT NULL,
  sort_order  TINYINT UNSIGNED NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  KEY idx_perks_tier (tier_id, sort_order),
  CONSTRAINT fk_perks_tier
    FOREIGN KEY (tier_id) REFERENCES membership_tiers (id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB;

-- ============================================================================
--  SUPPORTERS
--  A person who supports a creator. Accounts are NOT required ("No account
--  needed"), so a supporter is identified by the name/email they enter at
--  checkout. Anonymous tips have is_anonymous = TRUE and a NULL email.
--  One supporter row per (creator, email) lets us aggregate repeat supporters.
-- ============================================================================

CREATE TABLE supporters (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id       BIGINT UNSIGNED NOT NULL,                  -- the creator being supported
  display_name  VARCHAR(120)    NULL,                      -- "Emily R." / NULL when anonymous
  email         VARCHAR(190)    NULL,                       -- optional; used to dedupe repeat support
  is_anonymous  BOOLEAN         NOT NULL DEFAULT FALSE,
  first_supported_at TIMESTAMP  NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_supporter_creator_email (user_id, email),
  KEY idx_supporters_user (user_id),
  CONSTRAINT fk_supporters_user
    FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB;

-- ============================================================================
--  DONATIONS (one-time coffee tips)
--  Each row is a "buy N coffees" checkout. amount = cups * price_per_cup ($5).
--  A creator can reply once to a supporter message (Supporters dashboard).
-- ============================================================================

CREATE TABLE donations (
  id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id            BIGINT UNSIGNED NOT NULL,             -- creator receiving the tip
  supporter_id       BIGINT UNSIGNED NULL,                 -- NULL if not deduped/anonymous
  display_name       VARCHAR(120)    NULL,                 -- snapshot of name shown ("Marcus T.")
  is_anonymous       BOOLEAN         NOT NULL DEFAULT FALSE,
  cups               SMALLINT UNSIGNED NOT NULL DEFAULT 1, -- number of coffees
  amount             DECIMAL(10,2)   NOT NULL,             -- total charged
  currency           CHAR(3)         NOT NULL DEFAULT 'USD',
  message            TEXT            NULL,                  -- "Love your work!..."
  status             ENUM('pending','succeeded','failed','refunded') NOT NULL DEFAULT 'succeeded',
  stripe_charge_id   VARCHAR(255)    NULL,
  reply_message      TEXT            NULL,                 -- creator's reply (NULL = not replied)
  replied_at         TIMESTAMP       NULL,
  created_at         TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_donations_user_created (user_id, created_at),
  KEY idx_donations_supporter (supporter_id),
  CONSTRAINT fk_donations_user
    FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_donations_supporter
    FOREIGN KEY (supporter_id) REFERENCES supporters (id)
    ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB;

-- ============================================================================
--  MEMBERSHIPS (recurring subscriptions to a tier)
--  e.g. "joined Gold tier — $10.00/mo". Lifecycle tracked via status.
-- ============================================================================

CREATE TABLE memberships (
  id                     BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id                BIGINT UNSIGNED NOT NULL,         -- creator
  tier_id                BIGINT UNSIGNED NOT NULL,
  supporter_id           BIGINT UNSIGNED NULL,
  display_name           VARCHAR(120)    NULL,             -- "Aria Patel"
  amount                 DECIMAL(10,2)   NOT NULL,         -- monthly amount at time of join
  status                 ENUM('active','past_due','canceled','paused') NOT NULL DEFAULT 'active',
  stripe_subscription_id VARCHAR(255)    NULL,
  started_at             TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
  current_period_end     TIMESTAMP       NULL,
  canceled_at            TIMESTAMP       NULL,
  PRIMARY KEY (id),
  KEY idx_memberships_user (user_id, status),
  KEY idx_memberships_tier (tier_id),
  KEY idx_memberships_supporter (supporter_id),
  CONSTRAINT fk_memberships_user
    FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_memberships_tier
    FOREIGN KEY (tier_id) REFERENCES membership_tiers (id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_memberships_supporter
    FOREIGN KEY (supporter_id) REFERENCES supporters (id)
    ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB;

-- ============================================================================
--  POSTS
--  Creator updates shown on the profile "Posts" tab and Posts dashboard.
--  Visibility gates members-only content. likes/comments are cached counters.
-- ============================================================================

CREATE TABLE posts (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id         BIGINT UNSIGNED NOT NULL,
  title           VARCHAR(255)    NOT NULL,
  body            MEDIUMTEXT      NULL,                    -- full content
  preview         VARCHAR(500)    NULL,                    -- short teaser shown in lists
  visibility      ENUM('public','members') NOT NULL DEFAULT 'public',
  status          ENUM('draft','published') NOT NULL DEFAULT 'published',
  likes_count     INT UNSIGNED    NOT NULL DEFAULT 0,
  comments_count  INT UNSIGNED    NOT NULL DEFAULT 0,
  published_at    TIMESTAMP       NULL,
  created_at      TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_posts_user_published (user_id, published_at),
  KEY idx_posts_visibility (visibility),
  CONSTRAINT fk_posts_user
    FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB;

-- Media attachments for a post ("Add Media" in the post editor).
CREATE TABLE post_media (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  post_id     BIGINT UNSIGNED NOT NULL,
  url         VARCHAR(500)    NOT NULL,
  media_type  ENUM('image','video','file') NOT NULL DEFAULT 'image',
  sort_order  TINYINT UNSIGNED NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  KEY idx_media_post (post_id, sort_order),
  CONSTRAINT fk_media_post
    FOREIGN KEY (post_id) REFERENCES posts (id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB;

-- Comments on a post. Author may be a supporter (guest) — store a name snapshot.
CREATE TABLE post_comments (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  post_id       BIGINT UNSIGNED NOT NULL,
  supporter_id  BIGINT UNSIGNED NULL,
  author_name   VARCHAR(120)    NOT NULL,
  body          TEXT            NOT NULL,
  created_at    TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_comments_post (post_id, created_at),
  CONSTRAINT fk_comments_post
    FOREIGN KEY (post_id) REFERENCES posts (id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_comments_supporter
    FOREIGN KEY (supporter_id) REFERENCES supporters (id)
    ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB;

-- ============================================================================
--  PAYOUTS
--  Money paid out to the creator (Earnings dashboard "Payout History").
--  reference is the human-facing id shown in the UI (e.g. "PO-001").
-- ============================================================================

CREATE TABLE payouts (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id     BIGINT UNSIGNED NOT NULL,
  reference   VARCHAR(20)     NOT NULL,                    -- "PO-001"
  amount      DECIMAL(10,2)   NOT NULL,
  currency    CHAR(3)         NOT NULL DEFAULT 'USD',
  method      ENUM('stripe','paypal','bank') NOT NULL DEFAULT 'stripe',
  status      ENUM('pending','completed','failed') NOT NULL DEFAULT 'pending',
  payout_date DATE            NOT NULL,
  created_at  TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_payout_reference (reference),
  KEY idx_payouts_user_date (user_id, payout_date),
  CONSTRAINT fk_payouts_user
    FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB;

SET FOREIGN_KEY_CHECKS = 1;

-- ============================================================================
--  VIEWS
-- ============================================================================

-- Unified "supporter feed" combining one-time coffees and membership joins,
-- exactly as rendered on the Supporters dashboard and profile Supporters tab.
CREATE OR REPLACE VIEW supporter_feed AS
  SELECT
    d.user_id,
    'coffee'                       AS support_type,
    COALESCE(d.display_name, 'Anonymous') AS display_name,
    d.message,
    d.cups,
    d.amount,
    (d.reply_message IS NOT NULL)  AS replied,
    d.created_at
  FROM donations d
  WHERE d.status = 'succeeded'
  UNION ALL
  SELECT
    m.user_id,
    'membership'                   AS support_type,
    COALESCE(m.display_name, 'Anonymous') AS display_name,
    NULL                           AS message,
    0                              AS cups,
    m.amount,
    FALSE                          AS replied,
    m.started_at                   AS created_at
  FROM memberships m;

-- Per-creator earnings summary used by the dashboard stat cards.
CREATE OR REPLACE VIEW creator_earnings AS
  SELECT
    u.id AS user_id,
    COALESCE((SELECT SUM(amount) FROM donations
              WHERE user_id = u.id AND status = 'succeeded'), 0)
      + COALESCE((SELECT SUM(amount) FROM memberships
                  WHERE user_id = u.id AND status = 'active'), 0)            AS total_earned,
    COALESCE((SELECT SUM(amount) FROM payouts
              WHERE user_id = u.id AND status = 'completed'), 0)             AS total_paid_out,
    (SELECT COUNT(*) FROM supporters WHERE user_id = u.id)                   AS supporter_count
  FROM users u;

-- ============================================================================
--  SEED DATA  (mirrors the mock data in the React frontend)
-- ============================================================================

INSERT INTO categories (name, slug) VALUES
  ('Digital Art','digital-art'), ('Music','music'), ('Writing','writing'),
  ('Podcasting','podcasting'), ('Open Source','open-source'), ('Education','education'),
  ('Gaming','gaming'), ('Photography','photography'), ('Film','film'),
  ('Cooking','cooking'), ('Tech','tech'), ('Fitness','fitness');

-- NOTE: password_hash values below are placeholders ("password") — replace with
-- real bcrypt/argon2 hashes from your backend. Never store plaintext passwords.
INSERT INTO users (full_name, username, email, password_hash, bio, category_id, avatar_url) VALUES
  ('Sarah Chen','sarahchen','sarah@example.com','$2y$10$REPLACE_WITH_REAL_HASH',
     'Digital artist creating illustrations, tutorials, and design resources. I share weekly art process videos and exclusive assets for my supporters.',
     (SELECT id FROM categories WHERE slug='digital-art'), NULL),
  ('Alex Rivera','alexrivera','alex@example.com','$2y$10$REPLACE_WITH_REAL_HASH',
     'Indie musician crafting lo-fi beats and ambient soundscapes. Every coffee helps me produce my next album.',
     (SELECT id FROM categories WHERE slug='music'), NULL),
  ('Jordan Park','jordanpark','jordan@example.com','$2y$10$REPLACE_WITH_REAL_HASH',
     'Fiction writer and poet. I publish weekly short stories and poetry for my supporters.',
     (SELECT id FROM categories WHERE slug='writing'), NULL),
  ('Maya Johnson','mayajohnson','maya@example.com','$2y$10$REPLACE_WITH_REAL_HASH',
     'Host of "The Creative Hour" — a weekly podcast interviewing artists, designers, and creative entrepreneurs.',
     (SELECT id FROM categories WHERE slug='podcasting'), NULL),
  ('Leo Tanaka','leotanaka','leo@example.com','$2y$10$REPLACE_WITH_REAL_HASH',
     'Full-stack developer maintaining open source tools.',
     (SELECT id FROM categories WHERE slug='open-source'), NULL),
  ('Priya Sharma','priyasharma','priya@example.com','$2y$10$REPLACE_WITH_REAL_HASH',
     'Teaching math and science through visual explainers.',
     (SELECT id FROM categories WHERE slug='education'), NULL),
  ('Chris Lee','chrislee','chris@example.com','$2y$10$REPLACE_WITH_REAL_HASH',
     'Retro game streamer and speedrun enthusiast.',
     (SELECT id FROM categories WHERE slug='gaming'), NULL),
  ('Nina Costa','ninacosta','nina@example.com','$2y$10$REPLACE_WITH_REAL_HASH',
     'Street and travel photographer based in Lisbon.',
     (SELECT id FROM categories WHERE slug='photography'), NULL);

-- Social links for Sarah Chen.
INSERT INTO social_links (user_id, platform, url, sort_order)
SELECT id, 'twitter', 'https://twitter.com/sarahchen', 0 FROM users WHERE username='sarahchen'
UNION ALL SELECT id, 'instagram', 'https://instagram.com/sarahchen', 1 FROM users WHERE username='sarahchen'
UNION ALL SELECT id, 'website', 'https://sarahchen.art', 2 FROM users WHERE username='sarahchen';

-- Default settings + payout account + funding goal for Sarah Chen.
INSERT INTO notification_settings (user_id, new_supporter, new_message, weekly_report, marketing_emails)
SELECT id, TRUE, TRUE, FALSE, FALSE FROM users WHERE username='sarahchen';

INSERT INTO payout_accounts (user_id, provider, card_last4, is_connected, connected_at)
SELECT id, 'stripe', '4242', TRUE, CURRENT_TIMESTAMP FROM users WHERE username='sarahchen';

INSERT INTO goals (user_id, label, target_amount, current_amount, is_active)
SELECT id, 'New iPad Pro for drawing streams', 500.00, 340.00, TRUE FROM users WHERE username='sarahchen';

-- Membership tiers + perks for Sarah Chen.
SET @sarah := (SELECT id FROM users WHERE username='sarahchen');

INSERT INTO membership_tiers (user_id, name, price, color, sort_order) VALUES
  (@sarah, 'Coffee Supporter',  5.00, 'bg-brew-yellow-light', 0),
  (@sarah, 'Gold Member',       15.00, 'bg-brew-yellow/10',    1),
  (@sarah, 'Platinum Patron',   50.00, 'bg-brew-yellow/20',    2);

INSERT INTO tier_perks (tier_id, perk_text, sort_order)
SELECT id, 'Access to supporters feed', 0 FROM membership_tiers WHERE user_id=@sarah AND name='Coffee Supporter'
UNION ALL SELECT id, 'Name in supporter wall', 1 FROM membership_tiers WHERE user_id=@sarah AND name='Coffee Supporter'
UNION ALL SELECT id, 'Monthly newsletter', 2 FROM membership_tiers WHERE user_id=@sarah AND name='Coffee Supporter'
UNION ALL SELECT id, 'All Coffee Supporter perks', 0 FROM membership_tiers WHERE user_id=@sarah AND name='Gold Member'
UNION ALL SELECT id, 'Exclusive posts & downloads', 1 FROM membership_tiers WHERE user_id=@sarah AND name='Gold Member'
UNION ALL SELECT id, 'Monthly Q&A access', 2 FROM membership_tiers WHERE user_id=@sarah AND name='Gold Member'
UNION ALL SELECT id, 'Early content access', 3 FROM membership_tiers WHERE user_id=@sarah AND name='Gold Member'
UNION ALL SELECT id, 'All Gold Member perks', 0 FROM membership_tiers WHERE user_id=@sarah AND name='Platinum Patron'
UNION ALL SELECT id, '1-on-1 monthly call', 1 FROM membership_tiers WHERE user_id=@sarah AND name='Platinum Patron'
UNION ALL SELECT id, 'Custom illustration request', 2 FROM membership_tiers WHERE user_id=@sarah AND name='Platinum Patron'
UNION ALL SELECT id, 'Behind-the-scenes access', 3 FROM membership_tiers WHERE user_id=@sarah AND name='Platinum Patron'
UNION ALL SELECT id, 'Credits in all works', 4 FROM membership_tiers WHERE user_id=@sarah AND name='Platinum Patron';

-- Supporters + one-time coffee donations for Sarah Chen.
INSERT INTO supporters (user_id, display_name, email, is_anonymous) VALUES
  (@sarah, 'Emily Rodriguez', 'emily@example.com', FALSE),
  (@sarah, 'Marcus Thompson', 'marcus@example.com', FALSE),
  (@sarah, NULL,              NULL,                 TRUE),
  (@sarah, 'Lily Kim',        'lily@example.com',   FALSE),
  (@sarah, 'James Wilson',    'james@example.com',  FALSE);

INSERT INTO donations (user_id, supporter_id, display_name, is_anonymous, cups, amount, message, reply_message, replied_at)
SELECT @sarah, s.id, 'Emily R.', FALSE, 3, 15.00, 'Love your work! Keep creating amazing art! 🎨', NULL, NULL
  FROM supporters s WHERE s.user_id=@sarah AND s.display_name='Emily Rodriguez'
UNION ALL
SELECT @sarah, s.id, 'Marcus T.', FALSE, 1, 5.00, 'Your tutorials saved my portfolio. Thank you!', 'Thank you so much, Marcus!', CURRENT_TIMESTAMP
  FROM supporters s WHERE s.user_id=@sarah AND s.display_name='Marcus Thompson'
UNION ALL
SELECT @sarah, s.id, NULL, TRUE, 5, 25.00, NULL, NULL, NULL
  FROM supporters s WHERE s.user_id=@sarah AND s.is_anonymous=TRUE
UNION ALL
SELECT @sarah, s.id, 'Lily K.', FALSE, 2, 10.00, 'Supporting your journey! Can''t wait for more content.', NULL, NULL
  FROM supporters s WHERE s.user_id=@sarah AND s.display_name='Lily Kim'
UNION ALL
SELECT @sarah, s.id, 'James W.', FALSE, 1, 5.00, 'Incredible artist. Honored to support.', 'Honored to have you!', CURRENT_TIMESTAMP
  FROM supporters s WHERE s.user_id=@sarah AND s.display_name='James Wilson';

-- An active membership join (e.g. "joined Gold tier").
INSERT INTO memberships (user_id, tier_id, supporter_id, display_name, amount, status)
SELECT @sarah, t.id, NULL, 'Aria Patel', 20.00, 'active'
  FROM membership_tiers t WHERE t.user_id=@sarah AND t.name='Gold Member';

-- Posts for Sarah Chen.
INSERT INTO posts (user_id, title, preview, visibility, status, likes_count, comments_count, published_at) VALUES
  (@sarah, 'Behind the scenes: My latest illustration process',
     'A deep dive into how I created the ocean sunset piece that went viral on Instagram...',
     'public', 'published', 47, 12, '2026-04-22 10:00:00'),
  (@sarah, 'Exclusive: Full PSD files for January collection',
     'Download all 12 high-res illustration files including layered PSDs and brush presets...',
     'members', 'published', 89, 23, '2026-04-18 10:00:00'),
  (@sarah, 'Monthly Q&A Recap — Your questions answered',
     'Thank you for all the amazing questions this month! Here are my answers to the top 20...',
     'public', 'published', 34, 8, '2026-04-10 10:00:00'),
  (@sarah, 'Brush pack v3.0 — Premium Procreate brushes',
     'My custom brush pack updated with 15 new brushes optimized for iPad Pro and Apple Pencil...',
     'members', 'published', 156, 45, '2026-04-05 10:00:00'),
  (@sarah, 'New series announcement: Digital Landscapes',
     'Excited to announce my new series focused on creating stunning digital landscapes...',
     'public', 'published', 72, 19, '2026-03-28 10:00:00');

-- Payout history for Sarah Chen (Earnings dashboard).
INSERT INTO payouts (user_id, reference, amount, method, status, payout_date) VALUES
  (@sarah, 'PO-001', 450.00, 'stripe', 'completed', '2026-09-01'),
  (@sarah, 'PO-002', 380.00, 'stripe', 'completed', '2026-08-01'),
  (@sarah, 'PO-003', 420.00, 'stripe', 'completed', '2026-07-01'),
  (@sarah, 'PO-004', 310.00, 'stripe', 'completed', '2026-06-01'),
  (@sarah, 'PO-005', 220.00, 'stripe', 'completed', '2026-05-01');

-- ============================================================================
--  End of schema
-- ============================================================================
