CREATE OR REPLACE FUNCTION common.upsert_flight(
    p_flight_key          varchar,
    p_origin              varchar,
    p_destination         varchar,
    p_origin_airport      varchar,
    p_destination_airport varchar,
    p_departure_at        timestamp,
    p_return_at            timestamp,
    p_price               integer,
    p_currency             varchar,
    p_airline              varchar,
    p_flight_number        varchar,
    p_transfers             integer,
    p_return_transfers      integer,
    p_duration              integer,
    p_link                  text,
    p_booking_link          text
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
            f.price AS old_price
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
            updated_at   = NOW()

        RETURNING
            (xmax = 0) AS inserted,
            common.flights.price AS new_price
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