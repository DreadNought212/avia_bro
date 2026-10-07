CREATE OR REPLACE FUNCTION common.get_active_routes()
AS
$$
BEGIN
    RETURN QUERY
    SELECT
        *
    FROM common.monitored_routes
    WHERE is_active = TRUE
    ORDER BY id;
END;
$$;