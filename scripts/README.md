# 测试脚本

这个目录包含了项目的测试脚本。

## 脚本说明

### test_api.go
测试Go后端API的基础功能。

**使用方法**:
```bash
go run scripts/test_api.go
```

**测试内容**:
- 健康检查端点
- Todo列表获取
- 创建新Todo
- 验证API响应格式

### test_frontend_api.go
测试前后端集成，验证前端代理是否正常工作。

**使用方法**:
```bash
# 确保后端和前端服务都已启动
# 后端: go run cmd/server/main.go (端口7789)
# 前端: cd frontend && npm run dev (端口3000)

go run scripts/test_frontend_api.go
```

**测试内容**:
- 后端API直接访问
- 前端代理到后端
- API响应时间
- 错误处理

## 添加新的测试脚本

如需添加新的测试脚本，请：
1. 使用 `test_*.go` 的命名格式
2. 在此README中说明测试目的和使用方法
3. 确保脚本可以在项目根目录运行