package qbs

import (
	"github.com/coocood/assrt"
	"testing"
	"time"
)

func TestSqlTypeForMysqlDialect(t *testing.T) {
	assert := assrt.NewAssert(t)
	d := NewMysql()
	assert.Equal("boolean", d.sqlType(true, 0))
	var indirect interface{} = true
	assert.Equal("boolean", d.sqlType(indirect, 0))
	assert.Equal("int", d.sqlType(uint32(2), 0))
	assert.Equal("bigint", d.sqlType(int64(1), 0))
	assert.Equal("double", d.sqlType(1.8, 0))
	assert.Equal("longblob", d.sqlType([]byte("asdf"), 0))
	assert.Equal("longtext", d.sqlType("astring", 0))
	assert.Equal("longtext", d.sqlType("a", 65536))
	assert.Equal("varchar(128)", d.sqlType("b", 128))
	assert.Equal("timestamp", d.sqlType(time.Now(), 0))
}
