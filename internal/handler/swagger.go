package handler

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
)

// RegisterSwagger serves OpenAPI (Swagger) UI at /swagger/.
func RegisterSwagger(mux *http.ServeMux) {
	mux.Handle("/swagger/", httpSwagger.WrapHandler)
}
