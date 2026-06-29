# OCS AI 答题服务 使用文档

## 1. 概述

OCS AI 答题服务是一个基于 **OpenAI Chat Completions API** 兼容协议的大语言模型答题中间件。接收考试题目（文本 + 图片），调用配置的 LLM 进行推理作答并返回格式化结果。

**版本**: 3.1.0
**语言**: Go 1.22+
**依赖**: 无第三方运行时依赖，仅需任意 OpenAI API 兼容的 LLM 后端。

---

## 2. 快速开始

### 2.1 编译与运行

```bash
# 编译
go build -o ocs-ai .

# 直接运行（使用默认配置）
./ocs-ai
```

启动后输出：

```
OCS AI 答题服务已启动
监听地址: 0.0.0.0:5000
可用题型: 无
模型数量: 0 (启用 0)
```

### 2.2 验证服务

```bash
curl http://localhost:5000/api/health
```

---

## 3. API 接口

### 3.1 `POST /api/answer` — 答题

**请求格式** (`Content-Type: application/json`):

```json
{
  "question": "中国的首都是哪里？",
  "options": ["北京", "上海", "广州", "深圳"],
  "type": 0,
  "images": []
}
```

| 字段 | 类型 | 必需 | 说明 |
|---|---|---|---|
| `question` | string | ✅ | 题目文本，可包含图片 URL（自动提取） |
| `options` | string[] | ❌ | 选项列表，单选题/多选题必需 |
| `type` | int | ❌ | 题型编号：`0`=单选, `1`=多选, `3`=填空, `4`=判断 |
| `images` | string[] | ❌ | OCS 传入的附加图片 URL 列表 |

**成功响应** (`200`):

```json
{
  "success": true,
  "question": "中国的首都是哪里？",
  "answer": "北京",
  "ocs_answer": "北京",
  "type": "single",
  "raw_answer": "北京",
  "model": "gpt-4o",
  "provider": "openai",
  "reasoning_used": false,
  "ai_time": 0.85,
  "total_time": 0.85,
  "usage": {
    "prompt_tokens": 120,
    "completion_tokens": 8,
    "total_tokens": 128
  },
  "ocs_format": [
    "中国的首都是哪里？",
    "北京",
    {
      "ai": true,
      "tags": [
        {
          "text": "自定义模型",
          "title": "使用模型: gpt-4o",
          "color": "green"
        }
      ],
      "model": "gpt-4o",
      "provider": "openai",
      "ai_time": 0.85,
      "total_time": 0.85,
      "usage": {
        "prompt_tokens": 120,
        "completion_tokens": 8,
        "total_tokens": 128
      }
    }
  ]
}
```

| 字段 | 说明 |
|---|---|
| `answer` | 处理后的人类可读答案（匹配选项内容） |
| `ocs_answer` | OCS 系统匹配格式答案 |
| `raw_answer` | AI 原始输出（未处理） |
| `type` | 题型标识：`single`/`multiple`/`judgement`/`completion` |
| `model` | 实际使用的模型名称 |
| `provider` | 模型提供商 |
| `ai_time` / `total_time` | AI 耗时 / 总耗时（秒） |
| `usage` | Token 用量 |
| `ocs_format` | OCS 结构化数据，索引 `[0]`=题目, `[1]`=答案, `[2]`=元信息 |

**错误响应**:

```json
{
  "success": false,
  "error": "错误描述"
}
```

### 3.2 `GET /api/health` — 健康检查

**响应**:

```json
{
  "status": "ok",
  "service": "OCS AI Answerer (Multi-Model)",
  "version": "3.1.0",
  "api_configured": true,
  "model_count": 3,
  "enabled_model_count": 2,
  "ready_question_types": ["single", "multiple", "judgement", "completion"],
  "has_multimodal_model": true,
  "init_error": null
}
```

| 字段 | 说明 |
|---|---|
| `status` | `ok`（可用）或 `error`（未就绪） |
| `api_configured` | 是否至少一种题型有可用模型 |
| `ready_question_types` | 可处理的题型列表 |
| `init_error` | 初始化错误信息（如未配置模型） |

---

## 4. 配置模型

模型配置存储在 `custom_models.json` 文件中。支持手动编辑或通过管理 API 操作。

### 4.1 模型配置结构

```json
{
  "models": {
    "model-001": {
      "name": "GPT-4o",
      "provider": "openai",
      "api_key": "sk-xxxxxxxxxxxxxxxx",
      "base_url": "https://api.openai.com",
      "model_name": "gpt-4o",
      "is_multimodal": true,
      "max_tokens": 500,
      "temperature": 0.1,
      "top_p": 0.95,
      "api_protocol": "chat_completions",
      "enabled": true,
      "is_builtin": false,
      "created_at": "2026-06-29T00:00:00Z",
      "updated_at": "2026-06-29T00:00:00Z"
    }
  },
  "question_type_models": {
    "single":       { "models": ["model-001"] },
    "multiple":     { "models": ["model-001"] },
    "judgement":    { "models": ["model-001"] },
    "completion":   { "models": ["model-001"] },
    "image":        { "models": ["model-001"] }
  },
  "metadata": {
    "builtin_presets_bootstrap_version": "0"
  },
  "version": "1.0",
  "updated_at": "2026-06-29T00:00:00Z"
}
```

### 4.2 模型字段说明

| 字段 | 类型 | 说明 |
|---|---|---|
| `name` | string | 显示名称（任意字符串） |
| `provider` | string | 提供商标识（如 `openai`, `deepseek`, `ollama` 等） |
| `api_key` | string | API 密钥 |
| `base_url` | string | API 基础 URL（如 `https://api.openai.com`） |
| `model_name` | string | API 调用时传的 model 参数（如 `gpt-4o`, `deepseek-chat`） |
| `is_multimodal` | bool | 是否支持图片（多模态）输入 |
| `max_tokens` | int | 最大输出 token 数 |
| `temperature` | float | 采样温度 (0~2) |
| `top_p` | float | 核采样参数 (0~1) |
| `api_protocol` | string | 固定为 `chat_completions` |
| `ds_thinking_mode` | bool | 是否启用 DeepSeek 思考模式（见 4.4） |
| `enabled` | bool | 是否启用 |

### 4.3 DeepSeek 思考模式

当模型的 `ds_thinking_mode` 设为 `true`（或全局环境变量 `DS_THINKING_MODE=true`）时，请求会启用 DeepSeek 思考模式：

- **请求注入**：`"thinking": {"type": "enabled"}` 参数
- **自动移除**：`temperature` 和 `top_p`（思考模式下不兼容）
- **响应提取**：从 `choices[0].message.reasoning_content` 提取思考过程，写入 CSV 日志
- **响应标记**：`reasoning_used: true`，`ocs_format` 中打上"深度思考"标签

```json
{
  "ds_thinking_mode": true
}
```

> ⚠️ 仅 DeepSeek API 支持此参数。其他 LLM 后端开启会导致 400 错误。

### 4.4 题型映射

`question_type_models` 决定了哪些模型用于哪些题型：

| 键 | 题型 | 题型编号 |
|---|---|---|
| `single` | 单选题 | `0` |
| `multiple` | 多选题 | `1` |
| `judgement` | 判断题 | `4` |
| `completion` | 填空题 | `3` |
| `image` | 图片题 | —（自动识别，复用原题型编号） |

- 同一题型可配置多个模型，调用时按顺序**故障转移**（第一个失败则尝试下一个）
- 图片题的 `image` 映射优先于其他题型映射

### 4.4 接入第三方 LLM 示例

**OpenAI**:
```json
{
  "api_key": "sk-xxx",
  "base_url": "https://api.openai.com",
  "model_name": "gpt-4o",
  "is_multimodal": true
}
```

**DeepSeek**:
```json
{
  "api_key": "sk-xxx",
  "base_url": "https://api.deepseek.com",
  "model_name": "deepseek-chat",
  "is_multimodal": false
}
```

**Ollama (本地)**:
```json
{
  "api_key": "ollama",
  "base_url": "http://localhost:11434",
  "model_name": "qwen2.5:7b",
  "is_multimodal": false
}
```

**OpenRouter**:
```json
{
  "api_key": "sk-or-v1-xxx",
  "base_url": "https://openrouter.ai/api",
  "model_name": "openai/gpt-4o",
  "is_multimodal": true
}
```

**兼容 OpenAI API 的任何服务** 均支持，只要提供 `/v1/chat/completions` 端点。

---

## 5. 配置参数

### 5.1 环境变量 (`.env` 文件)

| 变量 | 默认值 | 说明 |
|---|---|---|
| `TEMPERATURE` | `0.1` | AI 采样温度（浮点数） |
| `MAX_TOKENS` | `500` | 最大输出 token 数 (1~8192) |
| `TOP_P` | `0.95` | 核采样参数（浮点数） |
| `TIMEOUT` | `1200` | API 请求超时（秒） |
| `MAX_RETRIES` | `3` | API 调用最大重试次数 |
| `HOST` | `0.0.0.0` | 监听地址 |
| `PORT` | `5000` | 监听端口 |
| `HTTP_PROXY` | — | HTTP 代理 |
| `HTTPS_PROXY` | — | HTTPS 代理 |
| `DEBUG` | `false` | 调试模式 |
| `DS_THINKING_MODE` | `false` | 全局启用 DeepSeek 思考模式 |
| `CSV_LOG_FILE` | `ocs_answers_log.csv` | 答题日志文件路径 |
| `LOG_LEVEL` | `INFO` | 日志级别 |

`.env` 文件示例：

```env
PORT=8080
TEMPERATURE=0.2
MAX_TOKENS=1024
HTTP_PROXY=http://127.0.0.1:7890
```

> **注意**: `.env` 文件不会覆盖已有的环境变量，环境变量优先级更高。

---

## 6. 题型说明

### 6.1 单选题 (`type: 0`)

要求 AI 从选项中选择**一个**最正确答案。AI 输出将匹配到选项文本。

```json
{
  "question": "HTML 的全称是什么？",
  "options": ["HyperText Markup Language", "HighText Machine Language", "HyperTool Multi Language"],
  "type": 0
}
```

### 6.2 多选题 (`type: 1`)

要求 AI 选择**所有**正确答案，多个答案之间用 `#` 分隔。

```json
{
  "question": "以下哪些是编程语言？",
  "options": ["Python", "Java", "HTML", "CSS"],
  "type": 1
}
```

返回 `answer` 示例: `"Python#Java"`

### 6.3 判断题 (`type: 4`)

要求 AI 从给定选项中选择判断结果（如 `正确`/`错误`、`对`/`错`）。

```json
{
  "question": "地球是太阳系中最大的行星。",
  "options": ["正确", "错误"],
  "type": 4
}
```

### 6.4 填空题 (`type: 3`)

AI 直接输出答案内容。多空题用 `#` 分隔。

```json
{
  "question": "水的化学式是___，由___和___组成。",
  "type": 3
}
```

返回 `answer` 示例: `"H2O#氢#氧"`

---

## 7. 图片题支持

题目中的图片 URL 会自动提取并发送给多模态模型。

### 7.1 图片来源

1. **题目文本中**的图片 URL（自动正则提取 `jpg|jpeg|png|gif|bmp|webp` 链接）
2. **选项中**的图片 URL
3. **`images` 数组**中 OCS 传入的图片 URL

### 7.2 配置图片题模型

```json
{
  "question_type_models": {
    "image": { "models": ["gpt-4o", "gemini-2.0-flash"] }
  }
}
```

- 图片题的模型必须设置 `"is_multimodal": true`
- 图片题**优先使用 `image` 映射**中的模型，不存在时才使用原题型的模型
- 系统会自动过滤图标类无关图片（如 `video.png`、`play.png` 等）

---

## 8. 日志

答题记录自动写入 CSV 文件（默认 `ocs_answers_log.csv`）：

| 列名 | 说明 |
|---|---|
| 时间戳 | 答题时间 |
| 题型 | 单选/多选/判断/填空 |
| 题目 | 原始题目 |
| 选项 | 选项（`\|` 分隔） |
| 原始回答 | AI 原始输出 |
| 思考过程 | （保留字段，始终为空） |
| 处理后答案 | 经系统处理后的最终答案 |
| AI耗时(秒) | AI 调用耗时 |
| 总耗时(秒) | 总耗时 |
| 模型 | 使用的模型名称 |
| 思考模式 | （保留字段，始终为 `false`） |
| 输入Token | Prompt token 数 |
| 输出Token | Completion token 数 |
| 总Token | 总 token 数 |
| 费用(元) | （保留字段） |
| 提供商 | 模型提供商 |

---

## 9. 故障转移机制

1. 按题型映射中模型的**定义顺序**依次尝试
2. 当前模型调用失败时，**等待 1 秒**后自动切换到下一个
3. 每个模型最多尝试 **3 次**（可通过 `MAX_RETRIES` 调整）
4. 全部失败时返回错误信息

---

## 10. 部署建议

### Docker 部署

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o ocs-ai .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/ocs-ai .
COPY custom_models.json .
EXPOSE 5000
CMD ["./ocs-ai"]
```

### Systemd 服务

```ini
[Unit]
Description=OCS AI Answerer
After=network.target

[Service]
Type=simple
User=nobody
WorkingDirectory=/opt/ocs-ai
ExecStart=/opt/ocs-ai/ocs-ai
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### 守护进程运行

```bash
# 使用 nohup
nohup ./ocs-ai > ocs-ai.log 2>&1 &

# 或使用 tmux/screen
tmux new -s ocs-ai ./ocs-ai
```

---

## 11. 常见问题

**Q: 启动后提示"未配置任何模型"？**
A: 编辑 `custom_models.json` 添加模型配置，并确保至少为一种题型设置了映射。

**Q: 如何添加多个模型实现故障转移？**
A: 在 `question_type_models` 对应题型的 `models` 数组中添加多个模型 ID：

```json
"single": { "models": ["model-primary", "model-backup"] }
```

**Q: 图片题没走期望的模型？**
A: 检查 `image` 映射是否配置，且对应模型的 `is_multimodal` 是否为 `true`。图片题优先使用 `image` 映射中的模型。

**Q: 支持流式输出吗？**
A: 不支持，当前仅支持非流式 (`stream: false`) 调用。

**Q: 修改 `custom_models.json` 后需要重启吗？**
A: 不需要，服务在每次请求时均读取最新配置（但建议重启以确保一致性）。
