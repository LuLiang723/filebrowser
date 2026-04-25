package fbhttp

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
)

type archiveRequest struct {
	Items  []string `json:"items"`
	Format string   `json:"format"`
}

func withPermArchive(fn handleFunc) handleFunc {
	return withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
		if !d.user.Perm.Create {
			return http.StatusForbidden, nil
		}
		return fn(w, r, d)
	})
}

var archiveHandler = withPermArchive(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	var req archiveRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return http.StatusBadRequest, fmt.Errorf("failed to decode body: %w", err)
		}
		defer r.Body.Close()
	}

	dstPath := r.URL.Path
	dstPath = filepath.Clean(dstPath)

	return http.StatusOK, nil
})