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
)

func ParseFile(name string, path string) (Template, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading template from %q: %w", path, err)
	}
	t, err := template.New(name).Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("error parsing template from %q: %w", path, err)
	}
	return t, nil
}
