package qbs

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"
	"time"
)

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
	value     interface{}       // Value
	sqlTags   map[string]string // The sql struct tags for this field
	pk        bool
}

// NotNull tests if the field is declared as NOT NULL
func (field *modelField) notNull() bool {
	_, ok := field.sqlTags["notnull"]
	return ok
}

// Default returns the default value for the field
func (field *modelField) dfault() string {
	return field.sqlTags["default"]
}

// Size returns the field size, e.g. for varchars
func (field *modelField) size() int {
	v, ok := field.sqlTags["size"]
	if ok {
		i, _ := strconv.Atoi(v)
		return i
	}
	return 0
}

// Model represents a parsed schema interface{}.
type model struct {
	pk      *modelField
	table   string
	fields  []*modelField
	refs    map[string]*reference
	m2m     map[string]*m2mRelation
	indexes Indexes
}

type m2mRelation struct {
	fieldName       string
	model           *model
	targetTable     string
	targetPkRef     string
	pkRef           string
	interMediaTable string
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
			if column.value == nil {
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

func (model *model) timeFiled(name string) *modelField {
	for _, v := range model.fields {
		if _, ok := v.value.(time.Time); ok {
			if _, ok := v.sqlTags[name]; ok {
				return v
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
	case int64:
		return model.pk.value.(int64) == 0
	}
	return true
}

func structPtrToModel(f interface{}, root bool, omitFields []string) *model {
	model := &model{
		pk:      nil,
		table:   tableName(f),
		fields:  []*modelField{},
		indexes: Indexes{},
		refs:    make(map[string]*reference),
		m2m:     make(map[string]*m2mRelation),
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
		parsedSqlTags := parseTags(sqlTag)
		switch kind {
		case reflect.Ptr, reflect.Map:
			continue
		case reflect.Slice:
			elemKind := structFiled.Type.Elem().Kind()
			interMediaTable, isM2mRelation := parsedSqlTags["m2m"]
			if isM2mRelation && root {
				if elemKind != reflect.Ptr && structFiled.Type.Elem().Elem().Kind() != reflect.Struct {
					panic("field type is not slice of ptr of struct")
				}
				// targetValue := reflect.New(structFiled.Type.Elem().Elem())
				// m2mModel := structPtrToModel(targetValue.Interface(), false, nil)
				m2m := new(m2mRelation)
				// m2m.model = m2mModel
				m2m.fieldName = structFiled.Name
				m2m.interMediaTable = toSnake(interMediaTable)
				targetTable, ok := parsedSqlTags["target"]
				if !ok {
					targetTable = structFiled.Name[:len(structFiled.Name)-1]
				}
				m2m.targetTable = toSnake(targetTable)
				targetPkRef, ok := parsedSqlTags["tPkRef"]
				if !ok {
					targetPkRef = m2m.targetTable
				}
				m2m.targetPkRef = targetPkRef
				pkRef, ok := parsedSqlTags["pkRef"]
				if !ok {
					pkRef = model.table
				}
				m2m.pkRef = pkRef
				model.m2m[m2m.fieldName] = m2m

			}
			if elemKind != reflect.Uint8 {
				continue
			}
		}
		fd := new(modelField)
		fd.camelName = structFiled.Name
		fd.name = toSnake(structFiled.Name)
		fd.sqlTags = parsedSqlTags
		fd.value = structValue.FieldByName(structFiled.Name).Interface()
		if _, ok := fd.sqlTags["pk"]; ok {
			fd.pk = true
		}
		if _, ok := fd.value.(int64); ok && fd.name == "id" {
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
			refName, fk = parsedSqlTags["fk"]
			if !fk {
				refName, explicitJoin = parsedSqlTags["join"]
			}
			if len(fd.name) > 3 && strings.HasSuffix(fd.name, "_id") {
				fdValue := reflect.ValueOf(fd.value)
				if fdValue.Kind() == reflect.Int64 {
					i := strings.LastIndex(fd.name, "_id")
					refName = snakeToUpperCamel(fd.name[:i])
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
						model.refs[refName] = ref
					} else if !implicitJoin {
						panic("Referenced field is not pointer")
					}
				} else if !implicitJoin {
					panic("Can not find referenced field")
				}
			}
			if _, ok := parsedSqlTags["unique"]; ok {
				model.indexes.AddUnique(fd.name)
			} else if _, ok := parsedSqlTags["index"]; ok {
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

func validateTag(tagMap map[string]string) {
	for k, v := range tagMap {
		if _, ok := ValidTags[k]; !ok {
			panic("invalid tag:" + k + v)
		}
		switch k {
		case "fk", "join", "default", "m2m", "target", "pkRef", "tPkRef":
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
	"m2m":     true, // many to many relation tag
	"target":  true, // specify target table of m2m
	"tPkRef":  true, // targetPkRef
	"pkRef":   true, // pkRef
}
