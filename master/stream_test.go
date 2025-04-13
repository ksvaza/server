package master

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTesting(t *testing.T) {
	t.Log("Test is running")
	text := `{"PSU":{"Uop":3124,"Iop":327,"Pop":8699,"Uip":6129,"Wh":15356},"GPS":{},"ACCEL":{}}`

	var payload payloadAll
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Errorf("Error: %s", err.Error())
	}

	out, err := json.Marshal(payload)
	if err != nil {
		t.Errorf("Error: %s", err.Error())
	}

	assert.Equal(t, text, string(out))
}
