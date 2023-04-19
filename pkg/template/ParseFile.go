// =================================================================
//
// Work of the U.S. Department of Defense, Defense Digital Service.
// Released as open source under the MIT License.  See LICENSE file.
//
// =================================================================

package template

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"path"
	"strings"
	"time"
)

var funcMap = template.FuncMap{
	"concat": func(s ...string) string {
		return strings.Join(s, "")
	},
	"dir": func(p string) string {
		return path.Dir(p)
	},
	"sumIntegers": func(x, y int) int {
		return x + y
	},
	"formatTime": func(t time.Time, f string) string {
		return t.Format(f)
	},
	"joinPath": func(p ...string) string {
		return path.Join(p...)
	},
	"trimPrefix": func(p string, b string) string {
		return strings.TrimPrefix(p, b)
	},
}

func ParseFile(name string, p string) (Template, error) {
	data, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("error reading template from %q: %w", p, err)
	}
	t, err := template.New(name).Funcs(funcMap).Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("error parsing template from %q: %w", p, err)
	}
	return t, nil
}
