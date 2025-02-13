# contrib

[![golang](https://img.shields.io/badge/Language-Go-green.svg?style=flat)](https://golang.org)
[![GitHub release](https://img.shields.io/github/release/yiigo/contrib.svg)](https://github.com/yiigo/contrib/releases/latest)
[![pkg.go.dev](https://img.shields.io/badge/dev-reference-007d9c?logo=go&logoColor=white&style=flat)](https://pkg.go.dev/github.com/yiigo/contrib)
[![Apache 2.0 license](http://img.shields.io/badge/license-Apache%202.0-brightgreen.svg)](http://opensource.org/licenses/apache2.0)

Go 开发实用库

## 获取

```shell
go get -u github.com/yiigo/contrib
```

## 包含

- xhash - 封装便于使用
- xcrypto - 封装便于使用(支持 AES & RSA)
- validator - 支持汉化和自定义规则
- 基于 Redis 的分布式锁
- 基于 sqlx 的轻量SQLBuilder
- 基于泛型的无限菜单分类层级树
- linklist - 一个并发安全的双向列表
- errgroup - 基于官方版本改良，支持并发协程数量控制
- xvalue - 用于处理 `k-v` 格式化的场景，如：生成签名串 等
- xcoord - 距离、方位角、经纬度与平面直角坐标系的相互转化
- timewheel - 简单实用的单层时间轮(支持一次性和多次重试任务)
- 实用的辅助方法：IP、file、time、slice、string、version compare 等

> ⚠️ 注意：如需支持协程并发复用的 `errgroup` 和 `timewheel`，请使用 👉 [nightfall](https://github.com/yiigo/nightfall)

## SQL Builder

> ⚠️ 目前支持的特性有限，复杂的SQL（如：子查询等）还需自己手写

```go
builder := contrib.NewSQLBuilder(*sqlx.DB, func(ctx context.Context, query string, args ...any) {
    fmt.Println(query, args)
})
```

### 👉 Query

```go
ctx := context.Background()

type User struct {
    ID     int    `db:"id"`
    Name   string `db:"name"`
    Age    int    `db:"age"`
    Phone  string `db:"phone,omitempty"`
}

var (
    record User
    records []User
)

builder.Wrap(
    contrib.Table("user"),
    contrib.Where("id = ?", 1),
).One(ctx, &record)
// SELECT * FROM user WHERE (id = ?)
// [1]

builder.Wrap(
    contrib.Table("user"),
    contrib.Where("name = ? AND age > ?", "yiigo", 20),
).All(ctx, &records)
// SELECT * FROM user WHERE (name = ? AND age > ?)
// [yiigo 20]

builder.Wrap(
    contrib.Table("user"),
    contrib.Where("name = ?", "yiigo"),
    contrib.Where("age > ?", 20),
).All(ctx, &records)
// SELECT * FROM user WHERE (name = ?) AND (age > ?)
// [yiigo 20]

builder.Wrap(
    contrib.Table("user"),
    contrib.WhereIn("age IN (?)", []int{20, 30}),
).All(ctx, &records)
// SELECT * FROM user WHERE (age IN (?, ?))
// [20 30]

builder.Wrap(
    contrib.Table("user"),
    contrib.Select("id", "name", "age"),
    contrib.Where("id = ?", 1),
).One(ctx, &record)
// SELECT id, name, age FROM user WHERE (id = ?)
// [1]

builder.Wrap(
    contrib.Table("user"),
    contrib.Distinct("name"),
    contrib.Where("id = ?", 1),
).One(ctx, &record)
// SELECT DISTINCT name FROM user WHERE (id = ?)
// [1]

builder.Wrap(
    contrib.Table("user"),
    contrib.LeftJoin("address", "user.id = address.user_id"),
    contrib.Where("user.id = ?", 1),
).One(ctx, &record)
// SELECT * FROM user LEFT JOIN address ON user.id = address.user_id WHERE (user.id = ?)
// [1]

builder.Wrap(
    contrib.Table("address"),
    contrib.Select("user_id", "COUNT(*) AS total"),
    contrib.GroupBy("user_id"),
    contrib.Having("user_id = ?", 1),
).All(ctx, &records)
// SELECT user_id, COUNT(*) AS total FROM address GROUP BY user_id HAVING (user_id = ?)
// [1]

builder.Wrap(
    contrib.Table("user"),
    contrib.Where("age > ?", 20),
    contrib.OrderBy("age ASC", "id DESC"),
    contrib.Offset(5),
    contrib.Limit(10),
).All(ctx, &records)
// SELECT * FROM user WHERE (age > ?) ORDER BY age ASC, id DESC LIMIT ? OFFSET ?
// [20, 10, 5]

wrap1 := builder.Wrap(
    contrib.Table("user_1"),
    contrib.Where("id = ?", 2),
)

builder.Wrap(
    contrib.Table("user_0"),
    contrib.Where("id = ?", 1),
    contrib.Union(wrap1),
).All(ctx, &records)
// (SELECT * FROM user_0 WHERE (id = ?)) UNION (SELECT * FROM user_1 WHERE (id = ?))
// [1, 2]

builder.Wrap(
    contrib.Table("user_0"),
    contrib.Where("id = ?", 1),
    contrib.UnionAll(wrap1),
).All(ctx, &records)
// (SELECT * FROM user_0 WHERE (id = ?)) UNION ALL (SELECT * FROM user_1 WHERE (id = ?))
// [1, 2]

builder.Wrap(
    contrib.Table("user_0"),
    contrib.WhereIn("age IN (?)", []int{10, 20}),
    contrib.Limit(5),
    contrib.Union(
        builder.Wrap(
            contrib.Table("user_1"),
            contrib.Where("age IN (?)", []int{30, 40}),
            contrib.Limit(5),
        ),
    ),
).All(ctx, &records)
// (SELECT * FROM user_0 WHERE (age IN (?, ?)) LIMIT ?) UNION (SELECT * FROM user_1 WHERE (age IN (?, ?)) LIMIT ?)
// [10, 20, 5, 30, 40, 5]
```

### 👉 Insert

```go
ctx := context.Background()

type User struct {
    ID     int64  `db:"-"`
    Name   string `db:"name"`
    Age    int    `db:"age"`
    Phone  string `db:"phone,omitempty"`
}

builder.Wrap(Table("user")).Insert(ctx, &User{
    Name: "yiigo",
    Age:  29,
})
// INSERT INTO user (name, age) VALUES (?, ?)
// [yiigo 29]

builder.Wrap(contrib.Table("user")).Insert(ctx, map[string]any{
    "name": "yiigo",
    "age":  29,
})
// INSERT INTO user (name, age) VALUES (?, ?)
// [yiigo 29]
```

### 👉 Batch Insert

```go
ctx := context.Background()

type User struct {
    ID     int64  `db:"-"`
    Name   string `db:"name"`
    Age    int    `db:"age"`
    Phone  string `db:"phone,omitempty"`
}

builder.Wrap(Table("user")).BatchInsert(ctx, []*User{
    {
        Name: "yiigo",
        Age:  20,
    },
    {
        Name: "yiigo",
        Age:  29,
    },
})
// INSERT INTO user (name, age) VALUES (?, ?), (?, ?)
// [yiigo 20 yiigo 29]

builder.Wrap(contrib.Table("user")).BatchInsert(ctx, []map[string]any{
    {
        "name": "yiigo",
        "age":  20,
    },
    {
        "name": "yiigo",
        "age":  29,
    },
})
// INSERT INTO user (name, age) VALUES (?, ?), (?, ?)
// [yiigo 20 yiigo 29]
```

### 👉 Update

```go
ctx := context.Background()

type User struct {
    Name   string `db:"name"`
    Age    int    `db:"age"`
    Phone  string `db:"phone,omitempty"`
}

builder.Wrap(
    contrib.Table("user"),
    contrib.Where("id = ?", 1),
).Update(ctx, &User{
    Name: "yiigo",
    Age:  29,
})
// UPDATE user SET name = ?, age = ? WHERE (id = ?)
// [yiigo 29 1]

builder.Wrap(
    contrib.Table("user"),
    contrib.Where("id = ?", 1),
).Update(ctx, map[string]any{
    "name": "yiigo",
    "age":  29,
})
// UPDATE user SET name = ?, age = ? WHERE (id = ?)
// [yiigo 29 1]

builder.Wrap(
    contrib.Table("product"),
    contrib.Where("id = ?", 1),
).Update(ctx, map[string]any{
    "price": contrib.SQLExpr("price * ? + ?", 2, 100),
})
// UPDATE product SET price = price * ? + ? WHERE (id = ?)
// [2 100 1]
```

### 👉 Delete

```go
ctx := context.Background()

builder.Wrap(
    contrib.Table("user"),
    contrib.Where("id = ?", 1),
).Delete(ctx)
// DELETE FROM user WHERE id = ?
// [1]

builder.Wrap(contrib.Table("user")).Truncate(ctx)
// TRUNCATE user
```

### 👉 Transaction

```go
builder.Transaction(context.Background(), func(ctx context.Context, tx contrib.TXBuilder) error {
    _, err := tx.Wrap(
        contrib.Table("address"),
        contrib.Where("user_id = ?", 1),
    ).Update(ctx, map[string]any{"default": 0})
    if err != nil {
        return err
    }

    _, err = tx.Wrap(
        contrib.Table("address"),
        contrib.Where("id = ?", 1),
    ).Update(ctx, map[string]any{"default": 1})

    return err
})
```

**Enjoy 😊**
