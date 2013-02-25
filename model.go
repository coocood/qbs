package qbs

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Index represents a table index and is returned via the Indexed interface.
type Index struct {
	Name    string
	Columns []string
	Unique  bool
}

// Indexes represents an array of indexes.
type Indexes []*Index

type Indexed interface {
	Indexes(indexes *Indexes)
}

// Add adds an index
func (ix *Indexes) Add(columns ...string) {
	name := strings.Join(columns, "_")
	*ix = append(*ix, &Index{Name: name, Columns: columns, Unique: false})
}

// AddUnique adds an unique index
func (ix *Indexes) AddUnique(columns ...string) {
	name := strings.Join(columns, "_")
	*ix = append(*ix, &Index{Name: name, Columns: columns, Unique: true})
}

// ModelField represents a schema field of a parsed model.
type ModelField struct {
	Name      string // Column name
	CamelName string
	Value     interface{}       // Value
	SqlTags   map[string]string // The sql struct tags for this field
	PK        bool
}

// NotNull tests if the field is declared as NOT NULL
func (field *ModelField) NotNull() bool {
	_, ok := field.SqlTags["notnull"]
	return ok
}

// Default returns the default value for the field
func (field *ModelField) Default() string {
	return field.SqlTags["default"]
}

// Size returns the field size, e.g. for varchars
func (field *ModelField) Size() int {
	v, ok := field.SqlTags["size"]
	if ok {
		i, _ := strconv.Atoi(v)
		return i
	}
	return 0
}

// Model represents a parsed schema interface{}.
type Model struct {
	Pk      *ModelField
	Table   string
	Fields  []*ModelField
	Refs    map[string]*Reference
	Indexes Indexes
}

type Reference struct {
	RefKey     string
	Model      *Model
	ForeignKey bool
}

func (model *Model) columnsAndValues(forUpdate bool) ([]string, []interface{}) {
	columns := make([]string, 0, len(model.Fields))
	values := make([]interface{}, 0, len(columns))
	for _, column := range model.Fields {
		var include bool
		if forUpdate {
			include = column.Value != nil && !column.PK
		} else {
			include = true
			if column.Value == nil{
				include = false
			}else if column.PK {
				if intValue,ok := column.Value.(int64); ok{
					include = intValue != 0
				}else if strValue, ok := column.Value.(string); ok{
					include = strValue != ""
				}
			}
		}
		if include {
			columns = append(columns, column.Name)
			values = append(values, column.Value)
		}
	}
	return columns, values
}

func (model *Model) timeFiled(name string) *ModelField {
	for _, v := range model.Fields {
		if _, ok := v.Value.(time.Time); ok {
			if _, ok := v.SqlTags[name]; ok {
				return v
			}
			if v.Name == name {
				return v
			}
		}
	}
	return nil
}

func (model *Model) pkZero() bool {
	if model.Pk == nil {
		return true
	}
	switch model.Pk.Value.(type) {
	case string:
		return model.Pk.Value.(string) == ""
	case int64:
		return model.Pk.Value.(int64) == 0
	}
	return true
}

func structPtrToModel(f interface{}, root bool, omitFields []string) *Model {
	model := &Model{
		Pk:      nil,
		Table:   tableName(f),
		Fields:  []*ModelField{},
		Indexes: Indexes{},
		Refs:    make(map[string]*Reference),
	}
	structType := reflect.TypeOf(f).Elem()
	structValue := reflect.ValueOf(f).Elem()
	for i := 0; i < structType.NumField(); i++ {
		structFiled := structType.Field(i)
		omit := false
		for _, v := range omitFields {
			if v == structFiled.Name {
				omit = true
			}
		}
		if omit {
			continue
		}
		sqlTag := structFiled.Tag.Get("qbs")
		if sqlTag == "-" {
			continue
		}
		kind := structFiled.Type.Kind()
		switch kind {
		case reflect.Ptr, reflect.Map:
			continue
		case reflect.Slice:
			elemKind := structFiled.Type.Elem().Kind()
			if elemKind != reflect.Uint8 {
				continue
			}
		}
		parsedSqlTags := parseTags(sqlTag)
		fd := new(ModelField)
		fd.CamelName = structFiled.Name
		fd.Name = toSnake(structFiled.Name)
		fd.SqlTags = parsedSqlTags
		fd.Value = structValue.FieldByName(structFiled.Name).Interface()
		if _, ok := fd.SqlTags["pk"]; ok {
			fd.PK = true
		}
		if _, ok := fd.Value.(int64); ok && fd.Name == "id" {
			fd.PK = true
		}
		if fd.PK {
			model.Pk = fd
		}
		model.Fields = append(model.Fields, fd)
		// fill in references map only in root model.
		if root {
			var fk, explicitJoin, implicitJoin bool
			var refName string
			refName, fk = parsedSqlTags["fk"]
			if !fk {
				refName, explicitJoin = parsedSqlTags["join"]
			}
			if len(fd.Name) > 3 && strings.HasSuffix(fd.Name, "_id") {
				fdValue := reflect.ValueOf(fd.Value)
				if fdValue.Kind() == reflect.Int64 {
					i := strings.LastIndex(fd.Name, "_id")
					refName = snakeToUpperCamel(fd.Name[:i])
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
						model.Indexes.Add(fd.Name)
						if fieldValue.IsNil() {
							fieldValue.Set(reflect.New(field.Type.Elem()))
						}
						refModel := structPtrToModel(fieldValue.Interface(), false, nil)
						ref := new(Reference)
						ref.ForeignKey = fk
						ref.Model = refModel
						ref.RefKey = fd.Name
						model.Refs[refName] = ref
					} else if !implicitJoin {
						panic("Referenced field is not pointer")
					}
				} else if !implicitJoin {
					panic("Can not find referenced field")
				}
			}
			if _, ok := parsedSqlTags["unique"]; ok {
				model.Indexes.AddUnique(fd.Name)
			} else if _, ok := parsedSqlTags["index"]; ok {
				model.Indexes.Add(fd.Name)
			}
		}
	}
	if root {
		if indexed, ok := f.(Indexed); ok {
			indexed.Indexes(&model.Indexes)
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
	return toSnake(t.Name())
}

func parseTags(s string) map[string]string {
	if s == "" {
		return nil
	}
	c := strings.Split(s, ",")
	m := make(map[string]string)
	for _, v := range c {
		c2 := strings.Split(v, ":")
		if len(c2) == 2 {
			m[c2[0]] = c2[1]
		} else {
			m[v] = ""
		}
	}
	validateTag(m)
	return m
}

func toSnake(s string) string {
	buf := bytes.NewBufferString("")
	for i, v := range s {
		if i > 0 && v >= 'A' && v <= 'Z' {
			buf.WriteRune('_')
		}
		buf.WriteRune(v)
	}
	return strings.ToLower(buf.String())
}

func snakeToUpperCamel(s string) string {
	buf := bytes.NewBufferString("")
	for _, v := range strings.Split(s, "_") {
		if len(v) > 0 {
			buf.WriteString(strings.ToUpper(v[:1]))
			buf.WriteString(v[1:])
		}
	}
	return buf.String()
}

func validateTag(tagMap map[string]string) {
	for k, v := range tagMap {
		if _, ok := ValidTags[k]; !ok {
			panic("invalid tag:" + k + v)
		}
		switch k {
		case "fk", "join", "default":
			if len(v) == 0 {
				panic(k + " tag syntax error")
			}
		case "size":
			if _, err := strconv.Atoi(v); err != nil {
				panic(k + " tag syntax error")
			}
		default:
			if len(v) != 0 {
				panic(k + " tag syntax error")
			}
		}
	}
}

var ValidTags = map[string]bool{
	"pk":      true,
	"fk":      true,
	"size":    true,
	"default": true,
	"join":    true,
	"-":       true,
	"index":   true,
	"unique":  true,
	"notnull": true,
	"updated": true,
	"created": true,
}
