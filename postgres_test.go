package qbs

import (
	"github.com/coocood/assrt"
	"testing"
	"time"
)

func TestSqlTypeForPgDialect(t *testing.T) {
	assert := assrt.NewAssert(t)
	d := NewPostgres()
	assert.Equal("boolean", d.sqlType(true, 0))
	var indirect interface{} = true
	assert.Equal("boolean", d.sqlType(indirect, 0))
	assert.Equal("integer", d.sqlType(uint32(2), 0))
	assert.Equal("bigint", d.sqlType(int64(1), 0))
	assert.Equal("double precision", d.sqlType(1.8, 0))
	assert.Equal("bytea", d.sqlType([]byte("asdf"), 0))
	assert.Equal("text", d.sqlType("astring", 0))
	assert.Equal("varchar(255)", d.sqlType("a", 255))
	assert.Equal("varchar(128)", d.sqlType("b", 128))
	assert.Equal("timestamp with time zone", d.sqlType(time.Now(), 0))
}
