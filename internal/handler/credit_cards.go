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

// CreditCards handles /api/credit-cards routes (session cookie).
type CreditCards struct {
	DB        *pgxpool.Pool
	JWTSecret []byte
}

// RegisterCreditCards mounts /api/credit-cards routes.
func (h *CreditCards) RegisterCreditCards(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/credit-cards", h.list)
	mux.HandleFunc("POST /api/credit-cards", h.create)
	mux.HandleFunc("GET /api/credit-cards/{cardId}", h.getOne)
	mux.HandleFunc("PATCH /api/credit-cards/{cardId}", h.patch)
	mux.HandleFunc("DELETE /api/credit-cards/{cardId}", h.remove)
}

// @Summary List credit cards
// @Tags credit-cards
// @Produce json
// @Security SessionCookie
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/credit-cards [get]
func (h *CreditCards) list(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	ctx := r.Context()
	rows, err := repository.ListCreditCards(ctx, h.DB, sess.Sub)
	if err != nil {
		slog.Error("list credit cards", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

// @Summary Get credit card
// @Tags credit-cards
// @Produce json
// @Security SessionCookie
// @Param cardId path string true "Credit card ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/credit-cards/{cardId} [get]
func (h *CreditCards) getOne(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	cardID := strings.TrimSpace(r.PathValue("cardId"))
	if cardID == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "card id required"})
		return
	}
	ctx := r.Context()
	row, err := repository.GetCreditCardByID(ctx, h.DB, sess.Sub, cardID)
	if err != nil {
		slog.Error("get credit card", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if row == nil {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"row": row})
}

type createCreditCardBody struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	MaxBalance          *float64 `json:"maxBalance"`
	UsedBalance         *float64 `json:"usedBalance"`
	LockedBalance       *float64 `json:"lockedBalance"`
	PreferredCategories []string `json:"preferredCategories"`
	BillGenerationDay   *float64 `json:"billGenerationDay"`
	BillDueDay          *float64 `json:"billDueDay"`
}

// @Summary Create credit card
// @Tags credit-cards
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param body body map[string]interface{} true "Card fields per API contract"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/credit-cards [post]
func (h *CreditCards) create(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	var body createCreditCardBody
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Credit card name is required"})
		return
	}
	if body.MaxBalance == nil || !isFinite(*body.MaxBalance) {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "maxBalance must be a valid number"})
		return
	}
	used := 0.0
	if body.UsedBalance != nil {
		if !isFinite(*body.UsedBalance) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "usedBalance must be a valid number"})
			return
		}
		used = *body.UsedBalance
	}
	locked := 0.0
	if body.LockedBalance != nil {
		if !isFinite(*body.LockedBalance) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "lockedBalance must be a valid number"})
			return
		}
		locked = *body.LockedBalance
	}
	if body.BillGenerationDay == nil || body.BillDueDay == nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "billGenerationDay and billDueDay must be integers between 1 and 31",
		})
		return
	}
	genD, ok1 := validBillDayFloatPtr(body.BillGenerationDay)
	dueD, ok2 := validBillDayFloatPtr(body.BillDueDay)
	if !ok1 || !ok2 {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "billGenerationDay and billDueDay must be integers between 1 and 31",
		})
		return
	}

	ctx := r.Context()
	row, err := repository.CreateCreditCard(ctx, h.DB, repository.CreateCreditCardInput{
		UserID:              sess.Sub,
		Name:                name,
		Description:         strings.TrimSpace(body.Description),
		MaxBalance:          *body.MaxBalance,
		UsedBalance:         used,
		LockedBalance:       locked,
		PreferredCategories: normalizeCategoryNamesSlice(body.PreferredCategories),
		BillGenerationDay:   genD,
		BillDueDay:          dueD,
	})
	if err != nil {
		var inv *repository.InvalidPreferredCategoriesError
		if errors.As(err, &inv) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": inv.Error()})
			return
		}
		slog.Error("create credit card", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if row == nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"row": row})
}

type patchCreditCardBody struct {
	Name                *string   `json:"name"`
	Description         *string   `json:"description"`
	MaxBalance          *float64  `json:"maxBalance"`
	UsedBalance         *float64  `json:"usedBalance"`
	LockedBalance       *float64  `json:"lockedBalance"`
	PreferredCategories *[]string `json:"preferredCategories"`
	BillGenerationDay   *float64  `json:"billGenerationDay"`
	BillDueDay          *float64  `json:"billDueDay"`
}

// @Summary Update credit card
// @Tags credit-cards
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param cardId path string true "Credit card ID"
// @Param body body map[string]interface{} true "Partial update"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/credit-cards/{cardId} [patch]
func (h *CreditCards) patch(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	cardID := strings.TrimSpace(r.PathValue("cardId"))
	if cardID == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "card id required"})
		return
	}
	var body patchCreditCardBody
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	if body.MaxBalance != nil && !isFinite(*body.MaxBalance) {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "maxBalance must be a valid number"})
		return
	}
	if body.UsedBalance != nil && !isFinite(*body.UsedBalance) {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "usedBalance must be a valid number"})
		return
	}
	if body.LockedBalance != nil && !isFinite(*body.LockedBalance) {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "lockedBalance must be a valid number"})
		return
	}

	var name, desc *string
	if body.Name != nil {
		v := strings.TrimSpace(*body.Name)
		name = &v
	}
	if body.Description != nil {
		v := strings.TrimSpace(*body.Description)
		desc = &v
	}

	var genD, dueD *int
	if body.BillGenerationDay != nil {
		i, ok := validBillDayFloatPtr(body.BillGenerationDay)
		if !ok {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{
				"error": "billGenerationDay and billDueDay must be integers between 1 and 31",
			})
			return
		}
		genD = &i
	}
	if body.BillDueDay != nil {
		i, ok := validBillDayFloatPtr(body.BillDueDay)
		if !ok {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{
				"error": "billGenerationDay and billDueDay must be integers between 1 and 31",
			})
			return
		}
		dueD = &i
	}

	in := repository.UpdateCreditCardInput{
		UserID:              sess.Sub,
		CardID:              cardID,
		Name:                name,
		Description:         desc,
		MaxBalance:          body.MaxBalance,
		UsedBalance:         body.UsedBalance,
		LockedBalance:       body.LockedBalance,
		BillGenerationDay:   genD,
		BillDueDay:          dueD,
		PreferredCategories: body.PreferredCategories,
	}
	ctx := r.Context()
	row, err := repository.UpdateCreditCard(ctx, h.DB, in)
	if err != nil {
		var inv *repository.InvalidPreferredCategoriesError
		if errors.As(err, &inv) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": inv.Error()})
			return
		}
		slog.Error("update credit card", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if row == nil {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"row": row})
}

// @Summary Delete credit card
// @Tags credit-cards
// @Produce json
// @Security SessionCookie
// @Param cardId path string true "Credit card ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/credit-cards/{cardId} [delete]
func (h *CreditCards) remove(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	cardID := strings.TrimSpace(r.PathValue("cardId"))
	if cardID == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "card id required"})
		return
	}
	ctx := r.Context()
	deleted, err := repository.DeleteCreditCard(ctx, h.DB, sess.Sub, cardID)
	if err != nil {
		slog.Error("delete credit card", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if !deleted {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func validBillDayFloatPtr(p *float64) (int, bool) {
	if p == nil {
		return 0, false
	}
	f := *p
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	i := int(f)
	if float64(i) != f {
		return 0, false
	}
	if i < 1 || i > 31 {
		return 0, false
	}
	return i, true
}

func normalizeCategoryNamesSlice(in []string) []string {
	var out []string
	seen := make(map[string]struct{})
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
