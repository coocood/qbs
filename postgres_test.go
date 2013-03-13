package qbs

import (
	"github.com/coocood/assrt"
	"testing"
	"time"
)

func TestSqlTypeForPgDialect(t *testing.T) {
	assert := assrt.NewAssert(t)
	d := NewPostgres()
	assert.Equal("boolean", d.SqlType(true, 0))
	var indirect interface{} = true
	assert.Equal("boolean", d.SqlType(indirect, 0))
	assert.Equal("integer", d.SqlType(uint32(2), 0))
	assert.Equal("bigint", d.SqlType(int64(1), 0))
	assert.Equal("double precision", d.SqlType(1.8, 0))
	assert.Equal("bytea", d.SqlType([]byte("asdf"), 0))
	assert.Equal("text", d.SqlType("astring", 0))
	assert.Equal("varchar(255)", d.SqlType("a", 255))
	assert.Equal("varchar(128)", d.SqlType("b", 128))
	assert.Equal("timestamp with time zone", d.SqlType(time.Now(), 0))
}
