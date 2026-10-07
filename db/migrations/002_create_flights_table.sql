-- Таблица для хранения билетов
CREATE TABLE IF NOT EXISTS common.flights (
    id                  bigserial     PRIMARY KEY,

    -- Уникальный идентификатор билета (по нему определяем дубликаты)
    flight_key          varchar(255)  UNIQUE NOT NULL,

    -- Маршрут
    origin              varchar(10)   NOT NULL,
    destination         varchar(10)   NOT NULL,
    origin_airport      varchar(10),
    destination_airport varchar(10),

    -- Время
    departure_at        timestamp     NOT NULL,
    return_at           timestamp,

    -- Цена
    price               bigint        NOT NULL,
    currency            varchar(3)    NOT NULL,

    -- Авиакомпания
    airline             varchar(10)   NOT NULL,
    flight_number       varchar(20),

    -- Детали
    transfers           integer       NOT NULL DEFAULT 0,
    return_transfers    integer       NOT NULL DEFAULT 0,
    duration            integer,

    -- Ссылки
    link                text,
    booking_link        text,

    -- Метаданные синхронизации
    first_seen_at       timestamp     NOT NULL DEFAULT NOW(),
    last_seen_at        timestamp     NOT NULL DEFAULT NOW(),
    is_active           boolean       NOT NULL DEFAULT TRUE,

    -- Системные поля
    created_at          timestamp     NOT NULL DEFAULT NOW(),
    updated_at          timestamp     NOT NULL DEFAULT NOW()
);

-- Индексы для ускорения поиска
CREATE INDEX idx_flights_route ON common.flights (origin, destination);

CREATE INDEX idx_flights_departure ON common.flights (departure_at);

CREATE INDEX idx_flights_price ON common.flights (price);

CREATE INDEX idx_flights_active ON common.flights (is_active);

CREATE INDEX idx_flights_first_seen ON common.flights (first_seen_at);

CREATE INDEX idx_flights_key ON common.flights (flight_key);

-- Комментарии
COMMENT ON TABLE common.flights IS 'Билеты из Travelpayouts API';

COMMENT ON COLUMN common.flights.flight_key IS 'Уникальный ключ: origin:destination:departure_at:return_at:airline:flight_number';

COMMENT ON COLUMN common.flights.first_seen_at IS 'Когда билет впервые появился в системе';

COMMENT ON COLUMN common.flights.last_seen_at IS 'Когда билет последний раз возвращался API';

COMMENT ON COLUMN common.flights.is_active IS 'FALSE если билет пропал из выдачи API';