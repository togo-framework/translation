# translation — DB-backed dynamic i18n for togo

A togo plugin that makes translations **editable at runtime, in the database** — no
redeploy. It overrides the kernel translator (`k.I18n`): keys resolve from the DB
first and **fall back to the static i18n catalog** (install `togo-framework/i18n` for
the static fallback).

```bash
togo install togo-framework/i18n          # static catalog (fallback)
togo install togo-framework/translation   # DB-backed dynamic overrides
```

## REST API

| Method | Path | Body |
|---|---|---|
| `GET`    | `/api/translations?locale=ar`     | — → `{key: value}` |
| `GET`    | `/api/translations/locales`       | — → `["ar","en"]` |
| `PUT`    | `/api/translations/{locale}/{key}`| `{"value":"..."}` |
| `DELETE` | `/api/translations/{locale}/{key}`| — |
| `POST`   | `/api/translations/import`        | `{locale:{key:value}}` |

## Go API

```go
t := translation.FromKernel(k)
_ = t.Set(ctx, "ar", "welcome", "مرحبا")
msg := k.I18n.T("ar", "welcome")   // DB value, else the static catalog
```

## Data model

`translations(locale text, tkey text, value text, PRIMARY KEY (locale, tkey))`,
created on boot. An in-memory cache backs `T()` (refreshed on writes).

MIT
