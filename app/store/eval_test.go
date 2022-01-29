package store

import (
	"github.com/cappuccinotm/dastracker/lib"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestEvaluate(t *testing.T) {
	v := lib.Vars(map[string]string{
		"var1": `{{ env "TESTVAR" }}`,
		"var2": `{{ seq (keys .Update.Fields) }}`,
		"var3": `{{ seq (values .Update.Fields) }}`,
		"var4": `static text`,
	})
	err := os.Setenv("TESTVAR", "blah")
	vs, err := Evaluate(v, Update{Content: Content{Fields: TicketFields{
		"f1": "f1v",
		"f2": "f2v",
	}}})
	assert.NoError(t, err)

	// checking field-by-field, as mustn't rely on order of walking over map
	assert.Equal(t, "blah", vs["var1"])
	assert.Contains(t, []string{"f1,f2", "f2,f1"}, vs["var2"])
	assert.Contains(t, []string{"f1v,f2v", "f2v,f1v"}, vs["var3"])
	assert.Equal(t, "static text", vs["var4"])
}
