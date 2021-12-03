package store

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"os"
	"testing"
)

func TestVars_UnmarshalYAML(t *testing.T) {
	const y = "var1: val1\nvar2: val2"
	v := Vars{}
	err := yaml.Unmarshal([]byte(y), &v)
	assert.NoError(t, err)
	assert.Equal(t, Vars{vals: map[string]string{"var1": "val1", "var2": "val2"}}, v)
}

func TestVars_Has(t *testing.T) {
	v := &Vars{vals: map[string]string{"var1": "val1"}}
	assert.True(t, v.Has("var1"))
	assert.False(t, v.Has("var2"))
	assert.NotPanics(t, func() { assert.False(t, (&Vars{}).Has("blah")) })
}

func TestVars_Get(t *testing.T) {
	v := &Vars{vals: map[string]string{"var1": "val1"}}
	assert.Equal(t, "val1", v.Get("var1"))
	assert.Empty(t, v.Get("var2"))
	assert.NotPanics(t, func() { assert.Empty(t, (&Vars{}).Get("var1")) })
}

func TestVars_Set(t *testing.T) {
	v := Vars{}
	v.Set("var1", "val1")
	assert.Equal(t, Vars{vals: map[string]string{"var1": "val1"}}, v)
	assert.NotPanics(t, func() { (&Vars{}).Set("var1", "val1") })
}

func TestVars_List(t *testing.T) {
	v := Vars{vals: map[string]string{"list": "a,b,c,d"}}
	assert.Equal(t, []string{"a", "b", "c", "d"}, v.List("list"))
	assert.NotPanics(t, func() { (&Vars{}).List("var1") })
}

func TestVars_Evaluated(t *testing.T) {
	v := Vars{evaluated: true}
	assert.True(t, v.Evaluated())
	assert.False(t, Vars{}.Evaluated())
}

func TestVars_Equal(t *testing.T) {
	assert.True(t, Vars{vals: map[string]string{"var1": "val1", "var2": "val2"}}.
		Equal(Vars{vals: map[string]string{"var2": "val2", "var1": "val1"}}))
	assert.True(t, Vars{}.Equal(Vars{}))

	// evaluated state is not considered in Equal
	assert.True(t, Vars{evaluated: true}.Equal(Vars{evaluated: true}))
	assert.True(t, Vars{evaluated: true}.Equal(Vars{evaluated: false}))

	assert.False(t, Vars{vals: map[string]string{"var1": "val1"}}.
		Equal(Vars{vals: map[string]string{"var2": "val2", "var1": "val1"}}))
}

func TestVars_Evaluate(t *testing.T) {
	v := Vars{vals: map[string]string{
		"var1": `{{ env "TESTVAR" }}`,
		"var2": `{{ seq (keys .Update.Fields) }}`,
		"var3": `{{ seq (values .Update.Fields) }}`,
		"var4": `static text`,
	}}
	err := os.Setenv("TESTVAR", "blah")
	vs, err := v.Evaluate(Update{Content: Content{Fields: TicketFields{
		"f1": "f1v",
		"f2": "f2v",
	}}})
	assert.NoError(t, err)

	// checking field-by-field, as mustn't rely on order of walking over map
	assert.True(t, vs.evaluated)
	assert.Equal(t, "blah", vs.vals["var1"])
	assert.Contains(t, []string{"f1,f2", "f2,f1"}, vs.vals["var2"])
	assert.Contains(t, []string{"f1v,f2v", "f2v,f1v"}, vs.vals["var3"])
	assert.Equal(t, "static text", vs.vals["var4"])
}