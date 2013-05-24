Qbs
=====

Qbs是一个Go语言的ORM

##特性

* 支持通过struct定义表结构，自动建表。
* 如果表已经存在，而struct定义了新的字段，Qbs会自动向数据库表添加相应的字段。
* 在查询时，struct里的字段会映射到"SELECT"语句里。
* 通过在struct里添加一个对应着父表的struct指针字段来实现关联查询。
* 增删改查都通过struct来实现。
* 查询后，需要的数据通过struct来取得。
* 通过Condition来编写查询条件，可以轻松地组织不同优先级的多个AND、OR子条件。
* 如果struct里包含Id字段，而且值大于零，这个字段的值会被视为查询条件，添加到Where语句里。
* 可以通过字段名或tag定义created和updated字段，当插入/更新时，会自动更新为当前时间。
* struct可以通过实现Validator interface，在插入或更新之前对数据进行验证。
* 目前支持MySQL， PosgreSQL， SQLite3，即将支持Oracle。
* 支持连接池。

##安装

    go get github.com/coocood/qbs

##API文档

[GoDoc](http://godoc.org/github.com/coocood/qbs)

##注意

* 新的版本可能会不兼容旧的API，使旧程序不能正常工作。
* 一次go get下载后，请保留当时的版本，如果需要其它机器上编辑，请复制当时的版本，不要在新的机器上通过go get来下载最新版本。
* 或者Fork一下，版本的更新自己来掌握。
* 每一次进行有可能破坏兼容性的更新时，会把之前的版本保存为一个新的branch, 名字是更新的日期。

##使用手册


### 首先要注册数据库：
- 参数比打开数据库要多两个，分别是数据库名:"qbs_test"和Dialect:`qbs.NewMysql()`。
- 一般只需要在应用启动是执行一次。

        func RegisterDb(){
        	qbs.Register("mysql","qbs_test@/qbs_test?charset=utf8&parseTime=true&loc=Local", "qbs_test", qbs.NewMysql())
        }


### 定义一个`User`类型：
- 如果字段字为`Id`而且类型为`int64`的话，会被Qbs视为主键。如果想用`Id`以外的名字做为主键名，可以在后加上`qbs:"pk"`来定义主键。
- `Name`后面的标签`qbs:"size:32,index"`用来定义建表时的字段属性。属性在双引号中定义，多个不同的属性用逗号区分，中间没有空格。
- 这里用到两个属性，一个是`size`，值是32，对应的SQL语句是`varchar(32)`。
- 另一个属性是`index`，建立这个字段的索引。也可以用`unique`来定义唯一约束索引。
- string类型的size属性很重要，如果加上size，而且size在数据库支持的范围内，会生成定长的varchar类型，不加size的话，对应的数据库类型是不定长的，有的数据库（MySQL)无法建立索引。


        type User struct {
            Id   int64
            Name string `qbs:"size:32,index"`
        }

- 如果需要联合索引，需要实现Indexes方法。


        func (*User) Indexes(indexes *qbs.Indexes){
            //indexes.Add("column_a", "column_b") or indexes.AddUnique("column_a", "column_b")
        }


### 新建表：
- `qbs.NewMysql`函数创建数据库的Dialect(方言)，因为不同数据库的SQL语句和数据类型有差异，所以需要不同的Dialect来适配。每个Qbs支持的数据库都有相应的Dialect函数。
- `qbs.NewMigration`函数用来创建Migration实例，用来进行建表操作。和数据库的CRUD操作的Qbs实例是分开的。
- 建表时，即使表已存在，如果发现有新增的字段或索引，会自动执行添加字段和索引的操作。
- 建表方法建议在程序启动时调用，而且完全可以用在产品数据库上。因为所有的迁移操作都是增量的，非破坏性的，所以不会有数据丢失的风险。
- `CreateTableIfNotExists`方法的参数必须是struct指针，不然会panic。


        func CreateUserTable() error{
            migration, err := qbs.GetMigration()
            if err != nil {
                return err
            }
            defer migration.Close()
            return migration.CreateTableIfNotExists(new(User))
        }


### 获取和使用`*qbs.Qbs`实例：
- 假设需要在一个http请求中获取和使用Qbs.
- 取得Qbs实例后，应该马上执行`defer q.Close()`来回收数据库连接。
- qbs使用连接池，默认大小为100，可以通过在应用启动时，调用`qbs.ChangePoolSize()`来修改。

        func GetUser(w http.ResponseWriter, r *http.Request){
        	q, err := qbs.GetQbs()
        	if err != nil {
        		fmt.Println(err)
        		w.WriteHeader(500)
        		return
        	}
        	defer q.Close()
        	u, err := FindUserById(q, 6)
        	data, _ := json.Marshal(u)
        	w.Write(data)
        }

### 插入数据：
- 如果处理一个请求需要多次进行数据库操作，最好在函数间传递*Qbs参数，这样只需要执行一次获取关闭操作就可以了。
- 插入数据时使用`Save`方法，如果`user`的主键Id没有赋值，`Save`会执行INSERT语句。
- 如果`user`的`Id`是一个正整数，`Save`会首先执行一次SELECT COUNT操作，如果发现count为0，会执行INSERT语句，否则会执行UPDATE语句。
- `Save`的参数必须是struct指针，不然会panic。


        func CreateUser(q *qbs.Qbs) (*User,error){
            user := new(User)
            user.Name = "Green"
            _, err := q.Save(user)
            return user,err
        }

### 查询数据：
- 如果需要根据Id主键查询，只要给user的Id赋值就可以了。


        func FindUserById(q *qbs.Qbs, id int64) (*User, error) {
            user := new(User)
            user.Id = id
            err := q.Find(user)
            return user, err
        }


- 查询多行需要调用`FindAll`，参数必须是slice的指针，slice的元素必须是struct的指针。


        func FindUsers(q *qbs.Qbs) ([]*User, error) {
        	var users []*User
        	err := q.Limit(10).Offset(10).FindAll(&users)
        	return users, err
        }


- 其它的查询条件，需要调用`Where`方法。这里的`WhereEqual("name", name)`相当于`Where（"name = ?", name)`，只是一个简写形式。
- `Where`/`WhereEqual`只有最后一次调用有效，之前调用的条件会被后面的覆盖掉，适用于简单的查询条件，。
- 注意，这里第一个参数字段名是`"name"`，而不是struct里的`"Name"`。所有代码里的`AbCd`形式的字段名，或类型名，在储存到数据库时会被转化为`ab_cd`的形式。
这样做的目的是为了符合go的命名规范，方便json序列化，同时避免大小写造成的数据库迁移错误。


        func FindUserByName(q *qbs.Qbs, n string) (*User, error) {
            user := new(User)
            err := q.WhereEqual("name", n).Find(user)
            return user, err
        }


- 如果需要定义复杂的查询条件，可以调用`Condition`方法。参数类型为`*Condition`，通过`NewCondition`或`NewEqualCondition`、`NewInCondition`函数来新建。
- `*Condition`类型支持`And`、`Or`等方法，可以连续调用。
- `Condition`方法同样也只能调用一次，而且不可以和`Where`同时使用。


        func FindUserByCondition(q *qbs.Qbs) (*User, error) {
            user := new(User)
            condition1 := qbs.NewCondition("id > ?", 100).Or("id < ?", 50).OrEqual("id", 75)
            condition2 := qbs.NewCondition("name != ?", "Red").And("name != ?", "Black")
            condition1.AndCondition(condition2)
            err := q.Condition(condition1).Find(user)
            return user, err
        }


### 更新一行：
- 更新一行数据需要先`Find`，再`Save`。


        func UpdateOneUser(q *qbs.Qbs, id int64, name string) (affected int64, error){
        	user, err := FindUserById(q, id)
        	if err != nil {
        		return 0, err
        	}
        	user.Name = name
        	return q.Save(user)
        }


### 更新多行：
- 多行的更新需要调用`Update`，需要注意的是，如果使用包含所有字段的struct，会把所有的字段都更新，这不会是想要的结果。
解决办法是在函数里定义临时的struct，只包含需要更新的字段。如果在函数里需要用到同名的struct，可以把冲突的部分放在block里`{...}`。


        func UpdateMultipleUsers(q *qbs.Qbs)(affected int64, error) {
        	type User struct {
        		Name string
        	}
        	user := new(User)
        	user.Name = "Blue"
        	return q.WhereEqual("name", "Green").Update(user)
        }

### 删除：
- 删除时条件不可以为空，要么在Id字段定义，要么在Where或Condition里定义。


        func DeleteUser(q *qbs.Qbs, id int64)(affected int64, err error) {
        	user := new(User)
        	user.Id = id
        	return q.Delete(user)
        }

### 定义需要关联查询的表：
- 这里`Post`里包含了一个名为`AuthorId`，类型为`int64`的字段，而且同时包含一个名为`Author`，类型为`*User`的字段。
- 使用类似 `{xxx}Id int64`, `{xxx} *{yyy}` 这样的格式，就可以定义关联查询。
- 这里`Author`这个字段因为是指针类型，所以在`Post`建表时不会被添加为column。
- 建表时，因为检测到关联字段，所以会自动为`author_id`建立索引。关联字段不需要在tag里定义索引。
- 关联字段名可以不符合以上格式，只要明确地在`AuthorId`的tag里加上`qbs:"join:Author"`，同样可以定义关联查询。
- 定义外键约束需要明确地在`AuthorId`对应的tag里添加`qbs:"fk:Author"`。
- 定义外键的同时，也就相当于定义了关联查询，同样会自动建立索引，区别仅仅是建表时添加了外键约束的语句。
- `Created time.Time`字段会在插入时写入当前时间，`Updated time.Time`字段会在更新时自动更新为当前时间。
- 如果想给自动赋值的时间字段用其它字段名，不想用"Created"，"Updated"，可以在tag里添加`qbs:"created"`，`qbs:"updated"`。


        type Post struct {
            Id int64
            AuthorId int64
            Author *User
            Content string
            Created time.Time
            Updated time.Time
        }


### 查询时忽略某些字段：
- 有时候，我们查询时并不需要某些字段，特别是关联查询的字段（比如`Author`字段），或数据很大的字段（比如`Content`字段）
，如果忽略掉，会提高查询效率。


        func FindPostsOmitContentAndCreated(q *qbs.Qbs) ([]*Post, error) {
        	var posts []*Post
        	err := q.OmitFields("Content","Created").Find(&posts)
        	return posts, err
        }


### 查询时忽略关联字段：
- 如果struct里定义了关联查询，每次Find都会自动JOIN，不需要特别指定，但有时候，我们在某一次查询时并不需要关联查询，
这时忽略掉关联查询会提高查询效率。当然我们可以用`OmitFields`实现同样的效果，但是那样需要在参数里手写字段名，不够简洁。
- 使用`OmitJoin`可以忽略所有关联查询，只返回单一表的数据，和`OmitFields`可以同时使用。


        func FindPostsOmitJoin(q *qbs.Qbs) ([]*Post, error) {
        	var posts []*Post
        	err := q.OmitJoin().OmitFields("Content").Find(&posts)
        	return posts, err
        }

。。。未完代续