-- Migration: 001_initial.sql
-- Description: Initial database schema for Aviasales Bot

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Flights table - основной стол с авиабилетами
CREATE TABLE IF NOT EXISTS flights (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    origin VARCHAR(3) NOT NULL,                  -- MOW, SPB, etc.
    destination VARCHAR(3) NOT NULL,
    depart_date DATE NOT NULL,
    return_date DATE,
    price BIGINT NOT NULL,                       -- в копейках (25500 = 255 RUB)
    currency VARCHAR(3) DEFAULT 'RUB',
    airline VARCHAR(100),
    depart_time TIME,
    arrival_time TIME,
    duration_minutes INT,
    transfers INT DEFAULT 0,
    booking_link TEXT NOT NULL,                  -- Aviasales ссылка с реф параметром
    score FLOAT DEFAULT 0.0,
    parsed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Индексы для частых запросов
    CONSTRAINT chk_positive_price CHECK (price > 0),
    CONSTRAINT chk_date_logic CHECK (return_date IS NULL OR return_date > depart_date)
);

CREATE INDEX idx_flights_origin ON flights(origin);
CREATE INDEX idx_flights_destination ON flights(destination);
CREATE INDEX idx_flights_origin_dest ON flights(origin, destination);
CREATE INDEX idx_flights_depart_date ON flights(depart_date);
CREATE INDEX idx_flights_price ON flights(price);
CREATE INDEX idx_flights_parsed_at ON flights(parsed_at DESC);
CREATE INDEX idx_flights_score ON flights(score DESC);
CREATE INDEX idx_flights_created_at ON flights(created_at DESC);

-- Partitioning по месяцам (опционально, для больших объемов)
-- CREATE TABLE flights_2024_05 PARTITION OF flights FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');

-- Published flights - история опубликованных билетов
CREATE TABLE IF NOT EXISTS published_flights (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    flight_id UUID NOT NULL REFERENCES flights(id) ON DELETE CASCADE,
    telegram_message_id BIGINT,
    appeal TEXT NOT NULL,                        -- привлекательное описание
    segment VARCHAR(50),                         -- budget_travelers, family, luxury, business
    published_at TIMESTAMP NOT NULL DEFAULT NOW(),
    clicks INT DEFAULT 0,
    conversions INT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_published_flights_flight_id ON published_flights(flight_id);
CREATE INDEX idx_published_flights_published_at ON published_flights(published_at DESC);
CREATE INDEX idx_published_flights_telegram_msg ON published_flights(telegram_message_id);

-- User profiles - профили пользователей в Telegram
CREATE TABLE IF NOT EXISTS user_profiles (
    telegram_user_id BIGINT PRIMARY KEY,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    username VARCHAR(255),
    preferred_origins VARCHAR(3)[] DEFAULT ARRAY[]::VARCHAR(3)[],
    preferred_destinations VARCHAR(3)[] DEFAULT ARRAY[]::VARCHAR(3)[],
    max_budget INT,                              -- в рублях
    min_rating FLOAT DEFAULT 4.0,               -- минимальный рейтинг авиакомпании
    notification_enabled BOOLEAN DEFAULT TRUE,
    language VARCHAR(10) DEFAULT 'ru',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_profiles_username ON user_profiles(username);
CREATE INDEX idx_user_profiles_created_at ON user_profiles(created_at DESC);

-- User actions - логирование всех действий пользователя
CREATE TABLE IF NOT EXISTS user_actions (
    id BIGSERIAL PRIMARY KEY,
    telegram_user_id BIGINT REFERENCES user_profiles(telegram_user_id) ON DELETE CASCADE,
    action_type VARCHAR(50) NOT NULL,           -- view, click_buy, share, bookmark
    flight_id UUID REFERENCES flights(id) ON DELETE SET NULL,
    published_flight_id UUID REFERENCES published_flights(id) ON DELETE SET NULL,
    metadata JSONB,                              -- любые доп данные в JSON
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_actions_user_id ON user_actions(telegram_user_id);
CREATE INDEX idx_user_actions_action_type ON user_actions(action_type);
CREATE INDEX idx_user_actions_created_at ON user_actions(created_at DESC);
CREATE INDEX idx_user_actions_user_action ON user_actions(telegram_user_id, action_type);

-- Analytics cache - кэшированная статистика
CREATE TABLE IF NOT EXISTS analytics_cache (
    id SERIAL PRIMARY KEY,
    metric_type VARCHAR(50) NOT NULL,           -- top_routes, conversion_rate, etc.
    data JSONB NOT NULL,
    calculated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,
    
    UNIQUE(metric_type)
);

CREATE INDEX idx_analytics_metric_type ON analytics_cache(metric_type);
CREATE INDEX idx_analytics_expires_at ON analytics_cache(expires_at);

-- Error logs - логирование ошибок в сервисах
CREATE TABLE IF NOT EXISTS error_logs (
    id BIGSERIAL PRIMARY KEY,
    service VARCHAR(50) NOT NULL,               -- parser, ranker, publisher, api
    error_message TEXT NOT NULL,
    stack_trace TEXT,
    context JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_error_logs_service ON error_logs(service);
CREATE INDEX idx_error_logs_created_at ON error_logs(created_at DESC);
CREATE INDEX idx_error_logs_service_created ON error_logs(service, created_at DESC);

-- Queue for async tasks (опционально)
CREATE TABLE IF NOT EXISTS task_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_type VARCHAR(50) NOT NULL,             -- publish_flight, send_notification
    data JSONB NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',       -- pending, processing, completed, failed
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX idx_task_queue_status ON task_queue(status);
CREATE INDEX idx_task_queue_created_at ON task_queue(created_at DESC);

-- Create sample data for testing
INSERT INTO user_profiles (telegram_user_id, first_name, username, notification_enabled, language)
VALUES (
    123456789,
    'Test User',
    'testuser',
    TRUE,
    'ru'
) ON CONFLICT DO NOTHING;

-- Create views for easier reporting

-- View: Top routes by publication count
CREATE OR REPLACE VIEW top_routes_published AS
SELECT 
    f.origin,
    f.destination,
    COUNT(pf.id) as publication_count,
    AVG(pf.clicks) as avg_clicks,
    SUM(pf.conversions) as total_conversions
FROM flights f
LEFT JOIN published_flights pf ON f.id = pf.flight_id
WHERE pf.published_at > NOW() - INTERVAL '30 days'
GROUP BY f.origin, f.destination
ORDER BY publication_count DESC;

-- View: Daily statistics
CREATE OR REPLACE VIEW daily_stats AS
SELECT 
    DATE(f.parsed_at) as parse_date,
    COUNT(DISTINCT f.id) as flights_parsed,
    COUNT(DISTINCT pf.id) as flights_published,
    SUM(pf.clicks) as total_clicks,
    SUM(pf.conversions) as total_conversions
FROM flights f
LEFT JOIN published_flights pf ON f.id = pf.flight_id
GROUP BY DATE(f.parsed_at)
ORDER BY parse_date DESC;

-- View: User engagement
CREATE OR REPLACE VIEW user_engagement AS
SELECT 
    ua.telegram_user_id,
    up.username,
    COUNT(DISTINCT CASE WHEN ua.action_type = 'view' THEN ua.id END) as views,
    COUNT(DISTINCT CASE WHEN ua.action_type = 'click_buy' THEN ua.id END) as clicks,
    COUNT(DISTINCT CASE WHEN ua.action_type = 'share' THEN ua.id END) as shares,
    MAX(ua.created_at) as last_action
FROM user_actions ua
LEFT JOIN user_profiles up ON ua.telegram_user_id = up.telegram_user_id
GROUP BY ua.telegram_user_id, up.username;

-- Функции для удобства

-- Function: Get average price for route in last N days
CREATE OR REPLACE FUNCTION get_avg_price_for_route(
    p_origin VARCHAR(3),
    p_destination VARCHAR(3),
    p_days INT DEFAULT 30
)
RETURNS BIGINT AS $$
SELECT COALESCE(AVG(price), 0)::BIGINT
FROM flights
WHERE origin = p_origin
    AND destination = p_destination
    AND parsed_at > NOW() - (p_days || ' days')::INTERVAL;
$$ LANGUAGE SQL STABLE;

-- Function: Calculate discount percentage
CREATE OR REPLACE FUNCTION calculate_discount_pct(
    p_current_price BIGINT,
    p_avg_price BIGINT
)
RETURNS INT AS $$
SELECT CASE 
    WHEN p_avg_price = 0 THEN 0
    ELSE ((p_avg_price - p_current_price) * 100 / p_avg_price)::INT
END;
$$ LANGUAGE SQL IMMUTABLE;

-- Initial stats insert
INSERT INTO analytics_cache (metric_type, data)
VALUES ('initialization', '{"status": "ready"}')
ON CONFLICT (metric_type) DO UPDATE
SET calculated_at = NOW();

-- Комментарии к таблицам (для документации)
COMMENT ON TABLE flights IS 'Основная таблица авиабилетов с информацией о цене и маршруте';
COMMENT ON TABLE published_flights IS 'История опубликованных билетов в Telegram с метриками';
COMMENT ON TABLE user_profiles IS 'Профили пользователей Telegram с предпочтениями';
COMMENT ON TABLE user_actions IS 'Логирование всех действий пользователей для аналитики';
COMMENT ON TABLE error_logs IS 'Логирование ошибок в сервисах';

-- Permissions (если используется отдельный пользователь БД)
-- GRANT CONNECT ON DATABASE aviasales_db TO bot;
-- GRANT USAGE ON SCHEMA public TO bot;
-- GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO bot;
-- GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO bot;
