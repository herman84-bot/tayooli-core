package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractUpSQL(t *testing.T) {
	raw := `CREATE TABLE foo (id INT);
INSERT INTO foo VALUES (1);

-- DOWN
DROP TABLE foo;
`
	up := extractUpSQL(raw)
	assert.Contains(t, up, "CREATE TABLE foo")
	assert.Contains(t, up, "INSERT INTO foo")
	assert.NotContains(t, up, "DROP TABLE foo")
	assert.NotContains(t, up, "DOWN")

	noDown := `CREATE TABLE bar (id INT);`
	assert.Equal(t, noDown, strings.TrimSpace(extractUpSQL(noDown)))
}
