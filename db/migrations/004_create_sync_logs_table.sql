-- Таблица для логирования синхронизаций
CREATE TABLE IF NOT EXISTS common.sync_logs (
    id                  bigserial   PRIMARY KEY,

    route_id            integer     REFERENCES common.monitored_routes (id) ON DELETE CASCADE,

    -- Время выполнения
    started_at          timestamp   NOT NULL,
    finished_at         timestamp,

    -- Статус
    status              varchar(20) NOT NULL,

    -- Статистика
    flights_found       integer     DEFAULT 0,
    flights_new         integer     DEFAULT 0,
    flights_updated     integer     DEFAULT 0,
    flights_deactivated integer     DEFAULT 0,

    -- Ошибки
    error_message       text,

    -- Системные поля
    created_at          timestamp   NOT NULL DEFAULT NOW()
);

-- Индексы
CREATE INDEX idx_sync_logs_route ON common.sync_logs (route_id);

CREATE INDEX idx_sync_logs_started ON common.sync_logs (started_at DESC);

CREATE INDEX idx_sync_logs_status ON common.sync_logs (status);

-- Комментарии
COMMENT ON TABLE common.sync_logs IS 'История синхронизаций маршрутов';

COMMENT ON COLUMN common.sync_logs.flights_new IS 'Новые билеты, появившиеся в этой синхронизации';