CREATE TABLE common.complex_routes (
    id                serial       PRIMARY KEY,
    name              varchar(100) NOT NULL,
    origin            varchar(10)  NOT NULL,
    min_trip_duration integer      NOT NULL,
    max_trip_duration integer      NOT NULL,
    active            boolean      NOT NULL DEFAULT true,
    created_at        timestamp    NOT NULL DEFAULT now(),
    updated_at        timestamp    NOT NULL DEFAULT now()
);

CREATE TABLE common.complex_route_destinations (
    id               serial      PRIMARY KEY,
    complex_route_id integer     NOT NULL
        REFERENCES common.complex_routes (id)
        ON DELETE CASCADE,

    destination      varchar(10) NOT NULL,

    UNIQUE (complex_route_id, destination)
);

ALTER TABLE common.flights
ADD COLUMN is_one_way boolean NOT NULL DEFAULT false;

DROP FUNCTION common.upsert_flight();

CREATE OR REPLACE FUNCTION common.upsert_flight(
    p_flight_key          varchar,
    p_origin              varchar,
    p_destination         varchar,
    p_origin_airport      varchar,
    p_destination_airport varchar,
    p_departure_at        timestamp,
    p_return_at           timestamp,
    p_price               integer,
    p_currency             varchar,
    p_airline              varchar,
    p_flight_number       varchar,
    p_transfers            integer,
    p_return_transfers    integer,
    p_duration             integer,
    p_link                text,
    p_booking_link        text,
    p_is_one_way          boolean
)
RETURNS TABLE (
    is_new        boolean,
    old_price     integer,
    price         integer,
    price_dropped boolean
)
LANGUAGE plpgsql
AS
$$
BEGIN
    RETURN QUERY

    WITH existing AS (
        SELECT
            f.price::integer AS old_price
        FROM common.flights AS f
        WHERE f.flight_key = p_flight_key
    ),

    upserted AS (
        INSERT INTO common.flights (
            flight_key,
            origin,
            destination,
            origin_airport,
            destination_airport,
            departure_at,
            return_at,
            price,
            currency,
            airline,
            flight_number,
            transfers,
            return_transfers,
            duration,
            link,
            booking_link,
            is_one_way,
            first_seen_at,
            last_seen_at,
            is_active
        )
        VALUES (
            p_flight_key,
            p_origin,
            p_destination,
            p_origin_airport,
            p_destination_airport,
            p_departure_at,
            p_return_at,
            p_price,
            p_currency,
            p_airline,
            p_flight_number,
            p_transfers,
            p_return_transfers,
            p_duration,
            p_link,
            p_booking_link,
            p_is_one_way,
            NOW(),
            NOW(),
            TRUE
        )

        ON CONFLICT (flight_key) DO UPDATE
        SET
            price        = EXCLUDED.price,
            link         = EXCLUDED.link,
            booking_link = EXCLUDED.booking_link,
            last_seen_at = NOW(),
            is_active    = TRUE,
            updated_at   = CASE
                WHEN common.flights.price IS DISTINCT FROM EXCLUDED.price
                    THEN NOW()
                ELSE common.flights.updated_at
            END

        RETURNING
            (xmax = 0) AS inserted,
            common.flights.price::integer AS new_price
    )

    SELECT
        upserted.inserted,
        existing.old_price,
        upserted.new_price,
        CASE
            WHEN existing.old_price IS NULL THEN FALSE
            WHEN upserted.new_price < existing.old_price THEN TRUE
            ELSE FALSE
        END
    FROM upserted
    LEFT JOIN existing ON TRUE;

END;
$$;