package handler

import (
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strings"

	"fintrack/api/internal/httpx"
	"fintrack/api/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BankAccounts handles /api/bank-accounts routes (session cookie).
type BankAccounts struct {
	DB        *pgxpool.Pool
	JWTSecret []byte
}

// RegisterBankAccounts mounts /api/bank-accounts routes.
func (h *BankAccounts) RegisterBankAccounts(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/bank-accounts", h.list)
	mux.HandleFunc("POST /api/bank-accounts", h.create)
	mux.HandleFunc("GET /api/bank-accounts/{accountId}", h.getOne)
	mux.HandleFunc("PATCH /api/bank-accounts/{accountId}", h.patch)
	mux.HandleFunc("DELETE /api/bank-accounts/{accountId}", h.remove)
}

func (h *BankAccounts) list(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	ctx := r.Context()
	rows, err := repository.ListBankAccounts(ctx, h.DB, sess.Sub)
	if err != nil {
		slog.Error("list bank accounts", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

func (h *BankAccounts) getOne(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(r.PathValue("accountId"))
	if accountID == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "account id required"})
		return
	}
	ctx := r.Context()
	row, err := repository.GetBankAccountByID(ctx, h.DB, sess.Sub, accountID)
	if err != nil {
		slog.Error("get bank account", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if row == nil {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"row": row})
}

type createBankAccountBody struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	InitialBalance      float64  `json:"initialBalance"`
	AccountType         string   `json:"accountType"`
	PreferredCategories []string `json:"preferredCategories"`
}

func (h *BankAccounts) create(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	var body createBankAccountBody
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Bank account name is required"})
		return
	}
	at := body.AccountType
	if at != "savings" && at != "current" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "accountType must be savings or current"})
		return
	}
	if !isFinite(body.InitialBalance) {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "initialBalance must be a number"})
		return
	}

	ctx := r.Context()
	row, err := repository.CreateBankAccount(ctx, h.DB, repository.CreateBankAccountInput{
		UserID:              sess.Sub,
		Name:                name,
		Description:         strings.TrimSpace(body.Description),
		AccountType:         at,
		InitialBalance:      body.InitialBalance,
		PreferredCategories: body.PreferredCategories,
	})
	if err != nil {
		var inv *repository.InvalidPreferredCategoriesError
		if errors.As(err, &inv) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": inv.Error()})
			return
		}
		slog.Error("create bank account", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if row == nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"row": row})
}

type patchBankAccountBody struct {
	Name                 *string    `json:"name"`
	Description          *string    `json:"description"`
	AccountType          *string    `json:"accountType"`
	Balance              *float64   `json:"balance"`
	LastDebitAt          *string    `json:"lastDebitAt"`
	LastCreditAt         *string    `json:"lastCreditAt"`
	PreferredCategories  *[]string  `json:"preferredCategories"`
}

func (h *BankAccounts) patch(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(r.PathValue("accountId"))
	if accountID == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "account id required"})
		return
	}
	var body patchBankAccountBody
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	var accountType *string
	switch {
	case body.AccountType == nil:
		accountType = nil
	case *body.AccountType == "savings" || *body.AccountType == "current":
		accountType = body.AccountType
	default:
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "accountType must be savings or current"})
		return
	}

	if body.Balance != nil && !isFinite(*body.Balance) {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "balance must be a number"})
		return
	}

	var name *string
	if body.Name != nil {
		v := strings.TrimSpace(*body.Name)
		name = &v
	}
	var desc *string
	if body.Description != nil {
		v := strings.TrimSpace(*body.Description)
		desc = &v
	}

	in := repository.UpdateBankAccountInput{
		UserID:       sess.Sub,
		AccountID:    accountID,
		Name:         name,
		Description:  desc,
		AccountType:  accountType,
		Balance:      body.Balance,
		LastDebitAt:  body.LastDebitAt,
		LastCreditAt: body.LastCreditAt,
		PreferredCat: body.PreferredCategories,
	}

	ctx := r.Context()
	row, err := repository.UpdateBankAccount(ctx, h.DB, in)
	if err != nil {
		var inv *repository.InvalidPreferredCategoriesError
		if errors.As(err, &inv) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": inv.Error()})
			return
		}
		slog.Error("update bank account", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if row == nil {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"row": row})
}

func (h *BankAccounts) remove(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(r.PathValue("accountId"))
	if accountID == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "account id required"})
		return
	}
	ctx := r.Context()
	deleted, err := repository.DeleteBankAccount(ctx, h.DB, sess.Sub, accountID)
	if err != nil {
		slog.Error("delete bank account", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if !deleted {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func isFinite(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}
