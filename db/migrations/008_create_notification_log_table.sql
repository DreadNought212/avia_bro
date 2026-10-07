CREATE TABLE common.notification_log (
    id                bigserial   PRIMARY KEY,

    rule_id           integer     NOT NULL
        REFERENCES common.notification_rules (id)
        ON DELETE CASCADE,

    flight_key        text        NOT NULL,

    price             integer     NOT NULL,

    notification_type varchar(30) NOT NULL,

    sent_at           timestamp   NOT NULL DEFAULT now()
);

CREATE INDEX idx_notification_log_flight
    ON common.notification_log (flight_key);

CREATE INDEX idx_notification_log_rule
    ON common.notification_log (rule_id);