package qbs

import (
	"github.com/coocood/assrt"
	"testing"
	"time"
)

func TestSqlTypeForMysqlDialect(t *testing.T) {
	assert := assrt.NewAssert(t)
	d := NewMysql()
	assert.Equal("boolean", d.SqlType(true, 0))
	var indirect interface{} = true
	assert.Equal("boolean", d.SqlType(indirect, 0))
	assert.Equal("int", d.SqlType(uint32(2), 0))
	assert.Equal("bigint", d.SqlType(int64(1), 0))
	assert.Equal("double", d.SqlType(1.8, 0))
	assert.Equal("longblob", d.SqlType([]byte("asdf"), 0))
	assert.Equal("longtext", d.SqlType("astring", 0))
	assert.Equal("longtext", d.SqlType("a", 65536))
	assert.Equal("varchar(128)", d.SqlType("b", 128))
	assert.Equal("timestamp", d.SqlType(time.Now(), 0))
}
