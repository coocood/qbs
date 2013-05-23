package qbs

import (
    "github.com/coocood/assrt"
    "testing"
    "time"
)

func TestSqlTypeForOrDialect(t *testing.T) {
    assert := assrt.NewAssert(t)
    d := NewOracle()
    assert.Equal("integer", d.sqlType(uint32(2), 0))
    assert.Equal("bigint", d.sqlType(int64(1), 0))
    assert.Equal("double precision", d.sqlType(1.8, 0))
    assert.Equal("bytea", d.sqlType([]byte("asdf"), 0))
    assert.Equal("varchar2(255)", d.sqlType("a", 255))
    assert.Equal("varchar2(128)", d.sqlType("b", 128))
    assert.Equal("DATE", d.sqlType(time.Now(), 0))
}
