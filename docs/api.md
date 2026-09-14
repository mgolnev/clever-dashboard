# HTTP API

База: `http://localhost:8080/api`. Ответы — JSON.

## `GET /api/health`

Проверка живости.

```json
{ "status": "ok" }
```

## `POST /api/import`

Прямая загрузка небольшой выгрузки Битрикса. `multipart/form-data`, поле `file`.
Для файлов больше 4 МБ штатный frontend использует порционную загрузку ниже.

```bash
curl -F "file=@sale_order.xls" localhost:8080/api/import
```

Ответ:

```json
{
  "importId": 1,
  "filename": "sale_order.xls",
  "rowsTotal": 1328,
  "ordersImported": 1328,
  "ordersAdded": 1200,
  "ordersUpdated": 128,
  "ordersSkipped": 0,
  "itemsImported": 4823,
  "periodStart": "2025-09-01T12:09:01Z",
  "periodEnd": "2026-05-28T18:34:24Z",
  "ordersCleared": 0
}
```

Импорт накопительный: новые номера заказов добавляются, совпавшие обновляются
вместе с позициями, а отсутствующие в файле заказы сохраняются. Более старый
снимок совпавшего заказа пропускается по `updated_at`. `ordersImported` равен
сумме `ordersAdded + ordersUpdated`; `ordersSkipped` содержит число устаревших
снимков. `ordersCleared` оставлен для совместимости и всегда равен нулю.

## Порционная загрузка больших файлов

`POST /api/import/uploads` создаёт временную сессию:

```json
{ "filename": "sale_order.xls", "size": 13631488 }
```

Ответ `201 Created` задаёт размер и количество частей:

```json
{
  "uploadId": "38d10da8c29b1c7ab9827becc52b6361",
  "filename": "sale_order.xls",
  "size": 13631488,
  "chunkSize": 4194304,
  "chunksTotal": 4
}
```

Каждая часть передаётся как `application/octet-stream`:

```text
PUT /api/import/uploads/{uploadId}/chunks/{index}
```

Индексы начинаются с нуля. После всех частей
`POST /api/import/uploads/{uploadId}/complete` запускает потоковый импорт и
возвращает обычный `ImportResult`. `DELETE /api/import/uploads/{uploadId}`
отменяет незавершённую загрузку. Общий лимит файла — 512 МБ; временные сессии
автоматически очищаются через 24 часа.

## `GET /api/bounds`

Доступный диапазон дат заказов (для инициализации UI).

```json
{ "min": "2025-09-01", "max": "2026-05-28" }
```

## `GET /api/goal?year=YYYY&month=M`

Компактная read-модель вкладки «Цель». Если параметры не заданы, используются
текущие год и месяц. Ответ объединяет годовой план, границы заказов, фактические
итоги выбранного месяца, историческую базу до него и статус источников трафика.

`history` — до 30 последних доступных дней, заканчивающихся не позднее дня перед
началом выбранного месяца. Если более ранних заказов нет, поле равно `null`.

```jsonc
{
  "year": 2026,
  "month": 9,
  "plan": { "year": 2026, "months": [/* 12 месяцев */] },
  "bounds": { "min": "2025-09-01", "max": "2026-09-11" },
  "current": {
    "start": "2026-09-01",
    "end": "2026-09-30",
    "channels": [
      { "channel": "all", "revenue": 1200000, "netRevenue": 780000,
        "aov": 4200, "sessions": 18400, "netCr": 1.31 },
      { "channel": "site", "revenue": 700000, "netRevenue": 450000,
        "aov": 4100, "sessions": 12000, "netCr": 1.25 },
      { "channel": "app", "revenue": 500000, "netRevenue": 330000,
        "aov": 4400, "sessions": 6400, "netCr": 1.41 }
    ]
  },
  "history": {
    "start": "2026-08-02",
    "end": "2026-08-31",
    "channels": [/* all, site, app */]
  },
  "analyticsStatus": { "enabled": true, "sources": [/* metrika, appmetrica */] }
}
```

- `revenue` — сумма неотменённых заказов; используется для AOV, прогноза и
  распределения плана по каналам.
- `netRevenue` — валовая выручка выкупленных заказов минус полные и частичные
  возвраты, по тем же правилам, что `metrics.current.kpi.stages.redeemedNet.revenue`.
- `netCr` — неотменённые заказы / сессии соответствующего канала, %.

Сохранение цели остаётся в `PUT /api/plan`. Архитектурное решение —
[ADR-0008](adr/0008-compact-goal-read-model.md).

## `GET /api/cities` · `GET /api/regions` · `GET /api/channels` · `GET /api/payments` · `GET /api/deliveries` · `GET /api/coupons`

Списки значений для фильтров (город, область/регион, витрина = канал заказа
«Приложение»/«Сайт», способ оплаты `payment_system`, способ доставки
`delivery_service`, промокод `coupon` = поле «Купоны заказа»), отсортированы по
убыванию числа заказов. Используются всеми вкладками. Промокод заполняется при
импорте; для существующих БД значения появятся после повторной загрузки выгрузки.

```json
[
  { "name": "Киров", "orders": 114 },
  { "name": "Екатеринбург", "orders": 81 }
]
```

## `GET /api/metrics?start=YYYY-MM-DD&end=YYYY-MM-DD&compareStart=YYYY-MM-DD&compareEnd=YYYY-MM-DD&city=<город>&region=<область>&channel=<витрина>&payment=<оплата>&delivery=<доставка>`

Метрики за период и за период сравнения. Если `start`/`end` не заданы — последние
7 дней данных. Необязательные `compareStart`/`compareEnd` (формат `YYYY-MM-DD`)
задают второй период явно; если не переданы — предыдущий период той же длины
непосредственно перед текущим (поведение по умолчанию). Необязательные `city`, `region`, `channel`
(витрина: «Приложение»/«Сайт»), `payment` (способ оплаты), `delivery` (способ
доставки) и `coupon` (промокод) поддерживают **мультивыбор**: список значений
через запятую (например `city=Киров,Казань`). Внутри списка — логика **ИЛИ**
(`IN`), между разными фильтрами — **И** (AND). Пустые — без фильтра. Фильтр
применяется и к текущему, и к предыдущему периоду. Те же фильтры поддерживают
`/api/funnel`, `/api/logistics` и `/api/dynamics`.

Структура ответа:

```jsonc
{
  "period":   { "start": "2026-05-22", "end": "2026-05-28", "days": 7 },
  "previous": { "start": "2026-05-15", "end": "2026-05-21", "days": 7 },
  "current":  { "kpi": { ... }, "funnel": [...], "byChannel": [...],
                "byPayment": [...], "byDelivery": [...], "byRegion": [...],
                "topProducts": [...], "byCategory": [...],
                "byGender": [...], "byBrand": [...] },
  "prev":     { "kpi": { ... }, ... }
}
```

`kpi`: `orders`, `netOrders`, `revenue`, `aov`, `asp`, `paidOrders`, `paidRate`,
`canceledOrders`, `canceledRate`, `units`, `customers`, `completed`,
`redeemedGross`, `returnedOrders`, `fullyReturned`, `redeemedNet`,
`refundAmount`, `terminal`, `inTransit`, `g2n`, `redemptionRate`.

- `aov` — средний чек на заказ (revenue / netOrders).
- `asp` — средняя цена позиции (выручка позиций / проданные единицы).
- `redeemedGross` — выкуплено валово: выполненные и затем возвращённые заказы.
- `returnedOrders` / `refundAmount` — количество возвратов и фактически возвращённые средства.
- `redeemedNet` — выкуплено чистыми: валовой выкуп без полностью возвращённых заказов.
- `g2n` — чистый выкуп / оформлено (redeemedNet / orders), %.
- `redemptionRate` — чистый выкуп / terminal (заказы в конечном статусе), %.
- `terminal` — заказы в конечном статусе (completed/canceled/closed/returned).
- `inTransit` — физически в доставке: `shipped` («Отправлен») или `in_pvz` («Прибыл в ПВЗ»).

Выручка (`revenue`) и `units` считаются по **не отменённым** заказам.

`kpi.stages` — абсолюты по стадиям воронки для карточек «Оформлено → Оплачено →
Транзит → Выкуплено валово». Каждая стадия (`created`/`paid`/`inTransit`/`redeemedGross`)
содержит `orders`, `revenue`, `units`, а также производные `aov` (revenue/orders),
`asp` (revenue/units), `upt` (units/orders, позиций на заказ).

```jsonc
"stages": {
  "created":   { "orders": 1328, "revenue": 6912373, "units": 5285, "aov": 5205, "asp": 1307, "upt": 3.98 },
  "paid":      { "orders": 996,  "revenue": 5158789, "units": 4164, "aov": 5179, "asp": 1238, "upt": 4.18 },
  "inTransit": { "orders": 402,  "revenue": 2254181, "units": 1793, "aov": 5607, "asp": 1257, "upt": 4.46 },
  "redeemedGross": { "orders": 594, "revenue": 2851063, "units": 2290, "aov": 4799, "asp": 1245, "upt": 3.86 },
  "returns": { "orders": 8, "revenue": 29000 },
  "redeemedNet": { "orders": 590, "revenue": 2822063, "units": 2270, "aov": 4783, "asp": 1243, "upt": 3.85 }
}
```

- `created` (оформлено) — все заказы периода (гросс, выручка по `total_amount`).
- `paid` (стадия «Оплачено») — заказы с `is_paid` либо со статусом `paid` и последующими стадиями; отдельный `kpi.paidOrders` остаётся строгим фактом оплаты по `is_paid`.
- `inTransit` (транзит) — физически в доставке (`shipped`/`in_pvz`), ещё не выкуплены.
- `redeemedGross` (выкуплено валово) — `completed` и `returned`: заказ был выдан клиенту.
- `returns` — отдельный исход; `orders` — все возвраты, `revenue` — возвращённые деньги.
- `redeemedNet` (выкуплено чистыми) — валовой выкуп без полных возвратов; `revenue` уменьшается на полные и частичные возвраты.
- `terminal` — заказы в конечном статусе (`completed/canceled/closed/returned`); знаменатель для «в кон. статусе».
- `paidTerminal` — оплаченные **и** в конечном статусе; знаменатель P2N «в кон. статусе».

UI показывает долю стадии от «Оформлено» для суммируемых метрик (выручка,
заказы, товары); для средних (`aov`/`asp`/`upt`) доля не выводится.

Коэффициенты выкупа (считаются на фронте из стадий, для суммируемых метрик):

- **G2N всего** = `redeemedNet / created` — чистый выкуп к оформленным (с учётом транзита).
- **G2N в кон. статусе** = `redeemedNet / terminal` — чистый выкуп среди дошедших до конца.
- **P2N всего** = `redeemedNet / paid` — чистый выкуп к оплаченным.
- **Возврат опл.** = `1 − redeemedNet / paidTerminal` — доля оплаченных заказов,
  которые **не** выкуплены (отмена/возврат) среди дошедших до конечного статуса;
  рост = плохо.

## `GET /api/customer-analytics?start=YYYY-MM-DD&end=YYYY-MM-DD&granularity=<month|quarter>&city=<город>&region=<область>&channel=<витрина>&payment=<оплата>&delivery=<доставка>&coupon=<промокод>`

Когортный retention клиентов. Сквозные фильтры имеют ту же семантику
мультивыбора, что `/api/metrics`. Вкладка «Клиенты» передаёт в `start` начало
всей доступной истории, а в `end` — конец выбранного пользователем периода.
`granularity` задаёт календарный месяц (по умолчанию) или квартал.

Ответ содержит две независимые матрицы: `gross` (первый/повторный оформленный
заказ) и `paid` (первый/повторный заказ со строгим `is_paid=true`). Клиент
идентифицируется по email, затем телефону, затем имени. `M0`/`Q0` — период
первой активности, `M1`/`Q1` — следующий календарный период.

```jsonc
{
  "period": { "start": "2026-01-01", "end": "2026-09-14" },
  "granularity": "month",
  "gross": {
    "mode": "gross",
    "summary": {
      "customers": 1840,
      "cohortCount": 9,
      "repeatCustomers": 316,
      "repeatRate": 17.2,
      "retention": [
        { "offset": 0, "customers": 1840, "eligibleCustomers": 1840, "rate": 100 },
        { "offset": 1, "customers": 132, "eligibleCustomers": 1610, "rate": 8.2 }
      ]
    },
    "cohorts": [{
      "month": "2026-01-01",
      "customers": 210,
      "cells": [{
        "offset": 1, "month": "2026-02-01", "customers": 19,
        "rate": 9.0, "available": true, "complete": true
      }]
    }],
    "composition": [{
      "period": "2026-08-01", "customers": 520,
      "new": 360, "active": 95, "returned": 65,
      "newRate": 69.2, "activeRate": 18.3, "returnedRate": 12.5,
      "complete": true
    }]
  },
  "paid": { "mode": "paid", "summary": { "...": "..." }, "cohorts": [] }
}
```

`summary.retention` — взвешенный retention: знаменатель включает только когорты,
которые успели прожить указанное число полных календарных месяцев. Ячейка с
`available=true, complete=false` относится к текущему незавершённому месяцу:
она видна в матрице, но не включена в агрегированный KPI.

`composition` раскладывает всех уникальных клиентов периода на три
взаимоисключающие группы: `new` — первый заказ; `active` — заказ есть и в
текущем, и в непосредственно предыдущем периоде; `returned` — в предыдущем
периоде заказа нет, но есть заказ в более ранней истории. Для `paid` учитываются
только оплаченные заказы. Сумма трёх групп равна `customers`.

Вариант «в конечном статусе» исключает транзит и не искажается свежими заказами,
которые ещё в пути. Сам по себе `completed / paidTerminal` в UI не выводится (при
предоплате он ≈100%), но его дополнение — «Возврат опл.» — показывает реальные
возвраты/невыкуп после оплаты.

## `GET /api/logistics?start=YYYY-MM-DD&end=YYYY-MM-DD&compareStart=YYYY-MM-DD&compareEnd=YYYY-MM-DD&city=<город>&region=<область>&granularity=<day|week|month>`

Метрики доставки для пилота «бесплатная доставка / без порога». Структура как у
`/api/metrics`: текущий и период сравнения, те же фильтры `city` / `region`.
Необязательные `compareStart`/`compareEnd` (формат `YYYY-MM-DD`) задают второй
период явно; по умолчанию — предыдущий период той же длины. Необязательный
`granularity` задаёт шаг `series`: `day`, `week` (по умолчанию) или `month`.

```jsonc
{
  "period": { "start": "...", "end": "...", "days": 7 },
  "previous": { "start": "...", "end": "...", "days": 7 },
  "dataAsOf": "2026-09-13",       // дата актуальности снимка заказов
  "pilotCities": ["Пермь", "Киров"],  // из LOGISTICS_PILOT_CITIES
  "pilotStart": "2026-06-01",         // из LOGISTICS_PILOT_START (опционально)
  "current": {
    "summary": {
      "orders": 0, "revenue": 0, "paidOrders": 0, "paidRate": 0,
      "processedOrders": 0, "shippedOrders": 0, "pendingShipment": 0,
      "shipmentRate": 0,
      "pending0To1": 0, "pending2To3": 0, "pending4Plus": 0,
      "avgPendingAgeDays": 0,
      "deliveryTotal": 0, "avgDelivery": 0, "freeOrders": 0, "freeDeliveryRate": 0
    },
    "byService": [{ "name": "...", "orders": 0, "share": 0, "paidOrders": 0, "paidRate": 0,
      "processedOrders": 0, "shippedOrders": 0, "pendingShipment": 0, "shipmentRate": 0,
      "pending0To1": 0, "pending2To3": 0, "pending4Plus": 0, "avgPendingAgeDays": 0,
      "revenue": 0, "deliveryTotal": 0, "avgDelivery": 0, "freeOrders": 0, "freeDeliveryRate": 0 }],
    "byCity": [{ "name": "...", "isPilot": true, "orders": 0, "share": 0, "paidOrders": 0,
      "paidRate": 0, "revenue": 0, "deliveryTotal": 0, "avgDelivery": 0, "freeOrders": 0, "freeDeliveryRate": 0 }],
    "cohorts": { "pilot": { ...summary }, "control": { ...summary } },
    "series": [{ "week": "2026-05-19", "orders": 0, "netOrders": 0, "paidOrders": 0,
      "revenue": 0, "units": 0, "aov": 0, "asp": 0, "upt": 0, "paidRate": 0,
      "avgDelivery": 0, "freeDeliveryRate": 0, "deliveryTotal": 0 }]
  },
  "prev": { ... }
}
```

- `orders` — гросс-заказы периода; `revenue` — сумма `total_amount` не отменённых.
- `processedOrders` — заказы периода, дошедшие до обработки: текущие стадии
  `processing`, `shipped`, `in_pvz`, `completed`, `returned`.
- `shippedOrders` — дошедшие до отправки: `shipped`, `in_pvz`, `completed`,
  `returned`; `pendingShipment = processedOrders − shippedOrders`, а
  `shipmentRate = shippedOrders / processedOrders`.
- `pending0To1` / `pending2To3` / `pending4Plus` — текущий неотправленный хвост
  по возрасту в календарных днях с `created_at` до `dataAsOf`;
  `avgPendingAgeDays` — средний возраст того же хвоста. Это индикатор риска, а
  не точный SLA «сборка → отправка»: история переходов статуса в выгрузке не
  сохраняется.
- `paidRate` — оплаченные / гросс (%), прокси «конверсии» в данных Битрикса.
- `avgDelivery` — среднее `delivery_cost` на заказ; `freeDeliveryRate` — доля с
  `delivery_cost = 0`.
- `share` — доля заказов сегмента от всех заказов периода (службы — от суммы
  показанных служб, города — от суммы показанных городов).
- `byService[].paidRate` / `byCity[].paidRate` — оплаченные / заказы сегмента (%).
- `cohorts` — только если задан `LOGISTICS_PILOT_CITIES`: пилотные города vs
  остальные (с учётом фильтра `region`, без фильтра `city`).
- `series` — временные бакеты (поле `week`: день YYYY-MM-DD, понедельник недели или
  1-е число месяца) со всеми метриками
  (`orders`, `paidOrders`, `revenue`, `units`, `aov`, `asp`, `upt`, `paidRate`,
  `avgDelivery`, `freeDeliveryRate`); UI переключает их в графике динамики.
  `aov`/`upt` считаются от не отменённых (`netOrders`), `asp` — выручка позиций /
  проданные единицы.

## `GET /api/dynamics?start=YYYY-MM-DD&end=YYYY-MM-DD&groupBy=<измерение>&granularity=<day|week|month>&<фильтры>`

Динамика в разрезе измерения (для вкладки «Динамика»). `groupBy` —
одно из `city` / `region` / `delivery` / `payment` / `channel` / `coupon`.
Необязательный `granularity` — `day`, `week` (по умолчанию) или `month`.
Поддерживает те же
фильтры (`city`/`region`/`channel`/`payment`/`delivery`, мультивыбор). Возвращает
топ-8 значений измерения по числу заказов; точки каждого значения выровнены по
общему списку бакетов (`weeks`), пропуски — нулевые точки. Каждая точка содержит
тот же набор метрик, что и `series` (заказы/оплаты/выручка/товары/чек/ASP/UPT/
ср. доставка/бесплатно %), поэтому переключение метрики на фронте не требует
повторного запроса.

```jsonc
{
  "period": { "start": "...", "end": "...", "days": 7 },
  "weeks": ["2026-05-11", "2026-05-18"],
  "groups": [
    { "name": "Москва", "points": [ { "week": "2026-05-11", "orders": 0, /* ... */ } ] },
    { "name": "Киров",  "points": [ /* ... выровнены по weeks ... */ ] }
  ]
}
```

## `GET /api/funnel?start=YYYY-MM-DD&end=YYYY-MM-DD&compareStart=YYYY-MM-DD&compareEnd=YYYY-MM-DD&city=<город>&region=<область>&channel=<витрина>&payment=<оплата>&delivery=<доставка>`

Воронка пути заказа за период. Пустые даты — последняя неделя данных.
Необязательные `compareStart`/`compareEnd` (формат `YYYY-MM-DD`) задают второй
период для сравнения стадий; по умолчанию — предыдущий период той же длины.
Необязательные `city`, `region`, `channel`, `payment` и `delivery` поддерживают
мультивыбор (список через запятую): внутри списка — `IN` (ИЛИ), между фильтрами —
AND. Фильтруют все стадии, разрезы и топы.

Каждая стадия содержит, кроме `orders`, кумулятивную `revenue` (сумма
`total_amount` заказов, дошедших до стадии) и `units` (сумма позиций `qty` тех же
заказов). Фронт переключает отображение воронки по этим метрикам; `fromStart`/
`fromPrev` в ответе считаются по заказам.

```jsonc
{
  "period": { "start": "...", "end": "...", "days": 7 },
  "previous": { "start": "...", "end": "...", "days": 7 },
  "stages": [   // кумулятивные стадии: заказ дошёл хотя бы до стадии
    { "key": "created", "label": "Создан (гросс)", "orders": 426, "revenue": 2100000, "units": 1680, "fromStart": 100, "fromPrev": 100 },
    { "key": "paid",    "label": "Оплачен",        "orders": 300, "revenue": 1550000, "units": 1210, "fromStart": 70.4, "fromPrev": 70.4 }
    // ... processing | shipped | delivered | completed
  ],
  "prevStages": [ /* те же поля, что stages — стадии периода сравнения */ ],
  "gross": 426, "canceled": 126, "returns": 8, "problems": 12, "canceledNoReason": 109,
  "segments": [  // by: payment | delivery | channel | region
    { "by": "payment", "label": "Способ оплаты", "rows": [
      { "name": "...", "gross": 0, "paid": 0, "paidRate": 0, "completed": 0,
        "completedRate": 0, "canceled": 0, "cancelRate": 0, "problems": 0, "revenue": 0 }
    ]}
  ],
  "topProblems":      [ { "label": "...", "orders": 0 } ],
  "topCancelReasons": [ { "label": "...", "orders": 0 } ]
}
```

Стадии **кумулятивны** (заказ учитывается, если дошёл хотя бы до стадии).
`paidRate`/`cancelRate`/`completedRate` в разрезах считаются от гросс данного
сегмента.

## `GET /api/acquisition?start=YYYY-MM-DD&end=YYYY-MM-DD&compareStart=YYYY-MM-DD&compareEnd=YYYY-MM-DD`

Сохранённый верх воронки: визиты/сессии, дневные пользователи и конверсия в заказы
Битрикса. Период сравнения работает так же, как в `/api/metrics`. Бизнес-фильтры
не принимаются: внешний агрегированный трафик нельзя корректно фильтровать по
полям заказа.

```jsonc
{
  "period": { "start": "2026-08-10", "end": "2026-08-16", "days": 7 },
  "previous": { "start": "2026-08-03", "end": "2026-08-09", "days": 7 },
  "current": {
    "hasTraffic": true,
    "sampled": false,
    "channels": [
      {
        "channel": "site", "label": "Сайт", "sessions": 12000, "users": 8400,
        "orders": 128, "paidOrders": 96, "netOrders": 110,
        "orderCr": 1.07, "paidCr": 0.8, "netCr": 0.92,
        "ecommerceAvailable": true,
        "trackedPurchaseUsers": 70,
        "ecommerceFunnel": [
          { "key": "product_view", "label": "Просмотрели товар", "count": 6088,
            "unit": "пользователи", "fromPrevious": 100 },
          { "key": "add_to_cart", "label": "Добавили в корзину", "count": 680,
            "unit": "пользователи", "fromPrevious": 11.17 },
          { "key": "created", "label": "Созданные заказы", "count": 128,
            "unit": "заказы", "fromPrevious": 18.82, "fromCreated": 100 },
          { "key": "paid", "label": "Оплачено", "count": 96,
            "unit": "заказы", "fromPrevious": 75, "fromCreated": 75 }
        ]
      }
    ],
    "daily": [
      { "day": "2026-08-10", "siteSessions": 1700, "appSessions": 900,
        "siteUsers": 1200, "appUsers": 500,
        "siteOrders": 18, "appOrders": 41,
        "sitePaidOrders": 12, "appPaidOrders": 29 }
    ]
  },
  "prev": { "hasTraffic": true, "sampled": false, "channels": [], "daily": [] }
}
```

- `orderCr` = все созданные заказы / визиты или сессии.
- `paidCr` = заказы со строгим `is_paid` / визиты или сессии.
- `netCr` = неотменённые заказы / визиты или сессии.
- `daily.*PaidOrders` = дневное число заказов со строгим `is_paid`; поля нужны
  для динамики конверсии трафика в оплату.
- `users` и `daily.*Users` — сумма дневной аудитории. Повторный пользователь в
  разные дни и один пользователь сайта и приложения учитываются повторно.
- Канал `all` суммирует сайт и приложение без межканальной дедупликации.
- `ecommerceFunnel` объединяет аудиторию событий Яндекса с каноническими
  заказами Битрикса. Для `paid` поле `fromCreated` всегда равно
  `paidOrders / orders`; `trackedPurchaseUsers` служит только для сверки трекинга.
- Этап `begin_checkout` возвращается для приложения при наличии данных. В общей
  и веб-воронке он пропущен, пока нет сопоставимого надёжного этапа сайта.
- Сессии и пользователи приложения загружаются публичным Reporting API по запросу, который
  экспортирует стандартный отчёт AppMetrica «Вовлечённость → Сессии».

## `GET /api/analytics/status`

Состояние фоновой синхронизации без раскрытия ID и токенов.

```jsonc
{
  "enabled": true,
  "syncing": false,
  "sources": [
    { "source": "metrika", "channel": "site", "configured": true,
      "status": "success", "dateFrom": "2026-08-10", "dateTo": "2026-08-16",
      "rowsImported": 7, "lastDataDay": "2026-08-16", "finishedAt": "..." },
    { "source": "appmetrica", "channel": "app", "configured": false,
      "status": "never", "rowsImported": 0 }
  ]
}
```

## `POST /api/analytics/sync`

Неблокирующий ручной запуск обновления дневного трафика из Яндекс Метрики и
AppMetrica. Endpoint сразу отвечает `202`, а выполнение продолжается в фоне:

```json
{ "started": true, "syncing": true }
```

Если синхронизация уже выполняется, новый параллельный запуск не создаётся:
`started` будет `false`, а `syncing` — `true`. Текущий прогресс и результат
читаются через `GET /api/analytics/status`; сохранённые данные меняются только
после успешного ответа внешнего источника.

## `GET /api/plan?year=YYYY` · `PUT /api/plan`

План продаж NET (выкупленная выручка) по месяцам и каналам. Параметр `year`
необязателен — по умолчанию текущий год (2000..2100).

**GET** — всегда 12 месяцев, даже если в БД пусто (нули):

```jsonc
{
  "year": 2026,
  "months": [
    {
      "month": 1,
      "daysInMonth": 31,
      "targets": { "all": 4500000, "site": 3000000, "app": 1500000 },
      "perDay":  { "all": 145161, "site": 96774, "app": 48387 }
    }
    // ... месяцы 2..12
  ]
}
```

`perDay` = округление `net_target / daysInMonth`. Каналы: `all` | `site` | `app`.

**PUT** — upsert элементов, ответ как у GET:

```jsonc
{
  "year": 2026,
  "items": [
    { "month": 1, "channel": "all", "netTarget": 4500000 },
    { "month": 1, "channel": "site", "netTarget": 3000000 }
  ]
}
```

Валидация: `month` 1..12, `channel` ∈ {all, site, app}, `netTarget` ≥ 0.

### Формула достижения плана (расчёт на фронте)

Во вкладке «Цель» пользователь вводит только общий месячный `NET`. Клиент
автоматически рассчитывает и сохраняет цели `site`/`app` пропорционально
фактической выручке неотменённых заказов за последние доступные 30 дней перед
целевым месяцем. Если такой истории нет, используется факт выбранного месяца;
при полном отсутствии продаж показывается явный временный fallback 50/50.

Сведение `/api/plan`, `/api/metrics`, `/api/acquisition` и
`/api/analytics/status` выполняется в отдельном frontend-hook; атомарные backend-
сервисы не вызывают друг друга.

- **NET (факт)** = `kpi.stages.redeemedNet.revenue` за выбранный месяц: валовой
  выкуп минус фактические возвраты.
- **CR в заказ** = `AcquisitionChannel.netCr`: неотменённые заказы / визиты или
  сессии из сохранённой аналитики.
- **AOV** = `kpi.revenue` / `kpi.netOrders`.
- **G2N** = `kpi.stages.redeemedNet.revenue` / `kpi.revenue` на последнем
  доступном историческом периоде.
- **Ожидаемый NET текущих заказов** = `kpi.revenue × G2N`, но не меньше уже
  зафиксированного факта NET.
- **Прогноз месяца** = ожидаемый NET / прошедшие календарные дни × дней в месяце.
- **Нужно визитов в день** = оставшийся NET / оставшиеся дни /
  `(CR × AOV × G2N)`.

Если трафик не загружен, денежный прогресс остаётся доступен, а зависимые
показатели CR и необходимых визитов показываются как отсутствующие данные без
ручного ввода.

## `GET /api/traffic?year=YYYY` · `PUT /api/traffic`

Помесячный контракт трафика для обратной совместимости. Автоматические дневные
записи Метрики и AppMetrica агрегируются по месяцу и имеют приоритет; ручные
значения могут использоваться внешними клиентами как fallback. Вкладка «Цель»
ручной трафик больше не запрашивает и берёт факт из `/api/acquisition`.

**GET**:

```jsonc
{
  "year": 2026,
  "months": [
    { "month": 1, "site": 50000, "app": 12000,
      "siteSource": "metrika", "appSource": "appmetrica" }
    // ... 12 месяцев
  ]
}
```

**PUT** — upsert, `channel` только `site` | `app`:

```jsonc
{
  "year": 2026,
  "items": [
    { "month": 1, "channel": "site", "visits": 50000 },
    { "month": 1, "channel": "app", "visits": 12000 }
  ]
}
```

`PUT` всегда сохраняет `source=manual` и не перезаписывает дневные авто-данные.
