-- Таблица для настройки маршрутов, которые мониторим
CREATE TABLE IF NOT EXISTS common.monitored_routes (
    id                    serial      PRIMARY KEY,

    -- Маршрут
    origin                varchar(10) NOT NULL,
    destination           varchar(10) NOT NULL,

    -- Параметры поиска
    start_month           date        NOT NULL,
    end_month             date        NOT NULL,
    direct                boolean     NOT NULL DEFAULT FALSE,
    min_trip_duration     integer     NOT NULL DEFAULT 5,
    max_trip_duration     integer     NOT NULL DEFAULT 8,
    currency              varchar(3)  NOT NULL DEFAULT 'rub',

    -- Расписание синхронизации
    sync_interval_minutes integer     NOT NULL DEFAULT 60,

    -- Метаданные
    is_active             boolean     NOT NULL DEFAULT TRUE,
    last_sync_at          timestamp,
    last_sync_status      varchar(20),

    -- Системные поля
    created_at            timestamp   NOT NULL DEFAULT NOW(),
    updated_at            timestamp   NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_route UNIQUE (origin, destination)
);

-- Индексы
CREATE INDEX idx_monitored_routes_active ON common.monitored_routes (is_active);

CREATE INDEX idx_monitored_routes_last_sync ON common.monitored_routes (last_sync_at);

-- Комментарии
COMMENT ON TABLE common.monitored_routes IS 'Маршруты для автоматического мониторинга';

COMMENT ON COLUMN common.monitored_routes.sync_interval_minutes IS 'Интервал синхронизации в минутах';