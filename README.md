# eCRM System

Репозиторий состоит из React-фронтенда в `ecrm/` и Go API в `backend/`. Данные хранятся в PostgreSQL, схема и справочные данные применяются автоматически при старте API.

## Быстрый запуск

```bash
docker compose up -d --build
cd ecrm
npm install
npm run dev
```

Фронтенд: `http://localhost:5173`, API: `http://localhost:8080`, readiness: `http://localhost:8080/health/ready`.

Локальная учётная запись из `docker-compose.yml`:

- email: `admin@ecrm.local`
- пароль: `ChangeMe123!`

Перед развёртыванием обязательно измените пароль, `JWT_SECRET`, пароль PostgreSQL и CORS origins. Для запуска без Docker скопируйте `backend/.env.example` в переменные окружения и выполните `go run ./cmd/api` из `backend/`.

## Реализованные модули

- JWT access-токены, ротация refresh-токенов, logout, bootstrap/первичная регистрация владельца;
- роли `admin`, `operator`, `finance`, `viewer` и защита mutating/admin endpoints;
- клиенты, сотрудники, поставщики, тарифы и склад eSIM с CRUD и фильтрами;
- пакетный импорт eSIM с контролем дублей ICCID;
- транзакционное создание заказа и атомарное назначение свободной eSIM;
- статусы заказа, оплаты, активации, отмены и возврат eSIM на склад;
- финансовый снимок заказа: цена, себестоимость, комиссия, налог, чистая прибыль;
- dashboard, продажи по дням/странам/поставщикам, глобальный поиск и CSV-экспорт;
- уведомления, история действий и журнал синхронизаций поставщиков;
- graceful shutdown, CORS, security headers, health checks, лимиты тела и PostgreSQL pool;
- встроенные версионируемые миграции и демо-справочники.
- интеграция Zesimo: wallet, полный каталог, on-demand provisioning и install links.

## Zesimo

Ключ хранится только на сервере в корневом `.env`, который исключён через `.gitignore`:

```dotenv
ZESIMO_API_KEY=replace-with-your-key
ZESIMO_KEY_NAME=my_key
```

После изменения ключа перезапустите API: `docker compose up -d --build api`.

- `GET /api/v1/integrations/zesimo` — состояние подключения и wallet без раскрытия ключа;
- `POST /api/v1/integrations/zesimo/sync` — синхронизация wallet и полного каталога;
- `POST /api/v1/orders/{id}/provision` — безопасный повтор provisioning;
- создание оплаченного заказа по тарифу Zesimo автоматически вызывает `POST /api/v1/orders` у поставщика;
- повтор использует тот же `Idempotency-Key`, поэтому не создаёт повторное списание;
- `activation_code`, `install_url`, `ios_tap_link` и `android_tap_link` сохраняются у eSIM и возвращаются фронтенду.

Каталог Zesimo ограничен примерно 60 запросами в минуту. Полная синхронизация занимает около 75–90 секунд; клиент автоматически обрабатывает HTTP 429. Цены и wallet Zesimo сохраняются в USD, локальные тарифы — в своей исходной валюте.

## Основные маршруты

Все бизнес-маршруты имеют префикс `/api/v1` и требуют `Authorization: Bearer <accessToken>`.

| Раздел | Маршруты |
| --- | --- |
| Auth | `POST /auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`, `GET /auth/me` |
| Dashboard | `GET /dashboard`, `GET /search?q=` |
| Customers | `GET/POST /customers`, `GET/PUT/DELETE /customers/{id}`, `GET /customers/stats` |
| Orders | `GET/POST /orders`, `GET/PATCH /orders/{id}` |
| eSIM | `GET/POST /esims`, `GET/PUT/DELETE /esims/{id}`, `POST /esims/import`, `GET /esims/stats` |
| Catalog | CRUD `/tariffs`, CRUD `/suppliers`, `POST /suppliers/{id}/sync`, `/suppliers/sync-all` |
| Admin | CRUD `/employees`, `GET /audit` |
| Notifications | `GET /notifications`, `PATCH /notifications/{id}/read`, `/notifications/read-all` |
| Reports | `GET /reports/sales.csv?from=YYYY-MM-DD&to=YYYY-MM-DD` |
| Zesimo | `GET /integrations/zesimo`, `POST /integrations/zesimo/sync`, `POST /orders/{id}/provision` |

Списки поддерживают `page`, `limit`, `q` и профильные фильтры. Ответы имеют вид `{"data": ..., "meta": ...}`, ошибки — `{"error":{"code":"...","message":"..."}}`.

Синхронизация поставщика сейчас выполняет внутреннюю сверку остатков, статуса и баланса. Для реальной закупки/синхронизации у BC Telecom, Joytel и Airalo потребуются их API URL, схемы авторизации и контракты ответов; эти секреты и спецификации в исходном фронтенде отсутствуют.

## Проверка

```bash
docker run --rm -v "$PWD/backend:/src" -w /src golang:1.23-alpine go test ./...
docker compose ps
```
