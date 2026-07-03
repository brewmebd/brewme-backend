-- Sequence and defined type
CREATE SEQUENCE IF NOT EXISTS categories_id_seq;
CREATE SEQUENCE IF NOT EXISTS users_id_seq;
CREATE SEQUENCE IF NOT EXISTS supporters_id_seq;
CREATE SEQUENCE IF NOT EXISTS membership_tiers_id_seq;
CREATE SEQUENCE IF NOT EXISTS goals_id_seq;
CREATE SEQUENCE IF NOT EXISTS donations_id_seq;
CREATE SEQUENCE IF NOT EXISTS memberships_id_seq;
CREATE SEQUENCE IF NOT EXISTS payout_accounts_id_seq;
CREATE SEQUENCE IF NOT EXISTS payouts_id_seq;
CREATE SEQUENCE IF NOT EXISTS posts_id_seq;
CREATE SEQUENCE IF NOT EXISTS post_media_id_seq;
CREATE SEQUENCE IF NOT EXISTS post_comments_id_seq;
CREATE SEQUENCE IF NOT EXISTS social_links_id_seq;
CREATE SEQUENCE IF NOT EXISTS tier_perks_id_seq;

DROP TABLE IF EXISTS "public"."categories";
-- Table Definition
CREATE TABLE "public"."categories" (
    "id" int4 NOT NULL DEFAULT nextval('categories_id_seq'::regclass),
    "name" varchar(60) NOT NULL,
    "slug" varchar(60) NOT NULL,
    "created_at" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX uq_categories_name ON public.categories USING btree (name);
CREATE UNIQUE INDEX uq_categories_slug ON public.categories USING btree (slug);

DROP TABLE IF EXISTS "public"."users";
-- Table Definition
CREATE TABLE "public"."users" (
    "id" int8 NOT NULL DEFAULT nextval('users_id_seq'::regclass),
    "full_name" varchar(120) NOT NULL,
    "username" varchar(60) NOT NULL,
    "email" varchar(190) NOT NULL,
    "password_hash" varchar(255) NOT NULL,
    "bio" text,
    "category_id" int4,
    "avatar_url" varchar(500) DEFAULT NULL::character varying,
    "email_verified_at" timestamp,
    "created_at" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "fk_users_category" FOREIGN KEY ("category_id") REFERENCES "public"."categories"("id") ON DELETE SET NULL ON UPDATE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX uq_users_username ON public.users USING btree (username);
CREATE UNIQUE INDEX uq_users_email ON public.users USING btree (email);

DROP TABLE IF EXISTS "public"."supporters";
-- Table Definition
CREATE TABLE "public"."supporters" (
    "id" int8 NOT NULL DEFAULT nextval('supporters_id_seq'::regclass),
    "user_id" int8 NOT NULL,
    "display_name" varchar(120) DEFAULT NULL::character varying,
    "email" varchar(190) DEFAULT NULL::character varying,
    "is_anonymous" bool NOT NULL DEFAULT false,
    "first_supported_at" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "fk_supporters_user" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX uq_supporter_creator_email ON public.supporters USING btree (user_id, email);

DROP TABLE IF EXISTS "public"."membership_tiers";
-- Table Definition
CREATE TABLE "public"."membership_tiers" (
    "id" int8 NOT NULL DEFAULT nextval('membership_tiers_id_seq'::regclass),
    "user_id" int8 NOT NULL,
    "name" varchar(80) NOT NULL,
    "price" numeric(10,2) NOT NULL,
    "billing_period" varchar(50) NOT NULL DEFAULT 'monthly'::character varying,
    "color" varchar(40) DEFAULT NULL::character varying,
    "sort_order" int2 NOT NULL DEFAULT 0,
    "is_active" bool NOT NULL DEFAULT true,
    "created_at" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "fk_tiers_user" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY ("id")
);

DROP TABLE IF EXISTS "public"."goals";
-- Table Definition
CREATE TABLE "public"."goals" (
    "id" int8 NOT NULL DEFAULT nextval('goals_id_seq'::regclass),
    "user_id" int8 NOT NULL,
    "label" varchar(200) NOT NULL,
    "target_amount" numeric(10,2) NOT NULL,
    "current_amount" numeric(10,2) NOT NULL DEFAULT 0.00,
    "is_active" bool NOT NULL DEFAULT true,
    "created_at" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "fk_goals_user" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY ("id")
);

DROP TABLE IF EXISTS "public"."donations";
-- Table Definition
CREATE TABLE "public"."donations" (
    "id" int8 NOT NULL DEFAULT nextval('donations_id_seq'::regclass),
    "user_id" int8 NOT NULL,
    "supporter_id" int8,
    "display_name" varchar(120) DEFAULT NULL::character varying,
    "is_anonymous" bool NOT NULL DEFAULT false,
    "cups" int2 NOT NULL DEFAULT 1,
    "amount" numeric(10,2) NOT NULL,
    "currency" bpchar(3) NOT NULL DEFAULT 'USD'::bpchar,
    "message" text,
    "status" varchar(50) NOT NULL DEFAULT 'succeeded'::character varying,
    "stripe_charge_id" varchar(255) DEFAULT NULL::character varying,
    "reply_message" text,
    "replied_at" timestamp,
    "created_at" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "fk_donations_supporter" FOREIGN KEY ("supporter_id") REFERENCES "public"."supporters"("id") ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT "fk_donations_user" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY ("id")
);

DROP TABLE IF EXISTS "public"."memberships";
-- Table Definition
CREATE TABLE "public"."memberships" (
    "id" int8 NOT NULL DEFAULT nextval('memberships_id_seq'::regclass),
    "user_id" int8 NOT NULL,
    "tier_id" int8 NOT NULL,
    "supporter_id" int8,
    "display_name" varchar(120) DEFAULT NULL::character varying,
    "amount" numeric(10,2) NOT NULL,
    "status" varchar(50) NOT NULL DEFAULT 'active'::character varying,
    "stripe_subscription_id" varchar(255) DEFAULT NULL::character varying,
    "reply_message" text,
    "replied_at" timestamp,
    "started_at" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "current_period_end" timestamp,
    "canceled_at" timestamp,
    CONSTRAINT "fk_memberships_supporter" FOREIGN KEY ("supporter_id") REFERENCES "public"."supporters"("id") ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT "fk_memberships_tier" FOREIGN KEY ("tier_id") REFERENCES "public"."membership_tiers"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT "fk_memberships_user" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY ("id")
);

DROP TABLE IF EXISTS "public"."notification_settings";
-- Table Definition
CREATE TABLE "public"."notification_settings" (
    "user_id" int8 NOT NULL,
    "new_supporter" bool NOT NULL DEFAULT true,
    "new_message" bool NOT NULL DEFAULT true,
    "weekly_report" bool NOT NULL DEFAULT false,
    "marketing_emails" bool NOT NULL DEFAULT false,
    "updated_at" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "fk_notif_user" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY ("user_id")
);

DROP TABLE IF EXISTS "public"."payout_accounts";
-- Table Definition
CREATE TABLE "public"."payout_accounts" (
    "id" int8 NOT NULL DEFAULT nextval('payout_accounts_id_seq'::regclass),
    "user_id" int8 NOT NULL,
    "provider" varchar(50) NOT NULL DEFAULT 'stripe'::character varying,
    "external_account_id" varchar(255) DEFAULT NULL::character varying,
    "card_last4" bpchar(4) DEFAULT NULL::bpchar,
    "is_connected" bool NOT NULL DEFAULT false,
    "connected_at" timestamp,
    CONSTRAINT "fk_payout_acct_user" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX uq_payout_user_provider ON public.payout_accounts USING btree (user_id, provider);

DROP TABLE IF EXISTS "public"."payouts";
-- Table Definition
CREATE TABLE "public"."payouts" (
    "id" int8 NOT NULL DEFAULT nextval('payouts_id_seq'::regclass),
    "user_id" int8 NOT NULL,
    "reference" varchar(20) NOT NULL,
    "amount" numeric(10,2) NOT NULL,
    "currency" bpchar(3) NOT NULL DEFAULT 'USD'::bpchar,
    "method" varchar(50) NOT NULL DEFAULT 'stripe'::character varying,
    "status" varchar(50) NOT NULL DEFAULT 'pending'::character varying,
    "payout_date" date NOT NULL,
    "created_at" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "fk_payouts_user" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX uq_payout_reference ON public.payouts USING btree (reference);

DROP TABLE IF EXISTS "public"."posts";
-- Table Definition
CREATE TABLE "public"."posts" (
    "id" int8 NOT NULL DEFAULT nextval('posts_id_seq'::regclass),
    "user_id" int8 NOT NULL,
    "title" varchar(255) NOT NULL,
    "body" text,
    "preview" varchar(500) DEFAULT NULL::character varying,
    "image_url" varchar(1000) DEFAULT NULL::character varying,
    "visibility" varchar(50) NOT NULL DEFAULT 'public'::character varying,
    "status" varchar(50) NOT NULL DEFAULT 'published'::character varying,
    "likes_count" int4 NOT NULL DEFAULT 0,
    "comments_count" int4 NOT NULL DEFAULT 0,
    "published_at" timestamp,
    "created_at" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "fk_posts_user" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY ("id")
);

DROP TABLE IF EXISTS "public"."post_media";
-- Table Definition
CREATE TABLE "public"."post_media" (
    "id" int8 NOT NULL DEFAULT nextval('post_media_id_seq'::regclass),
    "post_id" int8 NOT NULL,
    "url" varchar(500) NOT NULL,
    "media_type" varchar(50) NOT NULL DEFAULT 'image'::character varying,
    "sort_order" int2 NOT NULL DEFAULT 0,
    CONSTRAINT "fk_media_post" FOREIGN KEY ("post_id") REFERENCES "public"."posts"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY ("id")
);

DROP TABLE IF EXISTS "public"."post_comments";
-- Table Definition
CREATE TABLE "public"."post_comments" (
    "id" int8 NOT NULL DEFAULT nextval('post_comments_id_seq'::regclass),
    "post_id" int8 NOT NULL,
    "supporter_id" int8,
    "author_name" varchar(120) NOT NULL,
    "body" text NOT NULL,
    "created_at" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "fk_comments_post" FOREIGN KEY ("post_id") REFERENCES "public"."posts"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT "fk_comments_supporter" FOREIGN KEY ("supporter_id") REFERENCES "public"."supporters"("id") ON DELETE SET NULL ON UPDATE CASCADE,
    PRIMARY KEY ("id")
);

DROP TABLE IF EXISTS "public"."social_links";
-- Table Definition
CREATE TABLE "public"."social_links" (
    "id" int8 NOT NULL DEFAULT nextval('social_links_id_seq'::regclass),
    "user_id" int8 NOT NULL,
    "platform" varchar(50) NOT NULL,
    "url" varchar(500) NOT NULL,
    "sort_order" int2 NOT NULL DEFAULT 0,
    CONSTRAINT "fk_social_user" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY ("id")
);


-- Indices
CREATE UNIQUE INDEX uq_social_user_platform ON public.social_links USING btree (user_id, platform);

DROP TABLE IF EXISTS "public"."tier_perks";
-- Table Definition
CREATE TABLE "public"."tier_perks" (
    "id" int8 NOT NULL DEFAULT nextval('tier_perks_id_seq'::regclass),
    "tier_id" int8 NOT NULL,
    "perk_text" varchar(200) NOT NULL,
    "sort_order" int2 NOT NULL DEFAULT 0,
    CONSTRAINT "fk_perks_tier" FOREIGN KEY ("tier_id") REFERENCES "public"."membership_tiers"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    PRIMARY KEY ("id")
);

DROP VIEW IF EXISTS "public"."creator_earnings";
 SELECT id AS user_id,
    (COALESCE(( SELECT sum(donations.amount) AS sum
           FROM donations
          WHERE ((donations.user_id = u.id) AND ((donations.status)::text = 'succeeded'::text))), (0)::numeric) + COALESCE(( SELECT sum(memberships.amount) AS sum
           FROM memberships
          WHERE ((memberships.user_id = u.id) AND ((memberships.status)::text = 'active'::text))), (0)::numeric)) AS total_earned,
    COALESCE(( SELECT sum(payouts.amount) AS sum
           FROM payouts
          WHERE ((payouts.user_id = u.id) AND ((payouts.status)::text = 'completed'::text))), (0)::numeric) AS total_paid_out,
    ( SELECT count(0) AS count
           FROM supporters
          WHERE (supporters.user_id = u.id)) AS supporter_count
   FROM users u;

DROP VIEW IF EXISTS "public"."supporter_feed";
 SELECT d.user_id,
    'coffee'::text AS support_type,
        CASE
            WHEN (d.is_anonymous = true) THEN 'Anonymous'::character varying
            ELSE COALESCE(d.display_name, 'Anonymous'::character varying)
        END AS display_name,
    d.message,
    d.cups,
    d.amount,
    (d.reply_message IS NOT NULL) AS replied,
    d.created_at
   FROM donations d
  WHERE ((d.status)::text = 'succeeded'::text)
UNION ALL
 SELECT m.user_id,
    'membership'::text AS support_type,
    COALESCE(m.display_name, 'Anonymous'::character varying) AS display_name,
    NULL::text AS message,
    0 AS cups,
    m.amount,
    (m.reply_message IS NOT NULL) AS replied,
    m.started_at AS created_at
   FROM memberships m
  WHERE ((m.status)::text = 'active'::text);

