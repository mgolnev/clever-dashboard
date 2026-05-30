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

## `GET /api/cities` · `GET /api/regions`

Списки городов и областей/регионов для фильтров, отсортированы по убыванию числа
заказов. Используются вкладками «Обзор» и «Воронки».

```json
[
  { "name": "Киров", "orders": 114 },
  { "name": "Екатеринбург", "orders": 81 }
]
```

## `GET /api/metrics?start=YYYY-MM-DD&end=YYYY-MM-DD&city=<город>&region=<область>`

Метрики за период и за предыдущий период той же длины. Если `start`/`end` не
заданы — последние 7 дней данных. Необязательные `city` и `region` фильтруют
заказы по точному совпадению (пустые — без фильтра) и **комбинируются через AND**;
фильтр применяется и к текущему, и к предыдущему периоду.

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

## `GET /api/funnel?start=YYYY-MM-DD&end=YYYY-MM-DD&city=<город>&region=<область>`

Воронка пути заказа за период. Пустые даты — последняя неделя данных.
Необязательные `city` и `region` фильтруют все стадии, разрезы и топы
(комбинируются через AND).

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
