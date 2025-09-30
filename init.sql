-- Создание таблицы для URL
CREATE TABLE IF NOT EXISTS url (
                                   id SERIAL PRIMARY KEY,
                                   url TEXT NOT NULL,
                                   alias VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

-- Создание индекса для быстрого поиска по alias
CREATE INDEX IF NOT EXISTS idx_url_alias ON url(alias);

-- Создание индекса для created_at (опционально)
CREATE INDEX IF NOT EXISTS idx_url_created_at ON url(created_at);