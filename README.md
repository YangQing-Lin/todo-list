# Go Todo List

一个使用Go语言后端和React前端的Todo List全栈应用项目。

## 项目概述

这是一个为了学习Go语言而创建的Todo List应用，采用前后端分离架构：
- **后端**: Go + SQLite（使用标准库）
- **前端**: React + TypeScript + Vite
- **架构**: RESTful API

## 技术栈

### 后端
- **语言**: Go 1.20+
- **框架**: 标准库 net/http
- **数据库**: SQLite（计划中）
- **特点**: 无第三方框架依赖

### 前端
- **框架**: React 18
- **语言**: TypeScript
- **构建工具**: Vite
- **HTTP客户端**: Axios
- **样式**: CSS Modules

## 项目结构

```
todo-list/
├── cmd/server/              # Go后端主程序
│   └── main.go             # 服务器入口
├── handler/                # HTTP处理器
│   └── handler.go          # API处理器
├── model/                  # 数据模型
│   └── todo.go             # Todo结构体
├── api/                    # API路由
│   └── routes.go           # 路由配置
├── database/               # 数据库相关
│   └── (待实现)
├── frontend/               # React前端
│   ├── src/
│   │   ├── components/     # React组件
│   │   ├── pages/          # 页面组件
│   │   ├── services/       # API服务
│   │   ├── types/          # TypeScript类型
│   │   └── styles/         # 样式文件
│   ├── public/             # 静态资源
│   └── docs/               # 前端文档
├── scripts/                # 测试脚本
├── docs/                   # 项目文档
├── go.mod                  # Go模块
└── README.md               # 项目说明
```

## 快速开始

### 前置要求
- Go 1.20+
- Node.js 16+
- npm 或 yarn

### 1. 克隆项目
```bash
git clone <repository-url>
cd todo-list
```

### 2. 启动后端服务
```bash
# 启动Go服务器（端口: 7789）
go run cmd/server/main.go
```

### 3. 启动前端服务
```bash
# 进入前端目录
cd frontend

# 安装依赖
npm install

# 启动开发服务器（端口: 3000）
npm run dev
```

### 4. 访问应用
- 前端应用: http://localhost:3000
- 后端API: http://localhost:7789

## API文档

### 端点列表

| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/` | API信息 |
| GET | `/health` | 健康检查 |
| GET | `/api/todos` | 获取所有Todos |
| POST | `/api/todos` | 创建新Todo |

### 响应格式

```json
{
  "success": true,
  "data": {},
  "error": null,
  "message": "操作成功"
}
```

## 测试

### 运行API测试
```bash
# 测试后端API
go run scripts/test_api.go

# 测试前后端集成
go run scripts/test_frontend_api.go
```

## 开发进度

### ✅ 已完成
- [x] Go后端基础架构
- [x] React前端项目搭建
- [x] 基础CRUD功能（创建、列表）
- [x] 响应式UI设计
- [x] API代理配置

### 🚧 进行中
- [ ] SQLite数据库集成
- [ ] 完整的CRUD操作（更新、删除）
- [ ] 数据持久化

### 📋 计划中
- [ ] 优先级系统
- [ ] 截止日期管理
- [ ] 搜索和过滤
- [ ] 标签系统
- [ ] 单元测试
- [ ] Docker部署

## 学习路线

详细的学习计划请参考：[系统设计与学习路线.md](docs/系统设计与学习路线.md)

### 第一周目标
- Go语言基础语法
- HTTP服务器开发
- React基础组件

### 第二周目标
- Go数据库操作
- 状态管理
- 错误处理

### 第三周目标
- Go并发编程
- 前端性能优化
- 部署和监控

## 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

## 联系方式

如有问题或建议，请提交 Issue 或联系项目维护者。

---

**记住Linus的话**: "Talk is cheap. Show me the code." 👨‍💻