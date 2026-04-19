package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fintrack/api/internal/handler"
	"fintrack/api/internal/migrate"
	"fintrack/api/internal/testutil"

	"github.com/jackc/pgx/v5/pgxpool"
)

const testJWTSecret = "test-jwt-secret-for-integration-tests-32"

func poolWithMigrations(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	pool, cleanup := testutil.NewPostgresPool(t)
	ctx := context.Background()
	if err := migrate.Run(ctx, pool, testutil.MigrationsDir(t)); err != nil {
		cleanup()
		t.Fatalf("migrate: %v", err)
	}
	return pool, cleanup
}

// harness is an httptest server backed by a migrated Postgres pool and an HTTP client with a cookie jar.
type harness struct {
	BaseURL string
	Client  *http.Client
	Pool    *pgxpool.Pool
}

// newHarnessBare returns a test server against an empty migrated database (no bootstrap user).
func newHarnessBare(t *testing.T) *harness {
	t.Helper()
	pool, cleanup := poolWithMigrations(t)
	mux := handler.NewMux(handler.Deps{
		DB:           pool,
		JWTSecret:    []byte(testJWTSecret),
		CookieSecure: false,
	})
	srv := httptest.NewServer(mux)
	jar, err := cookiejar.New(nil)
	if err != nil {
		srv.Close()
		cleanup()
		t.Fatal(err)
	}
	h := &harness{
		BaseURL: srv.URL,
		Client:  &http.Client{Jar: jar},
		Pool:    pool,
	}
	t.Cleanup(func() {
		srv.Close()
		cleanup()
	})
	return h
}

func ensureIntegrationBootstrapUser(t *testing.T, h *harness) {
	t.Helper()
	ctx := context.Background()
	var n int64
	if err := h.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n > 0 {
		return
	}
	email := fmt.Sprintf("it-bootstrap-%d@example.com", time.Now().UnixNano())
	body := fmt.Sprintf(
		`{"email":%q,"password":"password123","name":"IT Admin","preferredCurrency":"USD"}`,
		email,
	)
	res, err := h.postJSON("/api/auth/bootstrap", body)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

// newHarness returns a server with at least one user so signup/login tests can run against an empty install policy.
func newHarness(t *testing.T) *harness {
	t.Helper()
	h := newHarnessBare(t)
	ensureIntegrationBootstrapUser(t, h)
	return h
}

func uniqueEmail() string {
	return fmt.Sprintf("it-%d@example.com", time.Now().UnixNano())
}

func mustStatus(t *testing.T, res *http.Response, want int) []byte {
	t.Helper()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != want {
		t.Fatalf("status %d want %d body: %s", res.StatusCode, want, b)
	}
	return b
}

func (h *harness) postJSON(path, body string) (*http.Response, error) {
	return h.Client.Post(h.BaseURL+path, "application/json", strings.NewReader(body))
}

func (h *harness) get(path string) (*http.Response, error) {
	return h.Client.Get(h.BaseURL + path)
}

func (h *harness) patchJSON(path, body string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPatch, h.BaseURL+path, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return h.Client.Do(req)
}

func (h *harness) delete(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, h.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return h.Client.Do(req)
}

func approveUser(t *testing.T, pool *pgxpool.Pool, email string) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `UPDATE users SET is_approved = true WHERE lower(email) = lower($1)`, email)
	if err != nil {
		t.Fatal(err)
	}
}

// signupAndApprove registers a new user (unapproved) and returns email + password.
func signupAndApprove(t *testing.T, h *harness, email, password string) {
	t.Helper()
	body := fmt.Sprintf(`{"email":%q,"password":%q,"name":"Test User","preferredCurrency":"USD"}`, email, password)
	res, err := h.postJSON("/api/auth/signup", body)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
	approveUser(t, h.Pool, email)
}

func login(t *testing.T, h *harness, email, password string) {
	t.Helper()
	body := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	res, err := h.postJSON("/api/auth/login", body)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

// signupApproveLogin creates a user, approves them, and establishes a session cookie.
func signupApproveLogin(t *testing.T, h *harness) (email, password string) {
	t.Helper()
	email = uniqueEmail()
	password = "password123"
	signupAndApprove(t, h, email, password)
	login(t, h, email, password)
	return email, password
}
