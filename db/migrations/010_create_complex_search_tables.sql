
-- История запусков поиска сложных маршрутов
CREATE TABLE common.complex_search_sessions (
    id BIGSERIAL PRIMARY KEY,

    status VARCHAR(20) NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'completed', 'failed')),

    search_params JSONB NOT NULL,

    departure_date_from DATE NOT NULL,
    departure_date_to DATE NOT NULL,

    currency VARCHAR(3) NOT NULL DEFAULT 'rub',

    error_message TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);

CREATE INDEX idx_complex_search_sessions_created_at
    ON common.complex_search_sessions (created_at DESC);

CREATE INDEX idx_complex_search_sessions_status
    ON common.complex_search_sessions (status);


-- Снимки билетов, полученных в рамках конкретного поиска
CREATE TABLE common.complex_search_flights (
    id BIGSERIAL PRIMARY KEY,

    session_id BIGINT NOT NULL
        REFERENCES common.complex_search_sessions(id),

    flight_key VARCHAR(500) NOT NULL,

    origin VARCHAR(10) NOT NULL,
    destination VARCHAR(10) NOT NULL,

    origin_airport VARCHAR(10),
    destination_airport VARCHAR(10),

    departure_at TIMESTAMPTZ NOT NULL,

    price BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,

    airline VARCHAR(10),
    flight_number VARCHAR(20),

    transfers INTEGER,
    duration INTEGER,

    link TEXT,
    booking_link TEXT,

    saved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (session_id, flight_key)
);

CREATE INDEX idx_complex_search_flights_session
    ON common.complex_search_flights (session_id);

CREATE INDEX idx_complex_search_flights_route
    ON common.complex_search_flights (session_id, origin, destination);

CREATE INDEX idx_complex_search_flights_departure
    ON common.complex_search_flights (session_id, departure_at);

CREATE INDEX idx_complex_search_flights_price
    ON common.complex_search_flights (session_id, price);
