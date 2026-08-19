# 🔑 GETTING API KEYS - Полная инструкция

---

## 1. AVIASALES API KEY

### Шаг 1: Регистрация на Aviasales Partnerès Program

1. Перейти на https://www.aviasales.com/affiliate-program
2. Нажать "Become a Partner" или "Присоединиться"
3. Заполнить форму регистрации:
   - Имя/компания
   - Email
   - Тип программы: "API Partner"
4. Согласиться с условиями
5. Нажать "Register"

### Шаг 2: Получение API Key

1. Проверить email (письмо придет в течение 1-2 часов)
2. Перейти по ссылке активации
3. Войти в личный кабинет Aviasales Partner
4. В разделе "API" или "Developer" найти API Key
5. Скопировать ключ (выглядит как: `your-api-key-12345abcde`)

### Шаг 3: Тестирование

```bash
curl "https://api.aviasales.ru/v2/search?origin=MOW&destination=BCN&depart_date=2024-05-15&one_way=1&token=YOUR_API_KEY"
```

Если вернулся JSON с билетами - ключ работает! ✅

### Documentation
https://www.aviasales.com/api

---

## 2. TELEGRAM BOT TOKEN

### Шаг 1: Создание бота

1. Открыть Telegram (мобильное приложение или web.telegram.org)
2. Найти контакт **@BotFather**
3. Отправить команду: `/start`
4. Отправить команду: `/newbot`

### Шаг 2: Конфигурация

Ответить на вопросы BotFather:
```
📝 Как назвать бота?
→ Aviasales Deals Bot

📝 Какое имя пользователя (username)?
→ aviasales_deals_bot
```

### Шаг 3: Получение Token

BotFather отправит сообщение:
```
✅ Done! Congratulations on your new bot. 
You will find it at https://t.me/aviasales_deals_bot. 
You can now add a description, about section and profile picture for your bot.

Use this token to access the HTTP API:
123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11

For a description of the Bot API, please see this page:
https://core.telegram.org/bots/api
```

**Скопировать token**: `123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11`

### Шаг 4: Добавление бота в канал

1. Создать Telegram канал (приватный или публичный)
   - В Telegram нажать "+" → "New Channel"
   - Дать название: "Aviasales Deals"
   - Выбрать "Private" или "Public"
   - Нажать "Create"

2. Добавить бота как администратора
   - В канале нажать имя канала → "Members"
   - Нажать "Add Member"
   - Найти `@aviasales_deals_bot`
   - Нажать "Add"
   - Нажать "Admin" и дать права на публикацию

### Шаг 5: Получение Channel ID

```bash
# Отправить от имени бота тестовое сообщение
curl -X POST https://api.telegram.org/bot123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11/sendMessage \
  -d chat_id=-1001234567890 \
  -d text="Test message"
```

Или более простой способ:
1. Отправить любое сообщение в канал
2. В браузере перейти:
   ```
   https://api.telegram.org/bot123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11/getUpdates
   ```
3. Найти в JSON `"chat":{"id": -1001234567890}`

**Это и есть Channel ID**: `-1001234567890`

### Проверка

```bash
# Отправить сообщение в канал
curl -X POST https://api.telegram.org/bot123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11/sendMessage \
  -d chat_id=-1001234567890 \
  -d text="✈️ Test message from bot"
```

Если сообщение появилось в канале - всё работает! ✅

### Documentation
https://core.telegram.org/bots/api

---

## 3. OPENAI API KEY

### Шаг 1: Создание аккаунта

1. Перейти на https://platform.openai.com
2. Нажать "Sign up"
3. Выбрать метод регистрации (Google, Microsoft или Email)
4. Заполнить данные:
   - Email
   - Пароль
   - Имя
   - Дата рождения
5. Подтвердить email

### Шаг 2: Добавление способа оплаты

1. В левом меню: "Billing" → "Overview"
2. Нажать "Add to billing method"
3. Добавить кредитную карту (Visa, MasterCard, etc.)
4. Подтвердить платёж

### Шаг 3: Установка лимита расходов

1. "Billing" → "Usage limits"
2. Установить лимит (например, $10/месяц)
3. Это предотвратит перерасходы

### Шаг 4: Создание API Key

1. В левом меню: "API keys"
2. Нажать "Create new secret key"
3. Выбрать organization
4. Нажать "Create secret key"
5. Скопировать ключ: `sk-proj-...` (больше не сможешь увидеть!)

⚠️ **ВАЖНО:** Сохрани ключ в безопасном месте! Больше ты его не сможешь увидеть.

### Шаг 5: Тестирование

```bash
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer sk-proj-YOUR_KEY_HERE"
```

Если вернулся JSON с моделями - ключ работает! ✅

### Используемая модель

Для нашего проекта используем **GPT-4 Turbo** (самая быстрая):

```bash
curl https://api.openai.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-proj-YOUR_KEY_HERE" \
  -d '{
    "model": "gpt-4-turbo",
    "messages": [{"role": "user", "content": "Привет!"}],
    "temperature": 0.7,
    "max_tokens": 500
  }'
```

### Стоимость

| Модель | Input | Output |
|--------|-------|--------|
| GPT-3.5 | $0.0005 / 1K | $0.0015 / 1K |
| GPT-4 Turbo | $0.01 / 1K | $0.03 / 1K |

**Примерно:** каждый анализ билета стоит ~$0.01

### Documentation
https://platform.openai.com/docs/api-reference

---

## 4. POSTGRESQL DATABASE

### Вариант 1: Docker (Рекомендуется)

Уже настроен в `docker-compose.yml`:

```bash
docker-compose up -d postgres redis
```

Доступ:
```
Host: localhost
Port: 5432
Username: bot
Password: secure_password (из .env)
Database: aviasales_db
```

### Вариант 2: Локальная установка (macOS)

```bash
# Установить через Homebrew
brew install postgresql

# Запустить сервис
brew services start postgresql

# Создать БД
createdb aviasales_db

# Подключиться
psql -d aviasales_db
```

### Вариант 3: Облачная БД (AWS RDS)

1. Перейти на https://console.aws.amazon.com
2. RDS → Create database
3. Выбрать PostgreSQL 15
4. Конфигурация:
   - DB instance class: db.t3.micro (free tier)
   - Allocated storage: 20 GB
   - Master username: bot
   - Master password: strong_password
5. Create
6. Ждать создания (5-10 минут)
7. В "Connectivity" скопировать Endpoint

CONNECTION STRING:
```
postgres://bot:password@your-rds-endpoint.amazonaws.com:5432/aviasales_db
```

---

## 5. REDIS CACHE

### Вариант 1: Docker (Рекомендуется)

Уже настроен в `docker-compose.yml`:

```bash
docker-compose up -d redis
```

Доступ:
```
Host: localhost
Port: 6379
Database: 0
```

### Вариант 2: Локальная установка (macOS)

```bash
# Установить
brew install redis

# Запустить
brew services start redis

# Тест
redis-cli ping
# OUTPUT: PONG
```

### Вариант 3: Облачная (Redis Cloud)

1. Перейти на https://redis.com/cloud
2. Sign up
3. Create database
4. Выбрать бесплатный план
5. Выбрать регион
6. Create
7. Скопировать CONNECTION STRING

---

## 6. POSTGRES ADMINISTRATION

### pgAdmin (веб-интерфейс)

Доступ: http://localhost:5050

```
Email: admin@example.com
Password: admin (из .env)
```

Или запустить:
```bash
make docker-up
# pgAdmin уже в docker-compose
```

---

## ✅ ФИНАЛЬНАЯ CHECKLIST

Перед запуском проекта заполни все:

```
API Keys:
☐ AVIASALES_API_KEY
☐ OPENAI_API_KEY
☐ TELEGRAM_TOKEN
☐ TELEGRAM_CHANNEL_ID

Database:
☐ DATABASE_URL настроена
☐ PostgreSQL запущена (docker или локально)
☐ Миграции применены (make migrate)

Redis:
☐ Redis запущена (docker или локально)

Telegram:
☐ Бот создан (@BotFather)
☐ Бот добавлен в канал
☐ Бот имеет права администратора

Environment:
☐ .env файл создан и заполнен
☐ .env не коммитится в git
☐ Все переменные из .env.example заполнены
```

---

## 🚀 БЫСТРЫЙ СТАРТ (после получения ключей)

```bash
# 1. Скопировать и заполнить .env
cp config/.env.example .env
nano .env
# Вставить все API ключи

# 2. Запустить docker-compose
docker-compose -f config/docker-compose.yml up -d

# 3. Применить миграции
make migrate

# 4. Проверить здоровье
curl http://localhost:8080/health

# 5. Смотреть логи
docker-compose logs -f parser
```

---

## 💬 TROUBLESHOOTING

### "Invalid API key"
- Проверить, что ключ скопирован полностью (без пробелов)
- Перечитать email от провайдера
- Убедиться, что ключ не истёк

### "Unauthorized to access chat"
- Проверить, что бот добавлен в канал
- Проверить, что бот - администратор
- Проверить Channel ID (должно быть отрицательное число)

### "Connection refused"
- PostgreSQL запущена? (`docker ps`)
- Redis запущена? (`redis-cli ping`)
- DATABASE_URL правильная в .env?

### "Rate limit exceeded"
- OpenAI: подождать 1 минуту перед следующим запросом
- Aviasales: максимум 10 запросов в секунду
- Telegram: максимум 30 сообщений в секунду

---

Made with ❤️

Questions? → GitHub Issues
