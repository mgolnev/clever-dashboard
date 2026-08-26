# Настройка Яндекс Метрики и AppMetrica

Интеграция загружает только обезличенные дневные агрегаты. OAuth-токены нужны
backend-процессу и не должны попадать во frontend, git или сообщения.

## Что подготовить

- Яндекс-аккаунт с правом просмотра нужного счётчика Метрики.
- Тот же или другой Яндекс-аккаунт с правом просмотра приложения AppMetrica.
- ID счётчика Метрики — число из настроек счётчика.
- Numeric Application ID AppMetrica — число из настроек приложения, не API key SDK.

## 1. Создать OAuth-приложение

1. Откройте [создание приложения в Яндекс ID](https://oauth.yandex.ru/client/new).
2. Укажите понятное имя, например `CLEVER Dashboard Analytics`.
3. Выберите платформу **Веб-сервисы**.
4. В Redirect URI добавьте `https://oauth.yandex.com/verification_code`.
5. В доступах добавьте `metrika:read` и `appmetrica:read`.
6. Сохраните приложение и скопируйте `ClientID`. Client secret этому backend не
   нужен: используется пользовательский OAuth access token.

Официальные инструкции: [Метрика](https://yandex.com/dev/metrika/en/intro/authorization),
[AppMetrica](https://appmetrica.yandex.com/docs/en/mobile-api/intro/authorization).

## 2. Выпустить токен

Подставьте ClientID и откройте ссылку в браузере, войдя под аккаунтом с доступом
к счётчику и приложению:

```text
https://oauth.yandex.ru/authorize?response_type=token&client_id=ВАШ_CLIENT_ID
```

Разрешите доступ и скопируйте полученный access token. Один токен с обоими
правами можно указать в обе переменные. Раздельные токены удобнее для независимой
ротации, но не обязательны.

## 3. Найти и проверить ID

Счётчик Метрики можно проверить так:

```bash
curl -sS 'https://api-metrika.yandex.net/management/v1/counters' \
  -H "Authorization: OAuth ВАШ_ТОКЕН"
```

В ответе найдите нужный сайт и его поле `id`.

Приложения AppMetrica:

```bash
curl -sS 'https://api.appmetrica.yandex.com/management/v1/applications' \
  -H "Authorization: OAuth ВАШ_ТОКЕН"
```

Нужен числовой ID приложения из ответа. Не используйте API key, встроенный в
мобильное приложение: это другой секрет и для Reporting API он не подходит.

## 4. Проверить чтение статистики

Подставьте реальные значения и завершённый диапазон дат.

```bash
curl -sS -G 'https://api-metrika.yandex.net/stat/v1/data' \
  -H "Authorization: OAuth ВАШ_ТОКЕН" \
  --data-urlencode 'ids=ID_СЧЁТЧИКА' \
  --data-urlencode 'date1=2026-08-01' \
  --data-urlencode 'date2=2026-08-07' \
  --data-urlencode 'dimensions=ym:s:date' \
  --data-urlencode 'metrics=ym:s:visits,ym:s:users' \
  --data-urlencode 'accuracy=full'
```

```bash
curl -sS -G 'https://api.appmetrica.yandex.ru/stat/v1/data' \
  -H "Authorization: OAuth ВАШ_ТОКЕН" \
  --data-urlencode 'ids=ID_ПРИЛОЖЕНИЯ' \
  --data-urlencode 'date1=2026-08-01' \
  --data-urlencode 'date2=2026-08-07' \
  --data-urlencode 'group=Day' \
  --data-urlencode 'dimensions=ym:s:date' \
  --data-urlencode 'metrics=ym:s:sessions,ym:s:users' \
  --data-urlencode 'accuracy=medium' \
  --data-urlencode 'include_undefined=true' \
  --data-urlencode 'currency=RUB' \
  --data-urlencode 'sort=-ym:s:sessions' \
  --data-urlencode 'lang=ru' \
  --data-urlencode 'request_domain=ru'
```

Запрос AppMetrica повторяет экспорт таблицы стандартного отчёта
«Вовлечённость → Сессии». Часовой пояс в него явно не передаётся: Reporting API
использует настройку приложения. Публичный Reporting API остаётся источником
данных для автоматизации; новый интерфейс AppMetrica может показывать немного
другой итог, хотя экспортирует этот же API-запрос.

При исторической загрузке коннектор делает такие запросы блоками не более семи
дней. AppMetrica может уточнять дневные сессии в зависимости от длины выбранного
периода; недельные блоки воспроизводят стандартную недельную сверку в UI и затем
без потерь складываются в локальную дневную таблицу.

Ожидается HTTP 200 и массив `data`. Пустой массив означает, что за диапазон нет
данных. `401` — токен не передан; `403` — токен не имеет нужного scope либо
аккаунту не дали доступ к счётчику/приложению.

## 5. Добавить секреты в Amvera

В проекте Amvera откройте **Настройки → Переменные** и добавьте:

```env
ANALYTICS_SYNC_ENABLED=true
METRIKA_COUNTER_ID=12345678
METRIKA_OAUTH_TOKEN=...
APPMETRICA_APPLICATION_ID=123456
APPMETRICA_OAUTH_TOKEN=...
ANALYTICS_TIMEZONE=Europe/Moscow
ANALYTICS_SYNC_INTERVAL=6h
ANALYTICS_SYNC_LOOKBACK_DAYS=7
ANALYTICS_BACKFILL_DAYS=365
```

После сохранения переменных перезапустите/пересоберите приложение. Backend сразу
начнёт backfill до вчерашнего дня, затем будет обновляться каждые 6 часов. Статус
виден на вкладке **Трафик и CR**. Данные хранятся в `/data/clever.db` и переживают
пересборку контейнера.

Тем же read-only токеном отдельный фоновый сервис загружает E-commerce этапы:
просмотр товара, добавление в корзину, начало оформления (если источник отдаёт)
и диагностический purchase. Итог воронки не берётся из purchase Яндекса:
созданные и оплаченные заказы всегда считаются по текущей выгрузке Битрикса.

## Безопасность и ротация

- Никогда не добавляйте токены в переменные `VITE_*`: Vite встраивает их в JS.
- Не коммитьте рабочий `.env`.
- При утечке отзовите токен в Яндекс ID, выпустите новый и замените секрет Amvera.
- Для интеграции достаточно прав чтения; `metrika:write` и `appmetrica:write` не нужны.
