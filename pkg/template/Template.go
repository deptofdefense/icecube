// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package template

import (
	"io"
)

type Template interface {
	Execute(w io.Writer, data any) error
}
