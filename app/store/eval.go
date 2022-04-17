package store

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/antonmedv/expr"
	"github.com/cappuccinotm/dastracker/app/errs"
	"github.com/cappuccinotm/dastracker/lib"
)

// Eval evaluates a predicate.
func (i If) Eval(upd Update) (bool, error) {
	env := map[string]interface{}{"Update": upd}
	copyFuncs(env)

	v, err := expr.Eval(i.Condition, env)
	if err != nil {
		return false, fmt.Errorf("evaluate expression: %w", err)
	}
	vv, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("expression %q, got type %T: %w", i.Condition, v, errs.ErrIfNotBool)
	}
	return vv, nil
}

// Evaluate evaluates the final values of each variable.
func Evaluate(v lib.Vars, data EvalData) (lib.Vars, error) {
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
		if err = tmpl.Execute(buf, data); err != nil {
			return lib.Vars{}, fmt.Errorf("evaluate the value of the %q variable: %w", key, err)
		}

		res[key] = buf.String()
	}

	return res, nil
}

// EvalData combines all the possible data to evaluate the template/expression..
type EvalData struct {
	Ticket Ticket
	Update Update
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
	"string_contains": strings.Contains,
}

func copyFuncs(f map[string]interface{}) {
	for k, v := range funcs {
		f[k] = v
	}
}
