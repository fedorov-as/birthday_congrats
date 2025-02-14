# Сервис напоминаний о днях рождения

## Краткое описание

В приложении можно создать аккаунт через форму регистрации на главной странице или войти в существующий аккаунт там же.

После регистрации или входа появляется список сотрудников, где можно подписаться на любого и выбрать для каждого количество дней, за сколько оповестить о дне рождения (на почту), а также можно отменить уже существующую подписку.

Фронт реализован при помощи html-шаблонов.

База данных разворачивается из докер-контейнера с помощью утилиты `docker-compose`.

## Запуск и остановка приложения

В корне репозитория находятся файлы `start_db.sh` и `start_service.sh` со скриптами для запуска базы данных и сервиса.

Перед запуском необходимо убедиться, что порты `3306` и `8080` свободны, так как их используют базы данных и само приложение. Далее, находясь в корне репозитория, выполнить:
```bash
./start_db.sh
```
Дождаться запуска базы данных и в другом терминале запустить сервер:
```bash
./start_service.sh
```

Чтобы остановить приложение, нажать `Enter`. База данных останавливается через `Ctrl+C`.

## Запуск тестов

Находясь в корне репозитория выполнить:
```bash
./run_tests.sh
```

## Описание каталогов

Сам проект лежит в директории `birthday_congrats`. Далее описание будет идти относительно нее.

- `cmd/birthday_congrats` - здесь лежит функция `main()`
- `databases` - папка с SQL-скриптами и `docker-compose` для запуска базы данных
- `internal/pkg` - модули проекта

    - `alert_manger` - менеджер оповещений (на электронную почту)
    - `handlers` - http-хендлеры
    - `middleware` - миддлверы (отлов паники, логгер, проверка авторизации)
    - `session` - описание и менеджер сессий (хранятся в бд)
    - `subscription` - описание и хранилище подписок (хранятся в бд)
    - `user` - описание и хранилище пользователей (хранятся в бд)

- `internal/service` - сам сервис (бизнес-логика)
- `templates` - html-шаблоны страниц

В каталогах также лежат тесты на соответствующие модули. Тестами покрыл модули `user`, `subscription`, `session`, `service` (не полностью), `handlers`.

Настроить некоторые параметры можно с помощью констант в файле `main.go` (описание приведено в коментариях).