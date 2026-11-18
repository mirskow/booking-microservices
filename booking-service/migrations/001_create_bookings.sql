BEGIN;

CREATE EXTENSION IF NOT EXISTS btree_gist;


CREATE TABLE IF NOT EXISTS bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',

    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,

    status VARCHAR(20) NOT NULL DEFAULT 'confirmed',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT bookings_title_not_empty
        CHECK (length(trim(title)) > 0),

    CONSTRAINT bookings_time_check
        CHECK (starts_at < ends_at),

    CONSTRAINT bookings_status_check
        CHECK (status IN ('confirmed', 'cancelled'))
);


CREATE TABLE IF NOT EXISTS booking_participants (
    booking_id UUID NOT NULL
        REFERENCES bookings(id)
        ON DELETE CASCADE,

    user_id UUID NOT NULL,

    role VARCHAR(20) NOT NULL,

    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,

    status VARCHAR(20) NOT NULL DEFAULT 'confirmed',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (booking_id, user_id),

    CONSTRAINT booking_participants_role_check
        CHECK (role IN ('organizer', 'invitee')),

    CONSTRAINT booking_participants_status_check
        CHECK (status IN ('confirmed', 'cancelled')),

    CONSTRAINT booking_participants_time_check
        CHECK (starts_at < ends_at),

    CONSTRAINT booking_participants_no_overlap
        EXCLUDE USING gist (
            user_id WITH =,
            tstzrange(starts_at, ends_at, '[)') WITH &&
        )
        WHERE (status = 'confirmed')
);


CREATE INDEX IF NOT EXISTS idx_booking_participants_user
    ON booking_participants (user_id);

CREATE INDEX IF NOT EXISTS idx_booking_participants_user_status
    ON booking_participants (user_id, status);

CREATE INDEX IF NOT EXISTS idx_bookings_starts_at
    ON bookings (starts_at);

CREATE INDEX IF NOT EXISTS idx_bookings_status
    ON bookings (status);


COMMIT;