# M3U8Box

`M3U8Box` 是一个基于 Go 的命令行工具，用来从页面中提取 `m3u8` 播放地址，下载加密密钥和 `ts` 分片，并通过 `ffmpeg` 合并为本地 `mp4` 文件。

当前项目的目标是跑通一条清晰的下载链路：

1. 输入页面 URL 或 URL 文件
2. 提取页面中的 `m3u8` 地址
3. 解析 `m3u8`，得到密钥和 `ts` 列表
4. 并发下载分片
5. 生成本地可用的 `m3u8`
6. 调用 `ffmpeg` 合并输出视频

## 特性

- 支持单个 URL 下载
- 支持通过文件批量读取 URL
- 支持两种提取策略
- 直接请求 HTML 提取 `m3u8`
- 使用 `chromedp` 监听浏览器网络请求提取 `m3u8`
- 支持并发下载 `ts` 分片
- 支持命令行实时日志输出
- 支持按天写入日志文件

## 工作流程

```txt
页面 URL / URL 文件
        |
        v
提取页面标题和 m3u8 地址
        |
        v
解析 m3u8 文件
        |
        v
下载密钥和 ts 分片
        |
        v
改写本地 m3u8 引用
        |
        v
ffmpeg 合并为 mp4
```

## 环境要求

- Go 1.26 或更高版本
- 已安装 `ffmpeg`，且可直接在终端执行
- 本机可运行 `chromedp` 所依赖的 Chromium / Chrome

如果你只具备 Go 环境但没有 `ffmpeg` 或浏览器运行环境，项目即使能编译，也无法完整跑通下载链路。

## 安装与构建

### 拉取依赖并编译

```bash
go build -o M3U8Box ./cmd/M3U8Box
```

### 直接运行

```bash
go run ./cmd/M3U8Box -i="https://example.com/video-page"
```

## 命令行参数

| 参数 | 说明 | 默认值 |
| --- | --- | --- |
| `-i` | 目标页面 URL | 无 |
| `-f` | 包含多个 URL 的文本文件路径 | 无 |
| `-d` | 输出目录 | 当前工作目录 |
| `-n` | 输出文件名，不含路径 | 使用页面标题 |
| `-c` | 下载并发数 | `10` |

### 参数规则

- `-i` 和 `-f` 至少提供一个
- 当传入 `-f` 时，程序会逐行读取文件中的链接
- 空行会被自动忽略
- `-d` 必须是已存在目录

## 使用示例

### 下载单个页面

```bash
./M3U8Box -i="https://example.com/video-page"
```

### 指定输出目录和文件名

```bash
./M3U8Box -i="https://example.com/video-page" -d="/Users/you/Videos" -n="demo"
```

### 批量下载

假设 `links.txt` 内容如下：

```txt
https://example.com/video-1
https://example.com/video-2
https://example.com/video-3
```

执行：

```bash
./M3U8Box -f="./links.txt" -d="/Users/you/Videos" -c=20
```

## 日志

程序启动后会同时输出两份日志：

- 控制台标准输出
- 本地日志文件

日志目录固定为：

```txt
~/m2u8box
```

日志文件按日期滚动，文件名格式为：

```txt
YYYY-MM-DD.log
```

例如：

```txt
~/m2u8box/2026-01-01.log
```

## 输出说明

程序运行时会在输出目录下创建：

- 最终的视频文件，例如 `example.mp4`
- `static/` 临时目录

`static/` 中通常包含：

- 原始 `m3u8` 文件
- 改写后的 `m3u8` 文件
- 下载的 `ts` 分片
- 密钥文件

当前版本不会自动清理这些临时文件。

## 项目结构

```txt
M3U8Box/
├── cmd/
│   └── M3U8Box/
│       ├── main.go
│       └── options.go
├── internal/
│   ├── app/
│   │   └── m3u8.go
│   ├── downloader/
│   │   └── downloader.go
│   ├── extractor/
│   │   └── extractor.go
│   ├── logger/
│   │   └── logger.go
│   ├── m3u8/
│   │   └── parser.go
│   ├── merge/
│   │   └── ffmpeg.go
│   └── utils/
│       ├── file.go
│       └── progress.go
├── pkg/
│   └── httpclient/
│       └── client.go
├── go.mod
└── README.md
```

## 模块说明

### `cmd/M3U8Box`

命令行入口。

- `main.go` 负责启动流程和依赖组装
- `options.go` 负责命令行参数解析、输入链接处理和输出目录校验

### `internal/app`

下载主流程编排层，负责把提取、解析、下载、合并串起来。

### `internal/extractor`

页面提取层，当前包含两种策略：

- `HLExtractor`：直接请求 HTML，通过正则提取标题和 `m3u8`
- `BrowserhExtractor`：通过 `chromedp` 监听页面请求，捕获 `m3u8`

### `internal/m3u8`

负责下载并解析 `m3u8` 文件，提取密钥地址和 `ts` 列表，同时保存原始 `m3u8`。

### `internal/downloader`

负责下载密钥和 `ts` 分片，支持并发和进度条。

### `internal/merge`

负责改写本地 `m3u8` 引用，并调用 `ffmpeg` 合并成最终视频文件。

### `internal/logger`

负责统一日志输出，同时写入控制台和按天分割的日志文件。

### `pkg/httpclient`

统一封装 HTTP 客户端超时、连接复用等配置。

## 当前实现特点

- 项目偏向可运行的命令行工具，而不是通用库
- 页面提取目前依赖正则和浏览器抓包，针对性较强
- 对不同站点的兼容性取决于页面结构和 `m3u8` 格式
- 错误处理和恢复机制正在持续整理中

## 已知限制

- 当前 `m3u8` 解析规则偏依赖特定格式，不是完整通用解析器
- 某些站点如果没有在 HTML 或浏览器网络请求中直接暴露 `m3u8`，提取可能失败
- 临时文件目前不会自动清理
- 多段视频场景下，当前输出文件策略仍较简单
- 项目暂未补齐自动化测试

## 开发建议

如果你准备继续扩展这个项目，优先级比较高的方向有：

- 完善错误处理，去掉运行路径中的 `panic`
- 增加下载成功率和重试机制
- 增加临时文件清理策略
- 抽象更稳定的 `m3u8` 解析逻辑
- 增加单元测试和集成测试
- 支持更多站点和更多请求头配置

## 免责声明

请仅在合法、合规且获得授权的前提下使用本项目。使用者需自行承担目标站点访问、媒体下载和内容处理带来的责任。
