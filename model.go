package qbs

import (
	"bytes"
	"database/sql"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type TableNamer interface {
	TableName() string
}

const QBS_COLTYPE_INT = "int"
const QBS_COLTYPE_BOOL = "boolean"
const QBS_COLTYPE_BIGINT = "bigint"
const QBS_COLTYPE_DOUBLE = "double"
const QBS_COLTYPE_TIME = "timestamp"
const QBS_COLTYPE_TEXT = "text"

//convert struct field name to column name.
var FieldNameToColumnName func(string) string = toSnake

//convert struct name to table name.
var StructNameToTableName func(string) string = toSnake

//onvert column name to struct field name.
var ColumnNameToFieldName func(string) string = snakeToUpperCamel

//convert table name to struct name.
var TableNameToStructName func(string) string = snakeToUpperCamel

// Index represents a table index and is returned via the Indexed interface.
type index struct {
	name    string
	columns []string
	unique  bool
}

// Indexes represents an array of indexes.
type Indexes []*index

type Indexed interface {
	Indexes(indexes *Indexes)
}

// Add adds an index
func (ix *Indexes) Add(columns ...string) {
	name := strings.Join(columns, "_")
	*ix = append(*ix, &index{name: name, columns: columns, unique: false})
}

// AddUnique adds an unique index
func (ix *Indexes) AddUnique(columns ...string) {
	name := strings.Join(columns, "_")
	*ix = append(*ix, &index{name: name, columns: columns, unique: true})
}

// ModelField represents a schema field of a parsed model.
type modelField struct {
	name      string // Column name
	camelName string
	value     interface{} // Value
	pk        bool
	notnull   bool
	index     bool
	unique    bool
	updated   bool
	created   bool
	size      int
	dfault    string
	fk        string
	join      string
	colType   string
	nullable  reflect.Kind
}

// Model represents a parsed schema interface{}.
type model struct {
	pk      *modelField
	table   string
	fields  []*modelField
	refs    map[string]*reference
	indexes Indexes
}

type reference struct {
	refKey     string
	model      *model
	foreignKey bool
}

func (model *model) columnsAndValues(forUpdate bool) ([]string, []interface{}) {
	columns := make([]string, 0, len(model.fields))
	values := make([]interface{}, 0, len(columns))
	for _, column := range model.fields {
		var include bool
		if forUpdate {
			include = column.value != nil && !column.pk
		} else {
			include = true
			if column.value == nil && column.nullable == reflect.Invalid {
				include = false
			} else if column.pk {
				if intValue, ok := column.value.(int64); ok {
					include = intValue != 0
				} else if strValue, ok := column.value.(string); ok {
					include = strValue != ""
				}
			}
		}
		if include {
			columns = append(columns, column.name)
			values = append(values, column.value)
		}
	}
	return columns, values
}

func (model *model) timeField(name string) *modelField {
	for _, v := range model.fields {
		if _, ok := v.value.(time.Time); ok {
			if name == "created" {
				if v.created {
					return v
				}
			} else if name == "updated" {
				if v.updated {
					return v
				}
			}
			if v.name == name {
				return v
			}
		}
	}
	return nil
}

func (model *model) pkZero() bool {
	if model.pk == nil {
		return true
	}
	switch model.pk.value.(type) {
	case string:
		return model.pk.value.(string) == ""
	case int8:
		return model.pk.value.(int8) == 0
	case int16:
		return model.pk.value.(int16) == 0
	case int32:
		return model.pk.value.(int32) == 0
	case int64:
		return model.pk.value.(int64) == 0
	case uint8:
		return model.pk.value.(uint8) == 0
	case uint16:
		return model.pk.value.(uint16) == 0
	case uint32:
		return model.pk.value.(uint32) == 0
	case uint64:
		return model.pk.value.(uint64) == 0
	}
	return true
}

func structPtrToModel(f interface{}, root bool, omitFields []string) *model {
	model := &model{
		pk:      nil,
		table:   tableName(f),
		fields:  []*modelField{},
		indexes: Indexes{},
	}
	structType := reflect.TypeOf(f).Elem()
	structValue := reflect.ValueOf(f).Elem()
	if structType.Kind() == reflect.Ptr {
		if structType.Elem().Kind() == reflect.Struct {
			panic("did you pass a pointer to a pointer to a struct?")
		}
	}
	for i := 0; i < structType.NumField(); i++ {
		structField := structType.Field(i)
		omit := false
		for _, v := range omitFields {
			if v == structField.Name {
				omit = true
			}
		}
		if omit {
			continue
		}
		fieldValue := structValue.FieldByName(structField.Name)
		if !fieldValue.CanInterface() {
			continue
		}
		sqlTag := structField.Tag.Get("qbs")
		if sqlTag == "-" {
			continue
		}
		fieldIsNullable := false
		kind := structField.Type.Kind()
		switch kind {
		case reflect.Ptr:
			switch structField.Type.Elem().Kind() {
			case reflect.Bool, reflect.String, reflect.Int64, reflect.Float64:
				kind = structField.Type.Elem().Kind()
				fieldIsNullable = true
			default:
				continue
			}
		case reflect.Map:
			continue
		case reflect.Slice:
			elemKind := structField.Type.Elem().Kind()
			if elemKind != reflect.Uint8 {
				continue
			}
		}

		fd := new(modelField)
		parseTags(fd, sqlTag)
		fd.camelName = structField.Name
		fd.name = FieldNameToColumnName(structField.Name)
		if fieldIsNullable {
			fd.nullable = kind
			if fieldValue.IsNil() {
				fd.value = nil
			} else {
				fd.value = fieldValue.Elem().Interface()
			}
		} else {
			//not nullable case
			fd.value = fieldValue.Interface()
		}
		if _, ok := fd.value.(int64); ok && fd.camelName == "Id" {
			fd.pk = true
		}
		if fd.pk {
			model.pk = fd
		}

		model.fields = append(model.fields, fd)
		// fill in references map only in root model.
		if root {
			var fk, explicitJoin, implicitJoin bool
			var refName string
			if fd.fk != "" {
				refName = fd.fk
				fk = true
			} else if fd.join != "" {
				refName = fd.join
				explicitJoin = true
			}

			if len(fd.camelName) > 3 && strings.HasSuffix(fd.camelName, "Id") {
				fdValue := reflect.ValueOf(fd.value)
				if _, ok := fd.value.(sql.NullInt64); ok || fdValue.Kind() == reflect.Int64 {
					i := strings.LastIndex(fd.camelName, "Id")
					refName = fd.camelName[:i]
					implicitJoin = true
				}
			}

			if fk || explicitJoin || implicitJoin {
				omit := false
				for _, v := range omitFields {
					if v == refName {
						omit = true
					}
				}
				if field, ok := structType.FieldByName(refName); ok && !omit {
					fieldValue := structValue.FieldByName(refName)
					if fieldValue.Kind() == reflect.Ptr {
						model.indexes.Add(fd.name)
						if fieldValue.IsNil() {
							fieldValue.Set(reflect.New(field.Type.Elem()))
						}
						refModel := structPtrToModel(fieldValue.Interface(), false, nil)
						ref := new(reference)
						ref.foreignKey = fk
						ref.model = refModel
						ref.refKey = fd.name
						if model.refs == nil {
							model.refs = make(map[string]*reference)
						}
						model.refs[refName] = ref
					} else if !implicitJoin {
						panic("Referenced field is not pointer")
					}
				} else if !implicitJoin {
					panic("Can not find referenced field")
				}
			}
			if fd.unique {
				model.indexes.AddUnique(fd.name)
			} else if fd.index {
				model.indexes.Add(fd.name)
			}
		}
	}
	if root {
		if indexed, ok := f.(Indexed); ok {
			indexed.Indexes(&model.indexes)
		}
	}
	return model
}

func tableName(talbe interface{}) string {
	if t, ok := talbe.(string); ok {
		return t
	}
	t := reflect.TypeOf(talbe).Elem()
	for {
		c := false
		switch t.Kind() {
		case reflect.Array, reflect.Chan, reflect.Map, reflect.Ptr, reflect.Slice:
			t = t.Elem()
			c = true
		}
		if !c {
			break
		}
	}
	if tn, ok := talbe.(TableNamer); ok {
		return tn.TableName()
	}
	return StructNameToTableName(t.Name())
}

func parseTags(fd *modelField, s string) {
	if s == "" {
		return
	}
	c := strings.Split(s, ",")
	for _, v := range c {
		c2 := strings.Split(v, ":")
		if len(c2) == 2 {
			switch c2[0] {
			case "fk":
				fd.fk = c2[1]
			case "size":
				fd.size, _ = strconv.Atoi(c2[1])
			case "default":
				fd.dfault = c2[1]
			case "join":
				fd.join = c2[1]
			case "coltype":
				fd.colType = c2[1]
			default:
				panic(c2[0] + " tag syntax error")
			}
		} else {
			switch c2[0] {
			case "created":
				fd.created = true
			case "pk":
				fd.pk = true
			case "updated":
				fd.updated = true
			case "index":
				fd.index = true
			case "unique":
				fd.unique = true
			case "notnull":
				fd.notnull = true
			default:
				panic(c2[0] + " tag syntax error")
			}
		}
	}
	return
}

func toSnake(s string) string {
	buf := new(bytes.Buffer)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				buf.WriteByte('_')
			}
			buf.WriteByte(c + 32)
		} else {
			buf.WriteByte(c)
		}
	}
	return buf.String()
}

func snakeToUpperCamel(s string) string {
	buf := new(bytes.Buffer)
	first := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' && first {
			buf.WriteByte(c - 32)
			first = false
		} else if c == '_' {
			first = true
			continue
		} else {
			buf.WriteByte(c)
		}
	}
	return buf.String()
}

var ValidTags = map[string]bool{
	"pk":      true, //primary key
	"fk":      true, //foreign key
	"size":    true,
	"default": true,
	"join":    true,
	"-":       true, //ignore
	"index":   true,
	"unique":  true,
	"notnull": true,
	"updated": true,
	"created": true,
	"coltype": true,
}
