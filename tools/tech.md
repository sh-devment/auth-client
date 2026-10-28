[<- Назад](/README.md)

# auth-client

Шаблон приложения экосистемы sh-development. Вход идёт через auth-center: редирект →
одноразовый code → обмен на данные пользователя на сервере → сессия (JWT-cookie) → запись в SQLite.

## Переменные окружения

| Переменная | Обязательно | Описание |
|---|:---:|---|
| `APP` | | Имя приложения. Показывается в интерфейсе (заголовок вкладки, логин-карточка), от него берутся имя бинаря, `/opt/$APP` и файл базы `<APP>.db`. В проде — repo-level variable, `deploy.sh` всегда пишет её в env-файл. По умолчанию — константа `defaultAppName` в `main.go` |
| `REGION` | | `ru` \| `com`: домен и язык всех строк интерфейса (`build/region-*.go`). В проде приходит из matrix деплоя. По умолчанию `ru` |
| `AUTH_URL` | ★ | Публичный URL auth-center — куда перенаправляется браузер пользователя |
| `AUTH_INTERNAL` | ★ | Внутренний URL auth-center для server-to-server вызовов `/exchange` и `/delegate` |
| `APP_URL` | ★ | Публичный URL этого приложения — auth-center редиректит сюда после входа |
| `APP_TOKEN` | ★ | Секрет приложения — должен совпадать с одним из `APP_TOKENS` в auth-center |
| `SECRET_KEY` | ★ | Ключ подписи cookie-сессий, произвольная строка |
| `DB_DIR` | | Папка, где лежит `<APP>.db`. Имя файла не настраивается — только папка. В dev — `/data` (docker-том), в проде не задаётся: база будет `/opt/$APP/$APP.db` |
| `PORT` | | Порт сервера (по умолчанию `8890`) |

`APP_TOKEN` никогда не попадает в браузер — только в server-to-server запросах.

## Регионы

Каждое приложение живёт на одном и том же поддомене в обеих зонах:
`<app>.sh-development.ru` и `<app>.sh-development.com`. Всё, что отличается между зонами, —
в `build/region-ru.go` и `build/region-com.go`: домен, язык страницы, каждая строка интерфейса.
Новая строка — это поле в `pageText` (или `jsText` для `script.js`) и значение в **обоих**
файлах: пустая строка в любом из них останавливает приложение при старте.

## Локальная разработка

Hot-reload через Air:
```bash
cd auth-client
cp .env.example .env   # заполнить переменные
docker-compose -f dev-compose.yml up --remove-orphans
```

Сервис доступен на `http://localhost:8890`. База — на docker-томе `db-data`.

## Сборка продакшн-бинаря (linux/amd64)

```bash
cd auth-client
docker-compose -f prod-compose.yml build release      # обязательно после любой правки кода
docker-compose -f prod-compose.yml run --rm release   # → bin/auth-client
```

Бинарь коммитится в репозиторий: на сервере сборки нет.

## Деплой

GitHub Actions → `Deploy` (запуск вручную). Workflow — matrix по `com` и `ru`: у каждого региона
свой self-hosted runner (`production-<region>`) и свой GitHub environment. Каждая задача
запускает `bin/deploy.sh`, который:

- кладёт `bin/$APP` в `/opt/$APP/`
- пишет `/opt/$APP/$APP.env` из переменных, перечисленных в `ENV_KEYS`
- генерирует systemd-юнит (`WorkingDirectory=/opt/$APP`)
- перезапускает приложение и проверяет `GET http://127.0.0.1:$PORT/`
- при неудаче откатывается на предыдущую версию. Бэкап базы перед деплоем сохраняется
  в `/opt/$APP/last-deploy-backup`, но автоматически не восстанавливается.

Что нужно завести в GitHub:
- repo-level variable `APP`
- в environments `production-ru` и `production-com`: variables `PORT`, `AUTH_URL`,
  `AUTH_INTERNAL`, `APP_URL`; secrets `APP_TOKEN`, `SECRET_KEY`

Пустая переменная из `ENV_KEYS` останавливает деплой до того, как он что-либо тронет.

## Ночной бэкап

Каждый деплой ставит `bin/backup.sh` в `/opt/$APP/backup.sh` и, если его ещё нет, добавляет
задание в crontab worker'а:

```
0 1 * * * flock /var/lock/backups.lock /opt/$APP/backup.sh $APP >> /backup/$APP/backup.log 2>&1
```

Время у всех приложений одно, общий lock выстраивает их в очередь. Хранятся 7 последних
копий `/backup/$APP/$APP-<дата>.db`. Разовая настройка сервера: `/backup` должен быть доступен
worker'у на запись.

Ручной бэкап (с WAL копировать один `.db` недостаточно):
```bash
sqlite3 /opt/<app>/<app>.db "VACUUM INTO '/backup/<app>/manual.db'"
```

`deploy.sh` и `backup.sh` одинаковые во всех приложениях (эталон — репозиторий `menu`):
исправление в одном копируется во все.
