# Context 超时使用指南

## 概述

项目现在强制执行 context 超时，以防止资源耗尽和级联故障。

## 中间件级别

所有 HTTP 请求自动应用 30 秒超时：

```go
// 在 router.go 中全局应用
router.Use(s.TimeoutMiddleware.Handler(), ...)
```

## 服务层使用

### 数据库操作超时

在服务层进行数据库操作时，应该使用 `WithDBTimeout`：

```go
import "github.com/go-sonic/sonic/handler/middleware"

func (s *myServiceImpl) GetUser(ctx context.Context, id int32) (*entity.User, error) {
    // 为数据库操作添加 5 秒超时
    dbCtx, cancel := middleware.WithDBTimeout(ctx, 0) // 0 = 使用默认 5 秒
    defer cancel()

    userDAL := dal.GetQueryByCtx(dbCtx).User
    user, err := userDAL.WithContext(dbCtx).Where(userDAL.ID.Eq(id)).Take()
    return user, WrapDBErr(err)
}
```

### 自定义数据库超时

对于可能较慢的查询，可以指定更长的超时：

```go
// 复杂查询使用 10 秒超时
dbCtx, cancel := middleware.WithDBTimeout(ctx, 10*time.Second)
defer cancel()

// 执行复杂查询
results, err := dal.GetQueryByCtx(dbCtx).ComplexQuery(...)
```

### API 调用超时

调用外部 API 时使用 `WithAPITimeout`：

```go
func (s *myServiceImpl) CallExternalAPI(ctx context.Context) error {
    // 为 API 调用添加 10 秒超时
    apiCtx, cancel := middleware.WithAPITimeout(ctx, 0) // 0 = 使用默认 10 秒
    defer cancel()

    req, _ := http.NewRequestWithContext(apiCtx, "GET", "https://api.example.com", nil)
    resp, err := http.DefaultClient.Do(req)
    // ...
}
```

## 超时处理

### 检测超时

```go
func (s *myServiceImpl) DoWork(ctx context.Context) error {
    dbCtx, cancel := middleware.WithDBTimeout(ctx, 0)
    defer cancel()

    result, err := someOperation(dbCtx)
    if err != nil {
        // 检查是否是超时错误
        if errors.Is(err, context.DeadlineExceeded) {
            return xerr.WithStatus(err, xerr.StatusGatewayTimeout).
                WithMsg("Operation timed out")
        }
        return err
    }
    return nil
}
```

### 取消操作

```go
func (s *myServiceImpl) CancellableOperation(ctx context.Context) error {
    dbCtx, cancel := middleware.WithDBTimeout(ctx, 5*time.Second)
    defer cancel()

    // 在 goroutine 中执行操作
    errChan := make(chan error, 1)
    go func() {
        errChan <- doSomething(dbCtx)
    }()

    // 等待完成或超时
    select {
    case err := <-errChan:
        return err
    case <-dbCtx.Done():
        return dbCtx.Err()
    }
}
```

## 最佳实践

### ✅ 推荐做法

1. **总是传播 context**
   ```go
   func (s *service) Method(ctx context.Context) error {
       return s.otherMethod(ctx) // 传递 context
   }
   ```

2. **为长时间操作设置超时**
   ```go
   dbCtx, cancel := middleware.WithDBTimeout(ctx, 10*time.Second)
   defer cancel()
   ```

3. **检查 context 取消**
   ```go
   select {
   case <-ctx.Done():
       return ctx.Err()
   default:
       // 继续执行
   }
   ```

4. **总是调用 cancel**
   ```go
   ctx, cancel := middleware.WithDBTimeout(parentCtx, 0)
   defer cancel() // 即使操作成功也要调用
   ```

### ❌ 避免的做法

1. **不要使用 context.Background() 替代传入的 context**
   ```go
   // ❌ 错误
   func (s *service) Method(ctx context.Context) error {
       newCtx := context.Background() // 丢失了超时信息
       return s.dal.Query(newCtx)
   }

   // ✅ 正确
   func (s *service) Method(ctx context.Context) error {
       return s.dal.Query(ctx) // 传播 context
   }
   ```

2. **不要忘记 defer cancel()**
   ```go
   // ❌ 错误 - 可能导致资源泄漏
   ctx, cancel := middleware.WithDBTimeout(parentCtx, 0)
   return s.dal.Query(ctx)

   // ✅ 正确
   ctx, cancel := middleware.WithDBTimeout(parentCtx, 0)
   defer cancel()
   return s.dal.Query(ctx)
   ```

3. **不要忽略超时错误**
   ```go
   // ❌ 错误
   _, err := s.dal.Query(ctx)
   if err != nil {
       return err // 没有区分超时错误
   }

   // ✅ 正确
   _, err := s.dal.Query(ctx)
   if err != nil {
       if errors.Is(err, context.DeadlineExceeded) {
           return xerr.WithStatus(err, xerr.StatusGatewayTimeout)
       }
       return err
   }
   ```

## 默认超时值

- **HTTP 请求**: 30 秒
- **数据库操作**: 5 秒
- **API 调用**: 10 秒

这些值可以根据需要自定义。

## 监控和调试

### 日志记录超时

```go
func (s *service) Method(ctx context.Context) error {
    start := time.Now()
    dbCtx, cancel := middleware.WithDBTimeout(ctx, 0)
    defer cancel()

    err := s.dal.Query(dbCtx)
    duration := time.Since(start)

    if errors.Is(err, context.DeadlineExceeded) {
        log.Warn("Query timed out",
            zap.Duration("duration", duration),
            zap.String("operation", "GetUser"))
    }

    return err
}
```

### 指标收集

建议收集以下指标：
- 超时次数
- 操作持续时间
- 超时操作的类型

## 迁移现有代码

对于现有代码，逐步添加超时：

1. 识别长时间运行的操作
2. 添加适当的超时
3. 测试超时行为
4. 监控生产环境

优先级：
1. 数据库查询（最高）
2. 外部 API 调用
3. 文件 I/O 操作
4. 复杂计算
