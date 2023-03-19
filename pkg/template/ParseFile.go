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
	"time"
)

func ParseFile(name string, path string) (Template, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading template from %q: %w", path, err)
	}
	funcMap := template.FuncMap{
		"sumIntegers": func(x, y int) int {
			return x + y
		},
		"formatTime": func(t time.Time, f string) string {
			return t.Format(f)
		},
	}
	t, err := template.New(name).Funcs(funcMap).Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("error parsing template from %q: %w", path, err)
	}
	return t, nil
}
