package qbs

import (
	"testing"
	"time"
)

func TestSqlTypeForOrDialect(t *testing.T) {
	assert := NewAssert(t)
	d := NewOracle()
	assert.Equal("NUMBER", d.sqlType(uint32(2), 0))
	assert.Equal("NUMBER", d.sqlType(int64(1), 0))
	assert.Equal("NUMBER(16,2)", d.sqlType(1.8, 0))
	assert.Equal("CLOB", d.sqlType([]byte("asdf"), 0))
	assert.Equal("VARCHAR2(255)", d.sqlType("a", 255))
	assert.Equal("VARCHAR2(128)", d.sqlType("b", 128))
	assert.Equal("DATE", d.sqlType(time.Now(), 0))
}
