# MX - 微服务网关

MX 是一个轻量级且灵活的微服务网关框架。

## 安装

安装 MX 命令行工具：

```bash
go install github.com/hysios/mx/cmd/mx@latest
```

## 命令使用说明

### 生成命令

生成新的服务或网关：

```bash
# 生成新服务
mx gen service --name user --pkg-name github.com/example/user

# 生成新网关
mx gen gateway --pkg-name github.com/example/gateway

# 在现有项目中添加新服务
mx gen add --name payment --pkg-name github.com/example/payment
```

### 网关命令

运行网关服务器：

```bash
mx gateway --addr :8080
```

### 配置命令

MX 支持多种配置后端，以下是使用方法：

#### 设置 Redis 后端

```bash
consul services register -name="mx.Config" \
  -meta=service_type=config_provider \
  -meta=targetURI=redis://127.0.0.1:6379/mx.config \
  -address=127.0.0.1 \
  -port=6379
```

#### 配置管理

```bash
# 设置配置项
mx config set -key=key=value

# 获取配置项
mx config get -key=key

# 获取配置项（静默模式）
mx config get -key=key --quite

# 查看所有配置
mx config cat

# 使用 JSON 数据更新配置
mx config update --data='{"key": "value"}'

# 从 JSON 文件更新配置
mx config update --data=@/path/to/file.json
```

## 配置类型支持

配置项支持以下数据类型：

- string：字符串类型
- int：整数类型
- bool：布尔类型
- float：浮点数类型
- duration：时间间隔类型（如：1h, 1m30s）
- time：时间类型（格式：2006-01-02 15:04:05）

## 许可证

MIT 许可证 