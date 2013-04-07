package qbs

import (
  "github.com/coocood/assrt"
	"testing"
	"time"
)

func TestSqlTypeForOrDialect(t *testing.T) {
	assert := assrt.NewAssert(t)
	d := NewOracle()
	assert.Equal("integer", d.SqlType(uint32(2), 0))
	assert.Equal("bigint", d.SqlType(int64(1), 0))
	assert.Equal("double precision", d.SqlType(1.8, 0))
	assert.Equal("bytea", d.SqlType([]byte("asdf"), 0))
	assert.Equal("varchar2(255)", d.SqlType("a", 255))
	assert.Equal("varchar2(128)", d.SqlType("b", 128))
	assert.Equal("DATE", d.SqlType(time.Now(), 0))
}

