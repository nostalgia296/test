# OCS 脚本客户端配置说明

将 `ocs_client_config.json` 中的配置复制到 OCS 脚本的「题库配置」中即可连接本答题服务。

## 快速使用

1. 确保 `ocs-server` 已启动（默认 `http://localhost:3000`）
2. 打开 OCS 脚本的「题库配置」界面
3. 将 `ocs_client_config.json` 的内容粘贴进去
4. 保存，开始答题

## 配置项说明

| 字段 | 值 | 说明 |
|------|-----|------|
| `name` | `OCS AI 答题服务` | 在脚本中显示的题库名称 |
| `homepage` | GitHub 仓库地址 | 题库主页（可选） |
| `url` | `http://localhost:3000/query` | 答题服务的 `/query` 接口 |
| `method` | `post` | HTTP POST 方法 |
| `contentType` | `json` | 请求和响应都使用 JSON |
| `type` | `fetch` | 使用浏览器原生 fetch API |

## data 字段解析

由于 OCS 脚本传递的参数格式与服务器期望不同，需要通过自定义字段解析方法转换：

### `question` → `${title}`

直接使用特殊占位符 `${title}` 传递题目文本。服务器会自动从题目文本中提取图片 URL。

### `type` — 题型编号转换

OCS 脚本传递的是字符串类型（`single`/`multiple`/`completion`/`judgement`），服务器需要整数编号：

| OCS 类型 | 编号 | 含义 |
|----------|------|------|
| `single` | `0` | 单选题 |
| `multiple` | `1` | 多选题 |
| `completion` | `3` | 填空题 |
| `judgement` | `4` | 判断题 |

### `options` — 选项拆分

OCS 传递的选项是用 `\n` 分隔的单个字符串，服务器需要字符串数组。

### `images` — 图片 URL

OCS 会传递图片 URL 数组（如存在），直接转发给服务器。

## handler 响应解析

服务器返回的标准响应中包含 `ocs_format` 字段，格式为 `[题目, 答案, 元信息]`，与 OCS 期望的格式完全一致。

```json
{
  "success": true,
  "question": "...",
  "answer": "...",
  "ocs_answer": "...",
  "ocs_format": ["题目", "答案", { "ai": true, ... }]
}
```

handler 优先使用 `ocs_format`，解析失败时回退到 `[question, ocs_answer]`。

## 自定义服务器地址

如果服务器不在 `localhost:3000`，修改 `url` 字段：

```json
"url": "http://192.168.1.100:5000/query"
```

## 远程服务器配置

如果服务部署在公网，使用 HTTPS + 域名：

```json
"url": "https://your-server.com/query",
"type": "GM_xmlhttpRequest"
```

注意：使用远程服务器时，需将 `type` 改为 `GM_xmlhttpRequest` 以支持跨域请求，并在脚本头部添加 `@connect` 元信息：

```
// @connect your-server.com
```

## 多题库配置

可在配置数组中添加多个题库实现故障转移或并行搜题：

```json
[
  {
    "name": "OCS AI 答题服务",
    "url": "http://localhost:3000/query",
    "method": "post",
    "contentType": "json",
    "type": "fetch",
    "data": { ... },
    "handler": "..."
  },
  {
    "name": "备用题库",
    "url": "https://example.com/search",
    "method": "get",
    "contentType": "json",
    "data": { "title": "${title}" },
    "handler": "return (res) => res.code === 1 ? [res.question, res.answer] : undefined"
  }
]
```

## 注意事项

1. **服务器必须先启动**才能答题，否则所有题目都会匹配失败
2. 模型需要在 `custom_models.json` 中正确配置并映射到题型
3. 检查服务器健康状态：`curl http://localhost:3000/api/health`
4. 日志记录在 `ocs_answers_log.csv`，可用于回顾答题情况
