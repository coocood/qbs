package qbs

type criteria struct {
	model      *model
	condition  *Condition
	orderBys   []order
	limit      int
	offset     int
	omitFields []string
	omitJoin   bool
}

func (c *criteria) mergePkCondition(d Dialect) {
	var con *Condition
	if !c.model.pkZero() {
		expr := d.quote(c.model.pk.name) + " = ?"
		con = NewCondition(expr, c.model.pk.value)
		con.AndCondition(c.condition)
	} else {
		con = c.condition
	}
	c.condition = con
}

type order struct {
	path string
	desc bool
}

// Conditions are structured in a way to define
// complex where clause easily.
type Condition struct {
	expr string
	args []interface{}
	sub  *Condition
	isOr bool
}

func NewCondition(expr string, args ...interface{}) *Condition {
	return &Condition{
		expr: expr,
		args: args,
	}
}

//Snakecase column name
func NewEqualCondition(column string, value interface{}) *Condition {
	expr := column + " = ?"
	return NewCondition(expr, value)
}

func NewInCondition(column string, values []interface{}) *Condition {
	expr := column + " IN ("
	for _ = range values {
		expr += "?, "
	}
	expr = expr[:len(expr)-2]
	expr += ")"
	return &Condition{
		expr: expr,
		args: values,
	}
}

func (c *Condition) And(expr string, args ...interface{}) *Condition {
	if c.sub != nil {
		c.expr, c.args = c.Merge()
	}
	c.sub = NewCondition(expr, args...)
	c.isOr = false
	return c
}

//Snakecase column name
func (c *Condition) AndEqual(column string, value interface{}) *Condition {
	expr := column + " = ?"
	c.And(expr, value)
	return c
}

func (c *Condition) AndCondition(subCondition *Condition) *Condition {
	if c.sub != nil {
		c.expr, c.args = c.Merge()
	}
	c.sub = subCondition
	c.isOr = false
	return c
}

func (c *Condition) Or(expr string, args ...interface{}) *Condition {
	if c.sub != nil {
		c.expr, c.args = c.Merge()
	}
	c.sub = NewCondition(expr, args...)
	c.isOr = true
	return c
}

//Snakecase column name
func (c *Condition) OrEqual(column string, value interface{}) *Condition {
	expr := column + " = ?"
	c.Or(expr, value)
	return c
}

func (c *Condition) OrCondition(subCondition *Condition) *Condition {
	if c.sub != nil {
		c.expr, c.args = c.Merge()
	}
	c.sub = subCondition
	c.isOr = true
	return c
}

func (c *Condition) Merge() (expr string, args []interface{}) {
	expr = c.expr
	args = c.args
	if c.sub == nil {
		return
	}
	expr = "(" + expr + ")"
	if c.isOr {
		expr += " OR "
	} else {
		expr += " AND "
	}
	subExpr, subArgs := c.sub.Merge()
	expr += "(" + subExpr + ")"
	args = append(args, subArgs...)
	return expr, args
}

//Used for in condition.
func StringsToInterfaces(strs ...string) []interface{} {
	ret := make([]interface{}, len(strs))
	for i := 0; i < len(strs); i++ {
		ret[i] = strs[i]
	}
	return ret
}

func IntsToInterfaces(ints ...int64) []interface{} {
	ret := make([]interface{}, len(ints))
	for i := 0; i < len(ints); i++ {
		ret[i] = ints[i]
	}
	return ret
}
