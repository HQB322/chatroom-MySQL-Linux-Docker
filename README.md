# 聊天室项目

一个基于 Go 语言开发的多人在线聊天室系统，支持用户注册、登录、实时群聊和私聊功能。

## 🌟 功能特性

### 用户认证系统

- **用户注册**：支持用户名、密码、昵称注册
- **用户登录**：安全登录验证
- **输入验证**：用户名/密码格式校验，防止 SQL 注入
- **唯一性检查**：用户名和昵称唯一性验证

### 聊天功能

- **实时群聊**：所有在线用户参与的大厅聊天
- **私聊功能**：支持用户间一对一私密聊天
- **在线用户列表**：实时查看当前在线用户
- **消息广播**：高效的消息广播机制

### 系统特性

- **并发安全**：使用读写锁保护共享资源
- **连接管理**：自动处理客户端连接和断开
- **错误恢复**：防止 panic 导致服务崩溃
- **超时控制**：防止资源泄漏
- **健康检查**：支持容器健康检查

## 📁 项目结构

text

```
chatroom/
├── Dockerfile.client           # 客户端 Dockerfile
├── Dockerfile.server           # 服务器 Dockerfile
├── docker-compose.yml          # Docker Compose 配置
├── go.mod                      # Go 模块定义
├── go.sum                      # 依赖锁文件
├── main.go                     # 服务器主程序
├── client/
│   └── main.go                 # 客户端程序
├── auth/
│   └── auth.go                 # 认证模块
├── database/
│   └── db.go                   # 数据库模块
├── data/                       # 数据目录（自动创建）
├── logs/                       # 日志目录（自动创建）
└── README.md                   # 项目说明文档
```



## 🚀 快速开始

### 环境要求

- Docker & Docker Compose
- 或 Go 1.21+

### 使用 Docker 一键部署

1. **克隆项目**

bash

```
git clone <repository-url>
cd chatroom
```



1. **启动服务**

bash

```
docker-compose up -d
```



1. **查看服务状态**

bash

```
docker-compose ps
```



1. **连接客户端**

bash

```
# 进入客户端容器
docker exec -it chatroom-client /bin/bash

# 运行客户端程序
./client
```



### 手动部署（开发环境）

1. **启动 MySQL 数据库**

bash

```
# 使用 Docker 启动 MySQL
docker run -d \
  --name chatroom-mysql \
  -e MYSQL_ROOT_PASSWORD=root123456 \
  -e MYSQL_DATABASE=chatroom \
  -e MYSQL_USER=chatroom_user \
  -e MYSQL_PASSWORD=chatroom123 \
  -p 3306:3306 \
  mysql:8.0
```



1. **设置环境变量**

bash

```
export MYSQL_HOST=localhost
export MYSQL_PORT=3306
export MYSQL_DATABASE=chatroom
export MYSQL_USER=chatroom_user
export MYSQL_PASSWORD=chatroom123
```



1. **启动服务器**

bash

```
go run main.go
```



1. **启动客户端**（新终端）

bash

```
cd client
go run main.go
```



## 🔧 配置说明

### 环境变量配置

| 变量名         | 默认值        | 说明           |
| :------------- | :------------ | :------------- |
| MYSQL_HOST     | localhost     | MySQL 主机地址 |
| MYSQL_PORT     | 3306          | MySQL 端口     |
| MYSQL_DATABASE | chatroom      | 数据库名       |
| MYSQL_USER     | chatroom_user | 数据库用户     |
| MYSQL_PASSWORD | chatroom123   | 数据库密码     |
| SERVER_HOST    | localhost     | 服务器地址     |
| SERVER_PORT    | 8080          | 服务器端口     |
| PORT           | 8080          | 服务器监听端口 |
| HEALTH_PORT    | 8081          | 健康检查端口   |

### Docker Compose 服务

- **mysql**: MySQL 8.0 数据库服务
- **chatroom-server**: 聊天室服务器
- **chatroom-client**: 聊天室客户端

## 📖 使用指南

### 1. 首次启动

1. 系统会自动创建测试用户：
   - 用户名: `user1`, 密码: `123456`, 昵称: `用户1`
   - 用户名: `user2`, 密码: `123456`, 昵称: `用户2`
   - 用户名: `user3`, 密码: `123456`, 昵称: `用户3`

### 2. 客户端操作

#### 认证阶段

text

```
=== 认证菜单 ===
1. 登录 - 格式: /login <用户名> <密码>
   示例: /login user1 123456

2. 注册 - 格式: /register <用户名> <密码> <昵称>
   示例: /register newuser 123456 新用户

3. 退出 - 输入: /exit
=================
```



#### 聊天阶段

text

```
📋 可用命令:
/help        - 显示帮助信息
/list        - 查看在线用户
/w 昵称 消息 - 发送私聊消息
/whisper 昵称 消息 - 发送私聊消息
/exit        - 退出聊天室
```



### 3. 验证规则

- **用户名**: 3-20位字母、数字、下划线
- **密码**: 至少6位
- **昵称**: 2-20位中文、字母、数字、下划线

## 🛠️ 开发指南

### 项目构建

bash

```
# 构建服务器
go build -o server main.go

# 构建客户端
cd client && go build -o client main.go
```



### Docker 构建

bash

```
# 构建服务器镜像
docker build -f Dockerfile.server -t chatroom-server .

# 构建客户端镜像
docker build -f Dockerfile.client -t chatroom-client .
```



### 测试连接

bash

```
# 健康检查
curl http://localhost:8081/health

# 就绪检查
curl http://localhost:8081/ready
```



## 📊 监控与日志

### 查看日志

bash

```
# 查看服务器日志
docker logs chatroom-server -f

# 查看客户端日志
docker logs chatroom-client -f

# 查看数据库日志
docker logs chatroom-mysql -f
```



### 数据持久化

- MySQL 数据存储在 `mysql-data` Docker 卷
- 服务器日志存储在 `./logs` 目录
- 其他数据存储在 `./data` 目录

## 🔒 安全特性

1. **SQL 注入防护**: 使用参数化查询
2. **输入验证**: 客户端和服务端双重验证
3. **连接安全**: 超时控制和资源清理
4. **唯一性约束**: 用户名和昵称唯一性保证

## 🐛 故障排除

### 常见问题

1. **无法连接数据库**
   - 检查 MySQL 服务是否运行
   - 验证环境变量配置
   - 检查端口是否被占用
2. **客户端无法连接服务器**
   - 确认服务器正在运行
   - 检查网络连接
   - 验证端口配置
3. **注册失败**
   - 检查用户名/昵称是否符合格式
   - 确认用户名/昵称未被占用
   - 检查密码长度

### 查看服务状态

bash

```
# Docker Compose 状态
docker-compose ps

# 容器日志
docker-compose logs

# 服务健康状态
curl http://localhost:8081/health
```



## 📈 性能特性

- **并发处理**: 使用 goroutine 处理多客户端
- **连接池**: 数据库连接复用
- **消息缓冲**: 通道缓冲防止阻塞
- **读写分离**: 读写锁提高并发性能
- **工作池**: 限制并发处理数量

## 🧪 测试

### 压力测试

bash

```
# 可以使用多个客户端连接进行测试
for i in {1..10}; do
  docker run -d --name client-$i chatroom-client
done
```



### 数据库测试

bash

```
# 连接到 MySQL
docker exec -it chatroom-mysql mysql -uchatroom_user -pchatroom123 chatroom

# 查看用户表
SELECT * FROM users;
```



## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](https://license/) 文件了解详情

## 👥 作者

- **创建者** - [Your Name]
- **邮箱** - your.email@example.com

## 🙏 致谢

- 感谢所有贡献者
- 感谢开源社区的支持
- 使用技术栈：Go, MySQL, Docker
