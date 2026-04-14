package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"fintrack/api/internal/auth"
	"fintrack/api/internal/httpx"
	"fintrack/api/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Auth holds auth route dependencies.
type Auth struct {
	DB            *pgxpool.Pool
	JWTSecret     []byte
	CookieSecure  bool
}

func (a *Auth) session(r *http.Request) (*auth.SessionPayload, error) {
	raw := readSessionCookie(r)
	if raw == "" {
		return nil, errors.New("no session")
	}
	return auth.VerifySessionToken(a.JWTSecret, raw)
}

// RegisterAuth mounts /api/auth/* routes.
func (a *Auth) RegisterAuth(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/login", a.postLogin)
	mux.HandleFunc("POST /api/auth/signup", a.postSignup)
	mux.HandleFunc("POST /api/auth/logout", a.postLogout)
	mux.HandleFunc("GET /api/auth/me", a.getMe)
	mux.HandleFunc("PATCH /api/auth/me", a.patchMe)
	mux.HandleFunc("POST /api/auth/forgot-password", a.postForgotPassword)
	mux.HandleFunc("POST /api/auth/reset-password", a.postResetPassword)
	mux.HandleFunc("POST /api/auth/change-password/request-otp", a.postChangePasswordRequestOtp)
	mux.HandleFunc("POST /api/auth/change-password", a.postChangePassword)
	mux.HandleFunc("GET /api/auth/account-data", a.getAccountData)
	mux.HandleFunc("DELETE /api/auth/account-data", a.deleteAccountData)
}

func (a *Auth) postLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	if msg := auth.ValidateEmail(body.Email); msg != "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	if msg := auth.ValidatePassword(body.Password); msg != "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	email := auth.NormalizeEmail(body.Email)
	user, err := repository.FindUserByEmail(ctx, a.DB, email)
	if err != nil {
		slog.Error("login find user", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if user == nil || !auth.VerifyPassword(body.Password, user.PasswordHash) {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid email or password"})
		return
	}
	if !user.IsApproved {
		httpx.WriteJSON(w, http.StatusForbidden, map[string]string{
			"error": "Your account is not approved yet. Contact an administrator or wait for approval.",
		})
		return
	}
	tok, err := auth.SignSessionToken(a.JWTSecret, user.ID, auth.NormalizeEmail(user.Email))
	if err != nil {
		slog.Error("sign session", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	writeSessionCookie(w, tok, a.CookieSecure)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"user": map[string]string{
			"id":    user.ID,
			"email": auth.NormalizeEmail(user.Email),
		},
	})
}

func (a *Auth) postSignup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body struct {
		Email             string `json:"email"`
		Password          string `json:"password"`
		Name              string `json:"name"`
		PreferredCurrency string `json:"preferredCurrency"`
	}
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	if msg := auth.ValidateEmail(body.Email); msg != "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	if msg := auth.ValidatePassword(body.Password); msg != "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	if msg := auth.ValidateProfileName(body.Name); msg != "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	if msg := auth.ValidatePreferredCurrency(body.PreferredCurrency); msg != "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	email := auth.NormalizeEmail(body.Email)
	pw, err := auth.HashPassword(body.Password)
	if err != nil {
		slog.Error("hash signup", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	name := auth.NormalizeProfileName(body.Name)
	curr := auth.ParsePreferredCurrency(body.PreferredCurrency)
	err = repository.CreateUserWithProfile(ctx, a.DB, email, pw, name, curr)
	if err != nil {
		if repository.IsUniqueViolation(err) {
			httpx.WriteJSON(w, http.StatusConflict, map[string]string{"error": "An account with this email already exists"})
			return
		}
		slog.Error("signup", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"message": "Account created. An administrator must approve your account before you can sign in.",
	})
}

func (a *Auth) postLogout(w http.ResponseWriter, r *http.Request) {
	clearSessionCookie(w, a.CookieSecure)
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *Auth) getMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, err := a.session(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]any{"user": nil})
		return
	}
	user, err := repository.FindUserByID(ctx, a.DB, sess.Sub)
	if err != nil || user == nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]any{"user": nil})
		return
	}
	prof, err := repository.GetProfile(ctx, a.DB, user.ID)
	if err != nil {
		slog.Error("get profile", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	name := prof.Name
	if name == "" {
		name = "User"
	}
	pc := auth.ParsePreferredCurrency(prof.PreferredCurrency)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":                 user.ID,
			"email":              auth.NormalizeEmail(user.Email),
			"isApproved":         user.IsApproved,
			"name":               name,
			"preferredCurrency":  pc,
		},
	})
}

func (a *Auth) patchMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, err := a.session(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	var body struct {
		Name              *string `json:"name"`
		PreferredCurrency *string `json:"preferredCurrency"`
	}
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	hasName := body.Name != nil
	hasCurr := body.PreferredCurrency != nil
	if !hasName && !hasCurr {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Provide at least one profile field to update"})
		return
	}
	if hasName {
		if msg := auth.ValidateProfileName(*body.Name); msg != "" {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
			return
		}
	}
	if hasCurr {
		if msg := auth.ValidatePreferredCurrency(*body.PreferredCurrency); msg != "" {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
			return
		}
	}
	user, err := repository.FindUserByID(ctx, a.DB, sess.Sub)
	if err != nil || user == nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	normalizedEmail := auth.NormalizeEmail(user.Email)
	insertName := normalizedEmail
	if hasName {
		insertName = auth.NormalizeProfileName(*body.Name)
	}
	insertCurr := "USD"
	if hasCurr {
		insertCurr = auth.ParsePreferredCurrency(*body.PreferredCurrency)
	}
	var updName, updCurr *string
	if hasName {
		s := auth.NormalizeProfileName(*body.Name)
		updName = &s
	}
	if hasCurr {
		s := auth.ParsePreferredCurrency(*body.PreferredCurrency)
		updCurr = &s
	}
	if err := repository.UpsertProfile(ctx, a.DB, sess.Sub, normalizedEmail, insertName, insertCurr, updName, updCurr); err != nil {
		slog.Error("upsert profile", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	prof, err := repository.GetProfile(ctx, a.DB, user.ID)
	if err != nil {
		slog.Error("get profile after patch", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	displayName := prof.Name
	if displayName == "" {
		displayName = normalizedEmail
	}
	pc := auth.ParsePreferredCurrency(prof.PreferredCurrency)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"user": map[string]any{
			"id":                 user.ID,
			"email":              normalizedEmail,
			"isApproved":         user.IsApproved,
			"name":               displayName,
			"preferredCurrency":  pc,
		},
	})
}

func (a *Auth) postForgotPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body struct {
		Email string `json:"email"`
	}
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	if msg := auth.ValidateEmail(body.Email); msg != "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	user, err := repository.FindUserByEmail(ctx, a.DB, auth.NormalizeEmail(body.Email))
	if err != nil {
		slog.Error("forgot find user", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if user == nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"message": "If an account exists for this email, further instructions apply.",
		})
		return
	}
	em := auth.NormalizeEmail(user.Email)
	ticket, err := auth.IssueOTPTicket(a.JWTSecret, user.ID, auth.PurposePasswordReset, &em)
	if err != nil {
		slog.Error("issue otp", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	slog.Info("password_reset OTP", "email", user.Email, "user_id", user.ID, "otp", ticket.OTP)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"otpToken":  ticket.OtpToken,
		"expiresAt": ticket.ExpiresAt.Format(time.RFC3339Nano),
		"message":   "If an account exists for this email, continue to set a new password.",
	})
}

func (a *Auth) postResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body struct {
		Email       string `json:"email"`
		OTP         string `json:"otp"`
		NewPassword string `json:"newPassword"`
		OtpToken    string `json:"otpToken"`
	}
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	if msg := auth.ValidateEmail(body.Email); msg != "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	if msg := auth.ValidateOTP(body.OTP); msg != "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	if msg := auth.ValidatePassword(body.NewPassword); msg != "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	if strings.TrimSpace(body.OtpToken) == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "otpToken is required"})
		return
	}
	user, err := repository.FindUserByEmail(ctx, a.DB, auth.NormalizeEmail(body.Email))
	if err != nil || user == nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid or expired OTP"})
		return
	}
	em := auth.NormalizeEmail(body.Email)
	sub, ok := auth.VerifyOTPTicket(a.JWTSecret, body.OtpToken, strings.TrimSpace(body.OTP), auth.PurposePasswordReset, &em)
	if !ok || sub != user.ID {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid or expired OTP"})
		return
	}
	hash, err := auth.HashPassword(body.NewPassword)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if err := repository.UpdatePassword(ctx, a.DB, user.ID, hash); err != nil {
		slog.Error("reset password", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "Password updated. You can sign in with your new password.",
		"email":   auth.NormalizeEmail(user.Email),
	})
}

func (a *Auth) postChangePasswordRequestOtp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, err := a.session(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	user, err := repository.FindUserByID(ctx, a.DB, sess.Sub)
	if err != nil || user == nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	ticket, err := auth.IssueOTPTicket(a.JWTSecret, user.ID, auth.PurposePasswordChange, nil)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	slog.Info("password_change OTP", "email", user.Email, "user_id", user.ID, "otp", ticket.OTP)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"otpToken":  ticket.OtpToken,
		"expiresAt": ticket.ExpiresAt.Format(time.RFC3339Nano),
		"message":   "Use the OTP from the server log (dev) with otpToken, expiresAt, and your new password to confirm.",
	})
}

func (a *Auth) postChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, err := a.session(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	var body struct {
		NewPassword string `json:"newPassword"`
		OTP         string `json:"otp"`
		OtpToken    string `json:"otpToken"`
	}
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	if msg := auth.ValidateOTP(body.OTP); msg != "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	if msg := auth.ValidatePassword(body.NewPassword); msg != "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	if strings.TrimSpace(body.OtpToken) == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "otpToken is required"})
		return
	}
	user, err := repository.FindUserByID(ctx, a.DB, sess.Sub)
	if err != nil || user == nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	sub, ok := auth.VerifyOTPTicket(a.JWTSecret, body.OtpToken, strings.TrimSpace(body.OTP), auth.PurposePasswordChange, nil)
	if !ok || sub != user.ID {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid or expired OTP"})
		return
	}
	hash, err := auth.HashPassword(body.NewPassword)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	if err := repository.UpdatePassword(ctx, a.DB, user.ID, hash); err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "Password changed."})
}

func (a *Auth) getAccountData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, err := a.session(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	user, err := repository.FindUserByID(ctx, a.DB, sess.Sub)
	if err != nil || user == nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	raw, err := repository.ExportAccountPayload(ctx, a.DB, sess.Sub)
	if err != nil {
		slog.Error("export account", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not export account data"})
		return
	}
	date := time.Now().UTC().Format("2006-01-02")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="fintrack-export-%s.json"`, date))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (a *Auth) deleteAccountData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, err := a.session(r)
	if err != nil {
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	if err := repository.DeleteUser(ctx, a.DB, sess.Sub); err != nil {
		slog.Error("delete user", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server error"})
		return
	}
	clearSessionCookie(w, a.CookieSecure)
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
