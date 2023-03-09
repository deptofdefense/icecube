// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package server

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"
)

func ServeContent(w http.ResponseWriter, r *http.Request, p string, content io.ReadSeeker, modtime time.Time, download bool, errorHandler func(w http.ResponseWriter, r *http.Request, err error) error) {
	w.Header().Set("Cache-Control", "no-cache")
	if download {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(p)))
	}
	http.ServeContent(w, r, filepath.Base(p), modtime, content)
}
