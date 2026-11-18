BEGIN;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    name VARCHAR(255) NOT NULL,
    email VARCHAR(320) NOT NULL,
    timezone VARCHAR(100) NOT NULL DEFAULT 'UTC',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT users_name_not_empty
        CHECK (length(trim(name)) > 0),

    CONSTRAINT users_email_not_empty
        CHECK (length(trim(email)) > 0),

    CONSTRAINT users_timezone_not_empty
        CHECK (length(trim(timezone)) > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower
    ON users (lower(email));


CREATE TABLE IF NOT EXISTS availability_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    weekday SMALLINT NOT NULL,

    start_time TIME NOT NULL,
    end_time TIME NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT availability_rules_weekday_check
        CHECK (weekday BETWEEN 0 AND 6),

    CONSTRAINT availability_rules_time_check
        CHECK (start_time < end_time),

    CONSTRAINT availability_rules_unique
        UNIQUE (user_id, weekday, start_time, end_time)
);

CREATE INDEX IF NOT EXISTS idx_availability_rules_user_weekday
    ON availability_rules (user_id, weekday, start_time);


COMMIT;