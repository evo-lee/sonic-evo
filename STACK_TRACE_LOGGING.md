# 堆栈跟踪日志改进

## 概述

已启用错误级别的堆栈跟踪，所有 Error 及以上级别的日志都会自动包含完整的堆栈跟踪信息。

## 改进内容

### 修改前
```go
// 只在 DPanicLevel 时添加堆栈跟踪
logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.DPanicLevel))
```

### 修改后
```go
// 在 ErrorLevel 及以上添加堆栈跟踪
logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
```

## 影响

现在所有使用以下函数记录的错误都会包含堆栈跟踪：
- `log.Error()`
- `log.Errorf()`
- `log.CtxError()`
- `log.CtxErrorf()`
- `log.Fatal()`
- `log.Fatalf()`
- `log.CtxFatal()`
- `log.CtxFatalf()`

## 日志输出示例

### 修改前
```
2026-03-11T10:30:45.123Z	ERROR	handler/server.go:260	handler error
{"error": "database connection failed", "status": 500, "code": "db_error", "request_id": "abc123"}
```

### 修改后
```
2026-03-11T10:30:45.123Z	ERROR	handler/server.go:260	handler error
{"error": "database connection failed", "status": 500, "code": "db_error", "request_id": "abc123"}
github.com/go-sonic/sonic/handler.(*Server).wrapHandler.func1
	/Users/l/Documents/code/sonic/handler/server.go:260
github.com/go-sonic/sonic/handler/web/hertzadapter.(*Router).GET.func1
	/Users/l/Documents/code/sonic/handler/web/hertzadapter/router.go:45
github.com/cloudwego/hertz/pkg/app.(*Engine).ServeHTTP
	/go/pkg/mod/github.com/cloudwego/hertz@v0.8.0/pkg/app/server.go:789
...
```

## 使用场景

### 1. 生产环境调试

当生产环境出现错误时，堆栈跟踪可以帮助快速定位问题：

```go
func (s *myService) ProcessData(ctx context.Context) error {
    data, err := s.fetchData(ctx)
    if err != nil {
        // 这个错误日志现在会包含完整的堆栈跟踪
        log.CtxError(ctx, "failed to fetch data", zap.Error(err))
        return err
    }
    return nil
}
```

### 2. 错误传播追踪

可以看到错误是如何在调用链中传播的：

```go
// Service Layer
func (s *userService) GetUser(ctx context.Context, id int32) (*User, error) {
    user, err := s.dal.FindUser(ctx, id)
    if err != nil {
        log.CtxError(ctx, "failed to get user",
            zap.Int32("user_id", id),
            zap.Error(err))
        return nil, err
    }
    return user, nil
}

// Handler Layer
func (h *userHandler) GetUser(ctx web.Context) (interface{}, error) {
    user, err := h.userService.GetUser(ctx, userID)
    if err != nil {
        // 堆栈跟踪会显示从这里到 service 层的完整调用链
        return nil, err
    }
    return user, nil
}
```

### 3. 并发问题调试

在 goroutine 中的错误也会包含堆栈信息：

```go
func (s *service) ProcessAsync(ctx context.Context) {
    go func() {
        if err := s.doWork(ctx); err != nil {
            // 堆栈跟踪会显示 goroutine 的调用栈
            log.CtxError(ctx, "async work failed", zap.Error(err))
        }
    }()
}
```

## 性能考虑

### 堆栈跟踪的开销

- **CPU**: 捕获堆栈跟踪有一定的 CPU 开销
- **内存**: 堆栈信息会增加日志大小
- **磁盘**: 日志文件会变大

### 优化建议

1. **只在 Error 级别启用**（已实现）
   - Info/Warn 级别不包含堆栈跟踪
   - 减少不必要的开销

2. **使用日志轮转**（已实现）
   - Lumberjack 自动轮转日志文件
   - 防止磁盘空间耗尽

3. **合理使用日志级别**
   ```go
   // ✅ 正确 - 使用 Warn 而不是 Error
   if user == nil {
       log.Warn("user not found", zap.Int32("id", id))
       return nil, ErrNotFound
   }

   // ❌ 错误 - 不应该用 Error 记录预期的情况
   if user == nil {
       log.Error("user not found", zap.Int32("id", id))
       return nil, ErrNotFound
   }
   ```

## 日志分析

### 使用 grep 查找错误

```bash
# 查找所有错误日志
grep "ERROR" logs/sonic.log

# 查找特定错误
grep "database connection failed" logs/sonic.log

# 查看错误的堆栈跟踪
grep -A 20 "ERROR" logs/sonic.log
```

### 使用日志聚合工具

如果使用 ELK、Splunk 或其他日志聚合工具：

1. **堆栈跟踪会被解析为多行**
2. **可以按文件名和行号分组**
3. **可以创建告警规则**

## 最佳实践

### ✅ 推荐做法

1. **记录足够的上下文**
   ```go
   log.CtxError(ctx, "operation failed",
       zap.String("operation", "create_user"),
       zap.String("user_email", email),
       zap.Error(err))
   ```

2. **使用结构化日志**
   ```go
   // ✅ 使用 zap.Field
   log.Error("failed", zap.Error(err), zap.Int("id", id))

   // ❌ 不要使用格式化字符串
   log.Errorf("failed: %v, id: %d", err, id)
   ```

3. **包装错误以保留上下文**
   ```go
   if err != nil {
       return xerr.WithMsg(err, "failed to process user data")
   }
   ```

### ❌ 避免的做法

1. **不要过度记录**
   ```go
   // ❌ 不要在每一层都记录相同的错误
   func layer1() error {
       err := layer2()
       log.Error("layer1 failed", zap.Error(err))
       return err
   }

   func layer2() error {
       err := layer3()
       log.Error("layer2 failed", zap.Error(err))
       return err
   }

   // ✅ 只在最外层记录
   func layer1() error {
       err := layer2()
       if err != nil {
           log.Error("operation failed", zap.Error(err))
           return err
       }
       return nil
   }

   func layer2() error {
       return layer3()
   }
   ```

2. **不要记录敏感信息**
   ```go
   // ❌ 不要记录密码、token 等
   log.Error("auth failed",
       zap.String("password", password))

   // ✅ 只记录非敏感信息
   log.Error("auth failed",
       zap.String("username", username))
   ```

## 监控和告警

### 建议的告警规则

1. **错误率突增**
   - 5 分钟内错误数 > 100

2. **特定错误频繁出现**
   - "database connection failed" 出现 > 10 次/分钟

3. **堆栈跟踪模式**
   - 相同堆栈跟踪出现 > 50 次

## 总结

启用错误级别的堆栈跟踪后：

✅ **优点**:
- 更容易调试生产问题
- 快速定位错误源头
- 了解错误传播路径
- 发现并发问题

⚠️ **注意**:
- 日志文件会变大
- 有一定的性能开销
- 需要合理使用日志级别
- 不要记录敏感信息
