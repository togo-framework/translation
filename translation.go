// Package translation is a togo plugin: DB-backed dynamic i18n. It overrides the
// kernel translator (k.I18n) with one that resolves keys from the database first
// and falls back to the static i18n catalog. Translations are editable at runtime
// (no redeploy) via the Go API or the REST endpoints under /api/translations.
package translation

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/togo-framework/togo"
)

func init() {
	// Priority just after i18n (PriorityService) so we wrap the catalog it set.
	togo.RegisterProviderFunc("translation", togo.PriorityService+5, func(k *togo.Kernel) error {
		db, err := k.SQL(context.Background())
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(context.Background(),
			`CREATE TABLE IF NOT EXISTS translations (locale text NOT NULL, tkey text NOT NULL, value text NOT NULL, PRIMARY KEY (locale, tkey))`); err != nil {
			return err
		}
		s := &Store{db: db, fallback: k.I18n, cache: map[string]map[string]string{}}
		if err := s.reload(context.Background()); err != nil {
			return err
		}
		k.I18n = s // DB-backed translator with static fallback
		k.Set("translation", s)
		mount(k.Router, s)
		return nil
	})
}

// Store is the DB-backed translator + manager (kernel service "translation").
type Store struct {
	db       *sql.DB
	fallback togo.Translator
	mu       sync.RWMutex
	cache    map[string]map[string]string // locale -> key -> value
}

// FromKernel returns the translation store, or nil if not installed.
func FromKernel(k *togo.Kernel) *Store {
	if v, ok := k.Get("translation"); ok {
		if s, ok := v.(*Store); ok {
			return s
		}
	}
	return nil
}

// T implements togo.Translator: DB value if present, else the static catalog.
func (s *Store) T(locale, key string) string {
	s.mu.RLock()
	if m, ok := s.cache[locale]; ok {
		if v, ok := m[key]; ok {
			s.mu.RUnlock()
			return v
		}
	}
	s.mu.RUnlock()
	if s.fallback != nil {
		return s.fallback.T(locale, key)
	}
	return key
}

func (s *Store) reload(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT locale, tkey, value FROM translations`)
	if err != nil {
		return err
	}
	defer rows.Close()
	c := map[string]map[string]string{}
	for rows.Next() {
		var l, k, v string
		if err := rows.Scan(&l, &k, &v); err != nil {
			return err
		}
		if c[l] == nil {
			c[l] = map[string]string{}
		}
		c[l][k] = v
	}
	s.mu.Lock()
	s.cache = c
	s.mu.Unlock()
	return rows.Err()
}

// Set upserts a translation and updates the cache.
func (s *Store) Set(ctx context.Context, locale, key, value string) error {
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO translations (locale, tkey, value) VALUES (?, ?, ?) ON CONFLICT (locale, tkey) DO UPDATE SET value = excluded.value`,
		locale, key, value); err != nil {
		return err
	}
	s.mu.Lock()
	if s.cache[locale] == nil {
		s.cache[locale] = map[string]string{}
	}
	s.cache[locale][key] = value
	s.mu.Unlock()
	return nil
}

// Delete removes a translation.
func (s *Store) Delete(ctx context.Context, locale, key string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM translations WHERE locale=? AND tkey=?`, locale, key); err != nil {
		return err
	}
	s.mu.Lock()
	if m := s.cache[locale]; m != nil {
		delete(m, key)
	}
	s.mu.Unlock()
	return nil
}

// List returns all overrides for a locale.
func (s *Store) List(locale string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := map[string]string{}
	for k, v := range s.cache[locale] {
		out[k] = v
	}
	return out
}

// Locales returns the locales that have DB overrides.
func (s *Store) Locales() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []string{}
	for l := range s.cache {
		out = append(out, l)
	}
	return out
}

func mount(r chi.Router, s *Store) {
	r.Route("/api/translations", func(r chi.Router) {
		r.Get("/locales", func(w http.ResponseWriter, req *http.Request) { writeJSON(w, s.Locales()) })
		r.Get("/", func(w http.ResponseWriter, req *http.Request) { writeJSON(w, s.List(req.URL.Query().Get("locale"))) })
		r.Put("/{locale}/{key}", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				Value string `json:"value"`
			}
			_ = json.NewDecoder(req.Body).Decode(&body)
			writeErr(w, s.Set(req.Context(), chi.URLParam(req, "locale"), chi.URLParam(req, "key"), body.Value))
		})
		r.Delete("/{locale}/{key}", func(w http.ResponseWriter, req *http.Request) {
			writeErr(w, s.Delete(req.Context(), chi.URLParam(req, "locale"), chi.URLParam(req, "key")))
		})
		r.Post("/import", func(w http.ResponseWriter, req *http.Request) {
			var in map[string]map[string]string
			if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			for l, m := range in {
				for k, v := range m {
					if err := s.Set(req.Context(), l, k, v); err != nil {
						writeErr(w, err)
						return
					}
				}
			}
			writeErr(w, nil)
		})
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}
