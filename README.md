# Backup-Go | 备份系统

<div align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go Version" />
  <img src="https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square" alt="License: MIT" />
</div>

一个使用Go语言开发的灵活备份系统，支持数据库备份和文件备份，并提供Web界面进行管理。

A flexible backup system developed with Go language, supporting database and file backups with a web-based management interface.

## ✨ 功能特点 | Features

- 🗄️ 支持MySQL数据库备份
- 📁 支持文件和目录备份
- 💾 支持本地存储和S3协议存储
- 🔌 可扩展的存储和备份类型
- ⏱️ 基于Cron的任务调度
- 🌐 美观的Web管理界面
- 📊 备份历史记录和下载功能
- 🧹 自动清理过期备份

## 🔧 系统要求 | Requirements

- Go 1.21+
- MySQL 5.7+ 或 SQLite
- mysqldump命令行工具 (用于数据库备份)

## 🚀 快速开始 | Quick Start

### 1. 克隆仓库 | Clone Repository

```bash
git clone https://github.com/yourusername/backup-go.git
cd backup-go
```

### 2. 配置 | Configuration

创建或编辑`config.yaml`文件：

```yaml
# 服务器配置
server:
  port: 8080

# 数据库配置
database:
  type: sqlite  # mysql或sqlite
  # 如果需要使用MySQL，请配置以下内容
  # host: 127.0.0.1
  # username: root
  # password: your-password
  # port: 3306
  # name: backup_go
```

> **注意**: 如果您需要使用S3协议存储，可以在Web界面中进行配置。

### 3. 构建和运行 | Build and Run

```bash
# 下载依赖
go mod tidy

# 构建
go build -o backup-go

# 运行
./backup-go
```

### 4. 访问Web界面 | Access Web UI

在浏览器中访问 http://localhost:8080

## 🐳 Docker 部署指南 | Docker Deployment Guide

### 使用预构建镜像 | Using Pre-built Images

我们提供了预构建的多架构 Docker 镜像，支持 `amd64`、`arm64` 等平台：

```bash
# 拉取最新版本
docker pull zzzgr/backup-go:1.0.0-amd64

# 运行容器
docker run -d --name backup-go -p 8080:8080 \
     -v $(pwd):/app \
     zzzgr/backup-go:1.0.0-amd64
```

### 目录映射说明 | Volume Mapping

- `/app/config.yaml`: 配置文件

## 📂 项目结构 | Project Structure

```
backup-go/
├── api/                # API层，包含路由和控制器
│   ├── controller/     # 控制器
│   ├── middleware/     # 中间件
│   └── router/         # 路由配置
├── config/             # 配置相关
├── entity/             # 数据实体模型
├── model/              # 数据访问层
├── public/             # 静态资源
├── repository/         # 数据仓库
├── service/            # 业务逻辑层
│   ├── backup/         # 备份服务实现
│   ├── cleanup/        # 清理服务
│   ├── config/         # 配置服务
│   ├── scheduler/      # 调度服务
│   └── storage/        # 存储服务实现
├── test/               # 测试代码
├── config.yaml         # 配置文件
└── main.go             # 程序入口
```

## 🔍 使用指南 | Usage Guide

### 创建数据库备份任务 | Create Database Backup Task

1. 在Web界面点击"新建任务"
2. 选择"数据库备份"类型
3. 填写数据库连接信息
4. 配置调度计划（Cron表达式）
5. 选择存储方式
6. 保存任务

### 创建文件备份任务 | Create File Backup Task

1. 在Web界面点击"新建任务"
2. 选择"文件备份"类型
3. 填写要备份的文件/目录路径（每行一个）
4. 配置调度计划（Cron表达式）
5. 选择存储方式
6. 保存任务

### 手动执行任务 | Manual Execution

在任务列表中点击对应任务的"执行"按钮即可手动触发备份任务。

### 查看和下载备份 | View and Download Backups

在导航栏切换到"备份记录"页面，可以查看所有备份记录，对于成功的备份可以点击"下载"按钮下载备份文件。

## 🏗️ 架构设计 | Architecture Design

本系统采用模块化设计，易于扩展：

- **存储服务接口**: 支持本地存储和S3协议存储，可以扩展更多存储方式
- **备份服务接口**: 支持数据库备份和文件备份，可以扩展更多备份类型
- **Cron调度器**: 基于robfig/cron库实现任务调度
- **Web API**: 提供RESTful API接口
- **前端界面**: 基于Bootstrap实现的现代化Web界面

## 🧪 最新特性 | Latest Features

- **支持SQLite**: 除MySQL外，现在还支持SQLite作为系统数据库
- **备份统计**: 添加了备份统计页面，展示备份趋势和使用情况
- **多语言支持**: 界面支持中文和英文
- **S3兼容存储**: 支持Amazon S3, MinIO, 阿里云OSS等S3兼容存储
- **备份数据加密**: 支持对备份数据进行加密存储

## 🤝 贡献 | Contributing

我们欢迎各种形式的贡献，包括但不限于：

- 提交 Issue 报告bug或提出新功能建议
- 提交 Pull Request 改进代码
- 改进文档
- 分享使用经验

贡献前请查看我们的贡献指南。

## 📄 许可证 | License

本项目采用 [MIT 许可证](LICENSE) 进行许可。
