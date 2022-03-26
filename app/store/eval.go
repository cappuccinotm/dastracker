package store

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/cappuccinotm/dastracker/lib"
)

type evTmpl struct{ Update Update }

// Evaluate evaluates the final values of each variable.
func Evaluate(v lib.Vars, upd Update) (lib.Vars, error) {
	if len(v) == 0 {
		return nil, nil
	}

	res := lib.Vars(map[string]string{})
	for key, vv := range v {
		tmpl, err := template.New("").Funcs(funcs).Parse(vv)
		if err != nil {
			return lib.Vars{}, fmt.Errorf("parse %q variable: %w", key, err)
		}

		buf := &bytes.Buffer{}
		if err = tmpl.Execute(buf, evTmpl{Update: upd}); err != nil {
			return lib.Vars{}, fmt.Errorf("evaluate the value of the %q variable: %w", key, err)
		}

		res[key] = buf.String()
	}

	return res, nil
}

// map of functions to parse from the config file
var funcs = map[string]interface{}{
	"env": os.Getenv,
	"keys": func(s map[string]string) []string {
		res := make([]string, 0, len(s))
		for k := range s {
			res = append(res, k)
		}
		return res
	},
	"values": func(s map[string]string) []string {
		res := make([]string, 0, len(s))
		for _, v := range s {
			res = append(res, v)
		}
		return res
	},
	"seq": func(s []string) string {
		return strings.Join(s, ",")
	},
}
