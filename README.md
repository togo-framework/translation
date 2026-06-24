<!-- togo-header -->
<div align="center">
  <img src=".github/assets/togo-mark.svg" alt="togo" height="64" />
  <h1>togo-framework/translation</h1>
  <p>
    <a href="https://to-go.dev/marketplace"><img src="https://img.shields.io/badge/marketplace-to--go.dev-1FC7DC" alt="marketplace" /></a>
    <a href="https://pkg.go.dev/github.com/togo-framework/translation"><img src="https://pkg.go.dev/badge/github.com/togo-framework/translation.svg" alt="pkg.go.dev" /></a>
    <img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT" />
  </p>
  <p><strong>Part of the <a href="https://to-go.dev">togo</a> framework.</strong></p>
</div>

## Install

```bash
togo install togo-framework/translation
```

<!-- /togo-header -->

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

<!-- togo-sponsors -->
---

<div align="center">
  <h3>Premium sponsors</h3>
  <p>
    <a href="https://id8media.com"><strong>ID8 Media</strong></a> &nbsp;·&nbsp;
    <a href="https://one-studio.co"><strong>One Studio</strong></a>
  </p>
  <p><sub>Support togo — <a href="https://github.com/sponsors/fadymondy">become a sponsor</a>.</sub></p>
</div>
<!-- /togo-sponsors -->
