package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"fintrack/api/internal/auth"
	"fintrack/api/internal/repository"
)

func TestAPI_GET_health(t *testing.T) {
	h := newHarness(t)
	res, err := h.get("/health")
	if err != nil {
		t.Fatal(err)
	}
	b := mustStatus(t, res, http.StatusOK)
	if string(b) != `{"status":"ok"}` {
		t.Fatalf("body %s", b)
	}
}

func TestAPI_POST_auth_signup(t *testing.T) {
	h := newHarness(t)
	email := uniqueEmail()
	body := fmt.Sprintf(`{"email":%q,"password":"password123","name":"Test User","preferredCurrency":"USD"}`, email)
	res, err := h.postJSON("/api/auth/signup", body)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_POST_auth_signup_duplicateConflict(t *testing.T) {
	h := newHarness(t)
	email := uniqueEmail()
	body := fmt.Sprintf(`{"email":%q,"password":"password123","name":"Test User","preferredCurrency":"USD"}`, email)
	res, err := h.postJSON("/api/auth/signup", body)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
	res, err = h.postJSON("/api/auth/signup", body)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusConflict)
}

func TestAPI_POST_auth_login_unapprovedForbidden(t *testing.T) {
	h := newHarness(t)
	email := uniqueEmail()
	body := fmt.Sprintf(`{"email":%q,"password":"password123","name":"Test User","preferredCurrency":"USD"}`, email)
	res, err := h.postJSON("/api/auth/signup", body)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)

	res, err = h.postJSON("/api/auth/login", fmt.Sprintf(`{"email":%q,"password":"password123"}`, email))
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusForbidden)
}

func TestAPI_POST_auth_login_success(t *testing.T) {
	h := newHarness(t)
	email, password := uniqueEmail(), "password123"
	signupAndApprove(t, h, email, password)
	res, err := h.postJSON("/api/auth/login", fmt.Sprintf(`{"email":%q,"password":%q}`, email, password))
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_POST_auth_forgotPassword(t *testing.T) {
	h := newHarness(t)
	email := uniqueEmail()
	body := fmt.Sprintf(`{"email":%q,"password":"password123","name":"Test User","preferredCurrency":"USD"}`, email)
	res, err := h.postJSON("/api/auth/signup", body)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)

	res, err = h.postJSON("/api/auth/forgot-password", fmt.Sprintf(`{"email":%q}`, email))
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_GET_auth_me(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.get("/api/auth/me")
	if err != nil {
		t.Fatal(err)
	}
	b := mustStatus(t, res, http.StatusOK)
	var wrap struct {
		User map[string]any `json:"user"`
	}
	if err := json.Unmarshal(b, &wrap); err != nil {
		t.Fatal(err)
	}
	if wrap.User["email"] == nil {
		t.Fatal("expected user.email")
	}
}

func TestAPI_PATCH_auth_me(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.patchJSON("/api/auth/me", `{"name":"Updated Name","preferredCurrency":"INR"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_POST_auth_changePassword_requestOtp(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/auth/change-password/request-otp", "")
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_POST_auth_changePassword(t *testing.T) {
	h := newHarness(t)
	email, _ := signupApproveLogin(t, h)
	ctx := context.Background()
	var userID string
	if err := h.Pool.QueryRow(ctx, `SELECT id FROM users WHERE lower(email) = lower($1)`, email).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	res, err := h.postJSON("/api/auth/change-password/request-otp", "")
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)

	ticket, err := auth.IssueOTPTicket([]byte(testJWTSecret), userID, auth.PurposePasswordChange, nil)
	if err != nil {
		t.Fatal(err)
	}
	cpBody := fmt.Sprintf(`{"newPassword":"anotherpassword999","otp":%q,"otpToken":%q}`, ticket.OTP, ticket.OtpToken)
	res, err = h.postJSON("/api/auth/change-password", cpBody)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)

	res, err = h.postJSON("/api/auth/login", fmt.Sprintf(`{"email":%q,"password":"anotherpassword999"}`, email))
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_POST_auth_logout(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/auth/logout", "")
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)

	res, err = h.get("/api/auth/me")
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusUnauthorized)
}

func TestAPI_POST_auth_resetPassword(t *testing.T) {
	h := newHarness(t)
	email, _ := signupApproveLogin(t, h)
	ctx := context.Background()
	var userID string
	if err := h.Pool.QueryRow(ctx, `SELECT id FROM users WHERE lower(email) = lower($1)`, email).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	em := auth.NormalizeEmail(email)
	ticket, err := auth.IssueOTPTicket([]byte(testJWTSecret), userID, auth.PurposePasswordReset, &em)
	if err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"email":%q,"otp":%q,"newPassword":"newpassword999","otpToken":%q}`,
		email, ticket.OTP, ticket.OtpToken)
	res, err := h.postJSON("/api/auth/reset-password", body)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)

	res, err = h.postJSON("/api/auth/login", fmt.Sprintf(`{"email":%q,"password":"newpassword999"}`, email))
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_GET_expense_categories_list_empty(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.get("/api/expense-categories")
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_POST_expense_categories_create(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	body := `{"name":"Groceries","description":"d","iconUrl":"/icons/g.png","color":"green"}`
	res, err := h.postJSON("/api/expense-categories", body)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)
}

func TestAPI_GET_expense_categories_byID(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/expense-categories", `{"name":"Groceries","description":"d","iconUrl":"/icons/g.png","color":"green"}`)
	if err != nil {
		t.Fatal(err)
	}
	b := mustStatus(t, res, http.StatusCreated)
	var out struct {
		Row struct {
			ID string `json:"id"`
		} `json:"row"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}

	res, err = h.get("/api/expense-categories/" + out.Row.ID)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_PATCH_expense_categories_byID(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/expense-categories", `{"name":"Groceries","description":"d","iconUrl":"/icons/g.png","color":"green"}`)
	if err != nil {
		t.Fatal(h)
	}
	b := mustStatus(t, res, http.StatusCreated)
	var out struct {
		Row struct {
			ID string `json:"id"`
		} `json:"row"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}

	res, err = h.patchJSON("/api/expense-categories/"+out.Row.ID, `{"description":"updated"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_POST_bank_accounts_create(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/expense-categories", `{"name":"Groceries","description":"d","iconUrl":"/icons/g.png","color":"green"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)

	body := `{"name":"Main","description":"","initialBalance":10000,"accountType":"savings","preferredCategories":["Groceries"]}`
	res, err = h.postJSON("/api/bank-accounts", body)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)
}

func TestAPI_GET_bank_accounts_list(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/expense-categories", `{"name":"Groceries","description":"d","iconUrl":"/icons/g.png","color":"green"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)
	res, err = h.postJSON("/api/bank-accounts", `{"name":"Main","description":"","initialBalance":10000,"accountType":"savings","preferredCategories":["Groceries"]}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)

	res, err = h.get("/api/bank-accounts")
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_GET_bank_accounts_byID(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/expense-categories", `{"name":"Groceries","description":"d","iconUrl":"/icons/g.png","color":"green"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)
	res, err = h.postJSON("/api/bank-accounts", `{"name":"Main","description":"","initialBalance":10000,"accountType":"savings","preferredCategories":["Groceries"]}`)
	if err != nil {
		t.Fatal(err)
	}
	b := mustStatus(t, res, http.StatusCreated)
	var out struct {
		Row struct {
			ID string `json:"id"`
		} `json:"row"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}

	res, err = h.get("/api/bank-accounts/" + out.Row.ID)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_PATCH_bank_accounts_byID(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/expense-categories", `{"name":"Groceries","description":"d","iconUrl":"/icons/g.png","color":"green"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)
	res, err = h.postJSON("/api/bank-accounts", `{"name":"Main","description":"","initialBalance":10000,"accountType":"savings","preferredCategories":["Groceries"]}`)
	if err != nil {
		t.Fatal(err)
	}
	b := mustStatus(t, res, http.StatusCreated)
	var out struct {
		Row struct {
			ID string `json:"id"`
		} `json:"row"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}

	res, err = h.patchJSON("/api/bank-accounts/"+out.Row.ID, `{"name":"Main Checking"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_POST_credit_cards_create(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/expense-categories", `{"name":"Groceries","description":"d","iconUrl":"/icons/g.png","color":"green"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)

	body := `{"name":"Visa","description":"","maxBalance":5000,"usedBalance":0,"lockedBalance":0,"preferredCategories":["Groceries"],"billGenerationDay":5,"billDueDay":15}`
	res, err = h.postJSON("/api/credit-cards", body)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)
}

func TestAPI_GET_credit_cards_list(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/expense-categories", `{"name":"Groceries","description":"d","iconUrl":"/icons/g.png","color":"green"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)
	res, err = h.postJSON("/api/credit-cards", `{"name":"Visa","description":"","maxBalance":5000,"usedBalance":0,"lockedBalance":0,"preferredCategories":["Groceries"],"billGenerationDay":5,"billDueDay":15}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)

	res, err = h.get("/api/credit-cards")
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_GET_credit_cards_byID(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/expense-categories", `{"name":"Groceries","description":"d","iconUrl":"/icons/g.png","color":"green"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)
	res, err = h.postJSON("/api/credit-cards", `{"name":"Visa","description":"","maxBalance":5000,"usedBalance":0,"lockedBalance":0,"preferredCategories":["Groceries"],"billGenerationDay":5,"billDueDay":15}`)
	if err != nil {
		t.Fatal(err)
	}
	b := mustStatus(t, res, http.StatusCreated)
	var out struct {
		Row struct {
			ID string `json:"id"`
		} `json:"row"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}

	res, err = h.get("/api/credit-cards/" + out.Row.ID)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_DELETE_credit_cards_byID(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/expense-categories", `{"name":"Groceries","description":"d","iconUrl":"/icons/g.png","color":"green"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)
	res, err = h.postJSON("/api/credit-cards", `{"name":"Visa","description":"","maxBalance":5000,"usedBalance":0,"lockedBalance":0,"preferredCategories":["Groceries"],"billGenerationDay":5,"billDueDay":15}`)
	if err != nil {
		t.Fatal(err)
	}
	b := mustStatus(t, res, http.StatusCreated)
	var out struct {
		Row struct {
			ID string `json:"id"`
		} `json:"row"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}

	res, err = h.delete("/api/credit-cards/" + out.Row.ID)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_POST_fund_buckets_create(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/expense-categories", `{"name":"Groceries","description":"d","iconUrl":"/icons/g.png","color":"green"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)
	res, err = h.postJSON("/api/bank-accounts", `{"name":"Main","description":"","initialBalance":10000,"accountType":"savings","preferredCategories":["Groceries"]}`)
	if err != nil {
		t.Fatal(err)
	}
	b := mustStatus(t, res, http.StatusCreated)
	var ba struct {
		Row struct {
			ID string `json:"id"`
		} `json:"row"`
	}
	if err := json.Unmarshal(b, &ba); err != nil {
		t.Fatal(err)
	}

	body := fmt.Sprintf(`{"name":"Vacation","targetAmount":50,"bankAccountId":%q,"priority":"high"}`, ba.Row.ID)
	res, err = h.postJSON("/api/fund-buckets", body)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)
}

func TestAPI_GET_fund_buckets_list(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/expense-categories", `{"name":"Groceries","description":"d","iconUrl":"/icons/g.png","color":"green"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)
	res, err = h.postJSON("/api/bank-accounts", `{"name":"Main","description":"","initialBalance":10000,"accountType":"savings","preferredCategories":["Groceries"]}`)
	if err != nil {
		t.Fatal(err)
	}
	b := mustStatus(t, res, http.StatusCreated)
	var ba struct {
		Row struct {
			ID string `json:"id"`
		} `json:"row"`
	}
	if err := json.Unmarshal(b, &ba); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"name":"Vacation","targetAmount":50,"bankAccountId":%q,"priority":"high"}`, ba.Row.ID)
	res, err = h.postJSON("/api/fund-buckets", body)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)

	res, err = h.get("/api/fund-buckets")
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_POST_fund_buckets_allocate(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/expense-categories", `{"name":"Groceries","description":"d","iconUrl":"/icons/g.png","color":"green"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)
	res, err = h.postJSON("/api/bank-accounts", `{"name":"Main","description":"","initialBalance":10000,"accountType":"savings","preferredCategories":["Groceries"]}`)
	if err != nil {
		t.Fatal(err)
	}
	b := mustStatus(t, res, http.StatusCreated)
	var ba struct {
		Row struct {
			ID string `json:"id"`
		} `json:"row"`
	}
	if err := json.Unmarshal(b, &ba); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"name":"Vacation","targetAmount":50,"bankAccountId":%q,"priority":"high"}`, ba.Row.ID)
	res, err = h.postJSON("/api/fund-buckets", body)
	if err != nil {
		t.Fatal(err)
	}
	b = mustStatus(t, res, http.StatusCreated)
	var fb struct {
		Row struct {
			ID string `json:"id"`
		} `json:"row"`
	}
	if err := json.Unmarshal(b, &fb); err != nil {
		t.Fatal(err)
	}

	res, err = h.postJSON("/api/fund-buckets/"+fb.Row.ID+"/allocate", `{"amount":50}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_POST_fund_buckets_unlock(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/expense-categories", `{"name":"Groceries","description":"d","iconUrl":"/icons/g.png","color":"green"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)
	res, err = h.postJSON("/api/bank-accounts", `{"name":"Main","description":"","initialBalance":10000,"accountType":"savings","preferredCategories":["Groceries"]}`)
	if err != nil {
		t.Fatal(err)
	}
	b := mustStatus(t, res, http.StatusCreated)
	var ba struct {
		Row struct {
			ID string `json:"id"`
		} `json:"row"`
	}
	if err := json.Unmarshal(b, &ba); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"name":"Vacation","targetAmount":50,"bankAccountId":%q,"priority":"high"}`, ba.Row.ID)
	res, err = h.postJSON("/api/fund-buckets", body)
	if err != nil {
		t.Fatal(err)
	}
	b = mustStatus(t, res, http.StatusCreated)
	var fb struct {
		Row struct {
			ID string `json:"id"`
		} `json:"row"`
	}
	if err := json.Unmarshal(b, &fb); err != nil {
		t.Fatal(err)
	}
	res, err = h.postJSON("/api/fund-buckets/"+fb.Row.ID+"/allocate", `{"amount":50}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)

	res, err = h.postJSON("/api/fund-buckets/"+fb.Row.ID+"/unlock", "")
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_PATCH_fund_buckets_priority(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.postJSON("/api/expense-categories", `{"name":"Groceries","description":"d","iconUrl":"/icons/g.png","color":"green"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusCreated)
	res, err = h.postJSON("/api/bank-accounts", `{"name":"Main","description":"","initialBalance":10000,"accountType":"savings","preferredCategories":["Groceries"]}`)
	if err != nil {
		t.Fatal(err)
	}
	b := mustStatus(t, res, http.StatusCreated)
	var ba struct {
		Row struct {
			ID string `json:"id"`
		} `json:"row"`
	}
	if err := json.Unmarshal(b, &ba); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"name":"Vacation","targetAmount":50,"bankAccountId":%q,"priority":"high"}`, ba.Row.ID)
	res, err = h.postJSON("/api/fund-buckets", body)
	if err != nil {
		t.Fatal(err)
	}
	b = mustStatus(t, res, http.StatusCreated)
	var fb struct {
		Row struct {
			ID string `json:"id"`
		} `json:"row"`
	}
	if err := json.Unmarshal(b, &fb); err != nil {
		t.Fatal(err)
	}

	res, err = h.patchJSON("/api/fund-buckets/"+fb.Row.ID+"/priority", `{"priority":"low"}`)
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestAPI_GET_auth_account_data(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.get("/api/auth/account-data")
	if err != nil {
		t.Fatal(err)
	}
	b := mustStatus(t, res, http.StatusOK)
	var export map[string]any
	if err := json.Unmarshal(b, &export); err != nil {
		t.Fatal(err)
	}
	if export["schemaVersion"].(float64) != 1 {
		t.Fatalf("schema: %v", export["schemaVersion"])
	}
}

func TestAPI_DELETE_auth_account_data(t *testing.T) {
	h := newHarness(t)
	signupApproveLogin(t, h)
	res, err := h.delete("/api/auth/account-data")
	if err != nil {
		t.Fatal(err)
	}
	mustStatus(t, res, http.StatusOK)
}

func TestRepository_ExportAccountPayload(t *testing.T) {
	pool, cleanup := poolWithMigrations(t)
	defer cleanup()
	ctx := context.Background()
	email := fmt.Sprintf("export-%d@example.com", time.Now().UnixNano())
	hash, err := auth.HashPassword("pw")
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CreateUserWithProfile(ctx, pool, email, hash, "E", "USD"); err != nil {
		t.Fatal(err)
	}
	var uid string
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&uid); err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `UPDATE users SET is_approved = true WHERE id = $1`, uid)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := repository.ExportAccountPayload(ctx, pool, uid)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"schemaVersion"`)) {
		t.Fatalf("unexpected export: %s", raw)
	}
}
