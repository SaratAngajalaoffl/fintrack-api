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

// Catppuccin Mocha accent tokens allowed for expense category colors (see api/migrations/008_expense_category_colors_palette.sql).
var catppuccinMochaExpenseColors = map[string]struct{}{
	"rosewater": {}, "flamingo": {}, "pink": {}, "mauve": {}, "red": {}, "maroon": {},
	"peach": {}, "yellow": {}, "green": {}, "teal": {}, "sky": {}, "sapphire": {},
	"blue": {}, "lavender": {},
}

func isCatppuccinMochaExpenseColor(s string) bool {
	_, ok := catppuccinMochaExpenseColors[s]
	return ok
}

// ExpenseCategories handles /api/expense-categories routes (session cookie).
type ExpenseCategories struct {
	DB        *pgxpool.Pool
	JWTSecret []byte
}

// RegisterExpenseCategories mounts /api/expense-categories routes.
func (h *ExpenseCategories) RegisterExpenseCategories(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/expense-categories", h.list)
	mux.HandleFunc("POST /api/expense-categories", h.create)
	mux.HandleFunc("GET /api/expense-categories/{categoryId}", h.getOne)
	mux.HandleFunc("PATCH /api/expense-categories/{categoryId}", h.patch)
	mux.HandleFunc("DELETE /api/expense-categories/{categoryId}", h.remove)
}

// @Summary List expense categories
// @Tags expense-categories
// @Produce json
// @Security SessionCookie
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/expense-categories [get]
func (h *ExpenseCategories) list(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	ctx := r.Context()
	rows, err := repository.ListExpenseCategories(ctx, h.DB, sess.Sub)
	if err != nil {
		slog.Error("list expense categories", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

// @Summary Get expense category
// @Tags expense-categories
// @Produce json
// @Security SessionCookie
// @Param categoryId path string true "Category ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/expense-categories/{categoryId} [get]
func (h *ExpenseCategories) getOne(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	categoryID := strings.TrimSpace(r.PathValue("categoryId"))
	if categoryID == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "category id required"})
		return
	}
	ctx := r.Context()
	row, err := repository.GetExpenseCategoryByID(ctx, h.DB, sess.Sub, categoryID)
	if err != nil {
		slog.Error("get expense category", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if row == nil {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"row": row})
}

type createExpenseCategoryBody struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IconURL     string `json:"iconUrl"`
	Color       string `json:"color"`
}

// @Summary Create expense category
// @Tags expense-categories
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param body body map[string]interface{} true "name, color, …"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/expense-categories [post]
func (h *ExpenseCategories) create(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	var body createExpenseCategoryBody
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Category name is required"})
		return
	}
	iconURL := strings.TrimSpace(body.IconURL)
	if iconURL == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Category icon URL is required"})
		return
	}
	if !isCatppuccinMochaExpenseColor(body.Color) {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "color must be a valid Catppuccin Mocha color token"})
		return
	}

	ctx := r.Context()
	row, err := repository.CreateExpenseCategory(ctx, h.DB, repository.CreateExpenseCategoryInput{
		UserID:      sess.Sub,
		Name:        name,
		Description: strings.TrimSpace(body.Description),
		IconURL:     iconURL,
		Color:       body.Color,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			httpx.WriteJSON(w, http.StatusConflict, map[string]string{"error": "A category with this name already exists"})
			return
		}
		slog.Error("create expense category", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if row == nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"row": row})
}

type patchExpenseCategoryBody struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IconURL     *string `json:"iconUrl"`
	Color       *string `json:"color"`
}

// @Summary Update expense category
// @Tags expense-categories
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param categoryId path string true "Category ID"
// @Param body body map[string]interface{} true "Partial update"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/expense-categories/{categoryId} [patch]
func (h *ExpenseCategories) patch(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	categoryID := strings.TrimSpace(r.PathValue("categoryId"))
	if categoryID == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "category id required"})
		return
	}
	var body patchExpenseCategoryBody
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	if body.Color != nil && !isCatppuccinMochaExpenseColor(*body.Color) {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "color must be a valid Catppuccin Mocha color token"})
		return
	}

	var name, description, iconURL *string
	if body.Name != nil {
		v := strings.TrimSpace(*body.Name)
		name = &v
	}
	if body.Description != nil {
		v := strings.TrimSpace(*body.Description)
		description = &v
	}
	if body.IconURL != nil {
		v := strings.TrimSpace(*body.IconURL)
		iconURL = &v
	}

	in := repository.UpdateExpenseCategoryInput{
		UserID:      sess.Sub,
		CategoryID:  categoryID,
		Name:        name,
		Description: description,
		IconURL:     iconURL,
		Color:       body.Color,
	}
	ctx := r.Context()
	row, err := repository.UpdateExpenseCategory(ctx, h.DB, in)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			httpx.WriteJSON(w, http.StatusConflict, map[string]string{"error": "A category with this name already exists"})
			return
		}
		slog.Error("update expense category", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if row == nil {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"row": row})
}

// @Summary Delete expense category
// @Tags expense-categories
// @Produce json
// @Security SessionCookie
// @Param categoryId path string true "Category ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/expense-categories/{categoryId} [delete]
func (h *ExpenseCategories) remove(w http.ResponseWriter, r *http.Request) {
	sess, ok := requireSession(w, r, h.JWTSecret)
	if !ok {
		return
	}
	categoryID := strings.TrimSpace(r.PathValue("categoryId"))
	if categoryID == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "category id required"})
		return
	}
	ctx := r.Context()
	deleted, err := repository.DeleteExpenseCategory(ctx, h.DB, sess.Sub, categoryID)
	if err != nil {
		slog.Error("delete expense category", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if !deleted {
		httpx.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}
