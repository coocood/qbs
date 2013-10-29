package qbs

import (
	"testing"

//	"time"
)

var oracleSqlTypeResults []string = []string{
	"-",
	"NUMBER",
	"NUMBER",
	"NUMBER",
	"NUMBER",
	"NUMBER",
	"NUMBER",
	"NUMBER",
	"NUMBER",
	"NUMBER",
	"NUMBER",
	"NUMBER(16,2)",
	"NUMBER(16,2)",
	"VARCHAR2(128)",
	"CLOB",
	"DATE",
	"CLOB",
	"NUMBER",
	"NUMBER",
	"-",
	"NUMBER(16,2)",
	"DATE",
	"VARCHAR2(128)",
	"CLOB",
}

func TestSqlTypeForOrDialect(t *testing.T) {
	assert := NewAssert(t)
	d := NewOracle()
	//omitFields := []string{"Bool", "DerivedBool"}
	testModel := structPtrToModel(new(typeTestTable), false, nil)
	for index, column := range testModel.fields {
		if storedResult := oracleSqlTypeResults[index]; storedResult != "-" {
			result := d.sqlType(*column)
			assert.Equal(storedResult, result)
		}
	}
}
