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


### 首先写一个打开数据库的函数`OpenDb`：


        func OpenDb() (*sql.DB, error){
            db, err := sql.Open("mysql", "qbs_test@/qbs_test?charset=utf8&loc=Local")
            return db, err
        }


### 定义一个`User`类型：
- 如果字段字为`Id`而且类型为`int64`的话，会被Qbs视为主键。如果想用`Id`以外的名字做为主键名，可以在后加上`qbs:"pk"`来定义主键。
- `Name`后面的标签`qbs:"size:32,index"`用来定义建表时的字段属性。属性在双引号中定义，多个不同的属性用逗号区分，中间没有空格。
- 这里用到两个属性，一个是`size`，值是32，对应的SQL语句是`varchar(32)`。
- 另一个属性是`index`，建立这个字段的索引。也可以用`unique`来定义唯一约束索引。


        type User struct {
            Id   int64
            Name string `qbs:"size:32,index"`
        }


### 新建表：
- `qbs.NewMysql`函数创建数据库的Dialect(方言)，因为不同数据库的SQL语句和数据类型有差异，所以需要不同的Dialect来适配。每个Qbs支持的数据库都有相应的Dialect函数。
- `qbs.NewMigration`函数用来创建Migration实例，用来进行建表操作。和数据库的CRUD操作的Qbs实例是分开的。
- 建表时，即使表已存在，如果发现有新增的字段或索引，会自动执行添加字段和索引的操作。
- 建表方法建议在程序启动时调用，而且完全可以用在产品数据库上。因为所有的迁移操作都是增量的，非破坏性的，所以不会有数据丢失的风险。
- `CreateTableIfNotExists`方法的参数必须是struct指针，不然会panic。


        func CreateUserTable() error{
            db, err := OpenDb()
            if err != nil {
                return err
            }
            migration := qbs.NewMigration(db,"qbs_test", qbs.NewMysql())
            defer migration.Close()
            return migration.CreateTableIfNotExists(new(User))
        }


### 写一个获取`*qbs.Qbs`实例的函数：
- 这里的`qbs.GetFreeDb()`是从连接池里取出可重用的连接，是非阻塞的函数。如果连接池里有可用连接，就会取出一个，如果没有会返回`nil`。
- 连接池在最初并不会初始化任何连接，是空的，这里我们就需要打开一个新的连接。
- 取得Qbs实例后，应该马上执行`defer q.Close()`来回收数据库连接，这个函数也是非阻塞的。如果连接池未满，连接会被回收，如果连接池已满，连接会被关闭。


        func GetQbs() (q *qbs.Qbs, err error){
            db := qbs.GetFreeDB()
            if db == nil{
                db, err = OpenDb()
                if err != nil {
                    return nil,err
                }
            }
            q = qbs.New(db, qbs.NewMysql())
            return q, nil
        }

### 插入数据：
- 如果处理一个请求需要多次进行数据库操作，最好在函数间传递*Qbs参数，这样只需要执行一次获取关闭操作就可以了。
- 插入数据时使用`Save`方法，如果`user`的主键Id没有赋值，`Save`会执行INSERT语句。
- 如果`user`的`Id`是一个正整数，`Save`会首先执行一次UPDATE语句，如果发现影响的行数为0，会紧接着执行INSERT语句。
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

。。。未完待续