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

## `GET /api/cities`

Список городов для фильтра, отсортирован по убыванию числа заказов. Используется
вкладками «Обзор» и «Воронки».

```json
[
  { "name": "Киров", "orders": 114 },
  { "name": "Екатеринбург", "orders": 81 }
]
```

## `GET /api/metrics?start=YYYY-MM-DD&end=YYYY-MM-DD&city=<город>`

Метрики за период и за предыдущий период той же длины. Если `start`/`end` не
заданы — последние 7 дней данных. Необязательный `city` фильтрует заказы по
точному совпадению города (пустой `city` — без фильтра); фильтр применяется и к
текущему, и к предыдущему периоду.

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

## `GET /api/funnel?start=YYYY-MM-DD&end=YYYY-MM-DD&city=<город>`

Воронка пути заказа за период. Пустые даты — последняя неделя данных.
Необязательный `city` фильтрует все стадии, разрезы и топы по городу.

```jsonc
{
  "period": { "start": "...", "end": "...", "days": 7 },
  "stages": [   // кумулятивные стадии: заказ дошёл хотя бы до стадии
    { "key": "created",    "label": "Создан (гросс)",     "orders": 426, "fromStart": 100, "fromPrev": 100 },
    { "key": "paid",       "label": "Оплачен",            "orders": 300, "fromStart": 70.4, "fromPrev": 70.4 },
    { "key": "processing", "label": "В сборке/обработке",  "orders": 301, "fromStart": 70.7, "fromPrev": 100.3 },
    { "key": "shipped",    "label": "Отправлен",          "orders": 288, "fromStart": 67.6, "fromPrev": 95.7 },
    { "key": "delivered",  "label": "Доставлен в ПВЗ",     "orders": 200, "fromStart": 46.9, "fromPrev": 69.4 },
    { "key": "completed",  "label": "Выполнен (выкуп)",    "orders": 170, "fromStart": 39.9, "fromPrev": 85 }
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
