package server

import (
	"net/http"

	"go-project-template/requestid"
)

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := requestid.Resolve(r.Header.Get(requestid.HeaderName))
		ctx := requestid.WithContext(r.Context(), id)

		w.Header().Set(requestid.HeaderName, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
