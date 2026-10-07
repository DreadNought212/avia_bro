CREATE TABLE common.notification_rules (
    id               serial    PRIMARY KEY,

    route_id         integer   NOT NULL
        REFERENCES common.monitored_routes (id)
        ON DELETE CASCADE,

    max_price        integer   NOT NULL,

    telegram_enabled boolean   NOT NULL DEFAULT true,

    created_at       timestamp NOT NULL DEFAULT now(),
    updated_at       timestamp NOT NULL DEFAULT now(),

    CONSTRAINT uq_notification_rules_route
        UNIQUE (route_id),

    CONSTRAINT chk_notification_max_price
        CHECK (max_price > 0)
);