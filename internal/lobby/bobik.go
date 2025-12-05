package lobby

import (
	"net/http"
	"path/filepath"

	"main/internal/data"
)

func NewBobikHandler(store *data.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join("web", "templates", "bobik.html"))
	}
}
