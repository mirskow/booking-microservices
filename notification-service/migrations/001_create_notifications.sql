BEGIN;

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    event_id UUID NOT NULL,

    recipient_id UUID NOT NULL,
    recipient_name VARCHAR(255) NOT NULL DEFAULT '',
    recipient_email VARCHAR(320) NOT NULL DEFAULT '',
    booking_id UUID NOT NULL,

    type VARCHAR(50) NOT NULL,
    channel VARCHAR(20) NOT NULL DEFAULT 'email',

    status VARCHAR(20) NOT NULL DEFAULT 'pending',

    payload JSONB NOT NULL DEFAULT '{}'::jsonb,

    attempts INTEGER NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ,

    CONSTRAINT notifications_type_check
        CHECK (
            type IN (
                'booking.created',
                'booking.cancelled'
            )
        ),

    CONSTRAINT notifications_channel_check
        CHECK (channel IN ('email')),

    CONSTRAINT notifications_status_check
        CHECK (
            status IN (
                'pending',
                'sent',
                'failed'
            )
        ),

    CONSTRAINT notifications_attempts_check
        CHECK (attempts >= 0),

    CONSTRAINT notifications_event_recipient_unique
        UNIQUE (event_id, recipient_id)
);


CREATE INDEX IF NOT EXISTS idx_notifications_recipient_created
    ON notifications (recipient_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_status
    ON notifications (status);

CREATE INDEX IF NOT EXISTS idx_notifications_booking
    ON notifications (booking_id);

CREATE INDEX IF NOT EXISTS idx_notifications_event
    ON notifications (event_id);


COMMIT;