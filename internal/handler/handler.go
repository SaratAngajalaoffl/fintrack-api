package handler

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps are shared dependencies for HTTP handlers.
type Deps struct {
	DB           *pgxpool.Pool
	JWTSecret    []byte
	CookieSecure bool
}

// NewMux registers application routes (Go 1.22+ route patterns).
func NewMux(deps Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", Health)

	if deps.DB != nil && len(deps.JWTSecret) > 0 {
		a := &Auth{
			DB:           deps.DB,
			JWTSecret:    deps.JWTSecret,
			CookieSecure: deps.CookieSecure,
		}
		a.RegisterAuth(mux)

		b := &BankAccounts{
			DB:        deps.DB,
			JWTSecret: deps.JWTSecret,
		}
		b.RegisterBankAccounts(mux)

		e := &ExpenseCategories{
			DB:        deps.DB,
			JWTSecret: deps.JWTSecret,
		}
		e.RegisterExpenseCategories(mux)

		c := &CreditCards{
			DB:        deps.DB,
			JWTSecret: deps.JWTSecret,
		}
		c.RegisterCreditCards(mux)

		f := &FundBuckets{
			DB:        deps.DB,
			JWTSecret: deps.JWTSecret,
		}
		f.RegisterFundBuckets(mux)
	}

	return mux
}
