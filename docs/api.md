# HTTP API

База: `http://localhost:8080/api`. Ответы — JSON.

## `GET /api/health`

Проверка живости.

```json
{ "status": "ok" }
```

## `POST /api/import`

Загрузка файла выгрузки Битрикса. `multipart/form-data`, поле `file`.

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
  "itemsImported": 4823,
  "periodStart": "2025-09-01T12:09:01Z",
  "periodEnd": "2026-05-28T18:34:24Z"
}
```

## `GET /api/bounds`

Доступный диапазон дат заказов (для инициализации UI).

```json
{ "min": "2025-09-01", "max": "2026-05-28" }
```

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

## `GET /api/metrics?start=YYYY-MM-DD&end=YYYY-MM-DD&city=<город>&region=<область>&channel=<витрина>&payment=<оплата>&delivery=<доставка>`

Метрики за период и за предыдущий период той же длины. Если `start`/`end` не
заданы — последние 7 дней данных. Необязательные `city`, `region`, `channel`
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
`canceledOrders`, `canceledRate`, `units`, `customers`, `completed`, `terminal`,
`inTransit`, `g2n`, `redemptionRate`.

- `aov` — средний чек на заказ (revenue / netOrders).
- `asp` — средняя цена позиции (выручка позиций / проданные единицы).
- `g2n` — выкуплено / оформлено (completed / orders), %.
- `redemptionRate` — выкупаемость: completed / terminal (заказы в конечном статусе), %.
- `terminal` — заказы в конечном статусе (completed/canceled/closed/returned).
- `inTransit` — заказы «в пути» (new/processing/shipped/in_pvz), ещё не дошли до выкупа/отмены.

Выручка (`revenue`) и `units` считаются по **не отменённым** заказам.

`kpi.stages` — абсолюты по стадиям воронки для карточек «Оформлено → Оплачено →
Транзит → Выкуплено». Каждая стадия (`created`/`paid`/`inTransit`/`completed`)
содержит `orders`, `revenue`, `units`, а также производные `aov` (revenue/orders),
`asp` (revenue/units), `upt` (units/orders, позиций на заказ).

```jsonc
"stages": {
  "created":   { "orders": 1328, "revenue": 6912373, "units": 5285, "aov": 5205, "asp": 1307, "upt": 3.98 },
  "paid":      { "orders": 996,  "revenue": 5158789, "units": 4164, "aov": 5179, "asp": 1238, "upt": 4.18 },
  "inTransit": { "orders": 402,  "revenue": 2254181, "units": 1793, "aov": 5607, "asp": 1257, "upt": 4.46 },
  "completed": { "orders": 586,  "revenue": 2822063, "units": 2270, "aov": 4815, "asp": 1243, "upt": 3.87 }
}
```

- `created` (оформлено) — все заказы периода (гросс, выручка по `total_amount`).
- `paid` (оплачено) — заказы с `is_paid`.
- `inTransit` (транзит) — заказы не в конечном статусе (`new/processing/shipped/in_pvz`), ещё в пути.
- `completed` (выкуплено) — заказы со `status_stage='completed'`.
- `terminal` — заказы в конечном статусе (`completed/canceled/closed/returned`); знаменатель для «в кон. статусе».
- `paidTerminal` — оплаченные **и** в конечном статусе; знаменатель P2N «в кон. статусе».

UI показывает долю стадии от «Оформлено» для суммируемых метрик (выручка,
заказы, товары); для средних (`aov`/`asp`/`upt`) доля не выводится.

Коэффициенты выкупа (считаются на фронте из стадий, для суммируемых метрик):

- **G2N всего** = `completed / created` — выкуплено к оформленным (с учётом транзита).
- **G2N в кон. статусе** = `completed / terminal` — выкуплено среди дошедших до конца.
- **P2N всего** = `completed / paid` — выкуплено к оплаченным.
- **Возврат опл.** = `1 − completed / paidTerminal` — доля оплаченных заказов,
  которые **не** выкуплены (отмена/возврат) среди дошедших до конечного статуса;
  рост = плохо.

Вариант «в конечном статусе» исключает транзит и не искажается свежими заказами,
которые ещё в пути. Сам по себе `completed / paidTerminal` в UI не выводится (при
предоплате он ≈100%), но его дополнение — «Возврат опл.» — показывает реальные
возвраты/невыкуп после оплаты.

## `GET /api/logistics?start=YYYY-MM-DD&end=YYYY-MM-DD&city=<город>&region=<область>&granularity=<day|week|month>`

Метрики доставки для пилота «бесплатная доставка / без порога». Структура как у
`/api/metrics`: текущий и предыдущий период той же длины, те же фильтры `city` /
`region`. Необязательный `granularity` задаёт шаг `series`: `day`, `week`
(по умолчанию) или `month`.

```jsonc
{
  "period": { "start": "...", "end": "...", "days": 7 },
  "previous": { "start": "...", "end": "...", "days": 7 },
  "pilotCities": ["Пермь", "Киров"],  // из LOGISTICS_PILOT_CITIES
  "pilotStart": "2026-06-01",         // из LOGISTICS_PILOT_START (опционально)
  "current": {
    "summary": {
      "orders": 0, "revenue": 0, "paidOrders": 0, "paidRate": 0,
      "deliveryTotal": 0, "avgDelivery": 0, "freeOrders": 0, "freeDeliveryRate": 0
    },
    "byService": [{ "name": "...", "orders": 0, "share": 0, "paidOrders": 0, "paidRate": 0,
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

## `GET /api/funnel?start=YYYY-MM-DD&end=YYYY-MM-DD&city=<город>&region=<область>&channel=<витрина>&payment=<оплата>&delivery=<доставка>`

Воронка пути заказа за период. Пустые даты — последняя неделя данных.
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
  "stages": [   // кумулятивные стадии: заказ дошёл хотя бы до стадии
    { "key": "created", "label": "Создан (гросс)", "orders": 426, "revenue": 2100000, "units": 1680, "fromStart": 100, "fromPrev": 100 },
    { "key": "paid",    "label": "Оплачен",        "orders": 300, "revenue": 1550000, "units": 1210, "fromStart": 70.4, "fromPrev": 70.4 }
    // ... processing | shipped | delivered | completed
  ],
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
