package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"fintrack/api/internal/httpx"
	"fintrack/api/internal/repository"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FundBuckets handles /api/fund-buckets routes (session cookie).
type FundBuckets struct {
	DB        *pgxpool.Pool
	JWTSecret []byte
}

// RegisterFundBuckets mounts /api/fund-buckets routes.
func (h *FundBuckets) RegisterFundBuckets(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/fund-buckets", h.list)
	mux.HandleFunc("POST /api/fund-buckets", h.create)
	mux.HandleFunc("POST /api/fund-buckets/{bucketId}/allocate", h.allocate)
	mux.HandleFunc("POST /api/fund-buckets/{bucketId}/unlock", h.unlock)
	mux.HandleFunc("PATCH /api/fund-buckets/{bucketId}/priority", h.setPriority)
}

func (h *FundBuckets) list(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	ctx := r.Context()
	rows, err := repository.ListFundBuckets(ctx, h.DB, sess.Sub)
	if err != nil {
		slog.Error("list fund buckets", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

type createFundBucketBody struct {
	Name          string  `json:"name"`
	TargetAmount  float64 `json:"targetAmount"`
	BankAccountID string  `json:"bankAccountId"`
	Priority      string  `json:"priority"`
}

func parseFundBucketPriority(v string) (string, bool) {
	switch v {
	case "high", "medium", "low":
		return v, true
	default:
		if v == "" {
			return "medium", true
		}
		return "", false
	}
}

func (h *FundBuckets) create(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	var body createFundBucketBody
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Fund bucket name is required"})
		return
	}
	if !isFinite(body.TargetAmount) || body.TargetAmount <= 0 {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "targetAmount must be a number greater than 0"})
		return
	}
	bankID := strings.TrimSpace(body.BankAccountID)
	if bankID == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bankAccountId is required"})
		return
	}
	priority, ok := parseFundBucketPriority(strings.TrimSpace(body.Priority))
	if !ok {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "priority must be high, medium, or low"})
		return
	}

	ctx := r.Context()
	row, err := repository.CreateFundBucket(ctx, h.DB, repository.CreateFundBucketInput{
		UserID:        sess.Sub,
		Name:          name,
		TargetAmount:  body.TargetAmount,
		BankAccountID: bankID,
		Priority:      priority,
	})
	if err != nil {
		if errors.Is(err, repository.ErrBankAccountNotFound) {
			httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			httpx.WriteJSON(w, http.StatusConflict, map[string]string{"error": "A fund bucket with this name already exists for this bank account"})
			return
		}
		slog.Error("create fund bucket", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"row": row})
}

type allocateBody struct {
	Amount float64 `json:"amount"`
}

func (h *FundBuckets) allocate(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	bucketID := strings.TrimSpace(r.PathValue("bucketId"))
	if bucketID == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bucket id required"})
		return
	}
	var body allocateBody
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	if !isFinite(body.Amount) || body.Amount <= 0 {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "amount must be a number greater than 0"})
		return
	}
	ctx := r.Context()
	row, err := repository.AllocateFundsToBucket(ctx, h.DB, sess.Sub, bucketID, body.Amount)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrCannotAllocateUnlocked), errors.Is(err, repository.ErrInsufficientAllocation):
			httpx.WriteJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		case errors.Is(err, repository.ErrBankAccountNotFound):
			httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		default:
			slog.Error("allocate fund bucket", "error", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
			return
		}
	}
	if row == nil {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"row": row})
}

func (h *FundBuckets) unlock(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	bucketID := strings.TrimSpace(r.PathValue("bucketId"))
	if bucketID == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bucket id required"})
		return
	}
	ctx := r.Context()
	row, err := repository.UnlockFundBucket(ctx, h.DB, sess.Sub, bucketID)
	if err != nil {
		slog.Error("unlock fund bucket", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if row == nil {
		httpx.WriteJSON(w, http.StatusConflict, map[string]string{
			"error": "Fund bucket not found, already unlocked, or target not met",
		})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"row": row})
}

type priorityBody struct {
	Priority string `json:"priority"`
}

func (h *FundBuckets) setPriority(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	bucketID := strings.TrimSpace(r.PathValue("bucketId"))
	if bucketID == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bucket id required"})
		return
	}
	var body priorityBody
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	priority, ok := parseFundBucketPriority(strings.TrimSpace(body.Priority))
	if !ok {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "priority must be high, medium, or low"})
		return
	}
	ctx := r.Context()
	row, err := repository.SetFundBucketPriority(ctx, h.DB, sess.Sub, bucketID, priority)
	if err != nil {
		slog.Error("fund bucket priority", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if row == nil {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"row": row})
}
