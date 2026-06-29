#!/bin/bash

HOST="${1:-localhost}"
PORT="${2:-3000}"
BASE="http://${HOST}:${PORT}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

pass=0
fail=0

check() {
    local label="$1"
    local method="$2"
    local url="$3"
    local body="$4"
    local expected_code="$5"
    local expected_in="$6"

    if [ -n "$body" ]; then
        resp=$(curl -s -w "\n%{http_code}" -X "$method" "$url" \
            -H "Content-Type: application/json" \
            -d "$body" 2>&1)
    else
        resp=$(curl -s -w "\n%{http_code}" -X "$method" "$url" 2>&1)
    fi

    http_code=$(echo "$resp" | tail -1)
    body_only=$(echo "$resp" | sed '$d')

    if [ "$http_code" = "$expected_code" ]; then
        if [ -n "$expected_in" ]; then
            if echo "$body_only" | grep -q "$expected_in"; then
                echo -e "  ${GREEN}PASS${NC}  [$http_code] $label"
                pass=$((pass + 1))
            else
                echo -e "  ${RED}FAIL${NC}  [$http_code] $label — 响应体中未找到 '$expected_in'"
                echo "        响应: $(echo "$body_only" | head -c 200)"
                fail=$((fail + 1))
            fi
        else
            echo -e "  ${GREEN}PASS${NC}  [$http_code] $label"
            pass=$((pass + 1))
        fi
    else
        echo -e "  ${RED}FAIL${NC}  [$http_code] $label — 期望 $expected_code"
        echo "        响应: $(echo "$body_only" | head -c 200)"
        fail=$((fail + 1))
    fi
}

echo ""
echo -e "${CYAN}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║     OCS AI Answerer 端点可用性测试               ║${NC}"
echo -e "${CYAN}║     ${BASE}                        ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════╝${NC}"
echo ""

# ----- 健康检查 -----
echo -e "${YELLOW}[1] /api/health 健康检查${NC}"
check "health 返回 200"             GET  "$BASE/api/health" ""   200 '"status":"ok"'
check "health 包含 service 字段"    GET  "$BASE/api/health" ""   200 '"service"'
check "health 包含 version 字段"    GET  "$BASE/api/health" ""   200 '"version"'
check "health 包含 model_count"     GET  "$BASE/api/health" ""   200 '"model_count"'

# ----- /query 方法校验 -----
echo ""
echo -e "${YELLOW}[2] /query 方法校验${NC}"
check "GET /query → 405"            GET     "$BASE/query" "" 405 ""

# ----- /query 请求体校验 -----
echo ""
echo -e "${YELLOW}[3] /query 请求体校验${NC}"
check "空请求体 → 400"              POST    "$BASE/query" ''                                                  400 "success\":false"
check "非法 JSON → 400"             POST    "$BASE/query" 'not json'                                          400 "success\":false"
check "空题目 → 400"                POST    "$BASE/query" '{"question":"  "}'                                 400 "题目不能为空"
check "缺少 question → 400"         POST    "$BASE/query" '{}'                                                400 "success\":false"

# ----- /query 单选题 -----
echo ""
echo -e "${YELLOW}[4] /query 单选题 (type=0)${NC}"
check "单选题 → 200"                POST    "$BASE/query" \
    '{"question":"中国的首都是哪里？","options":["北京","上海","广州","深圳"],"type":0}' \
    200 "success\":true"
check "单选题 返回 answer 字段"      POST    "$BASE/query" \
    '{"question":"中国的首都是哪里？","options":["北京","上海","广州","深圳"],"type":0}' \
    200 '"answer"'
check "单选题 返回 ocs_format"       POST    "$BASE/query" \
    '{"question":"中国的首都是哪里？","options":["北京","上海","广州","深圳"],"type":0}' \
    200 '"ocs_format"'
check "单选题 返回 usage"            POST    "$BASE/query" \
    '{"question":"中国的首都是哪里？","options":["北京","上海","广州","深圳"],"type":0}' \
    200 '"total_tokens"'
check "单选题 返回 type=single"      POST    "$BASE/query" \
    '{"question":"中国的首都是哪里？","options":["北京","上海","广州","深圳"],"type":0}' \
    200 '"type":"single"'

# ----- /query 多选题 -----
echo ""
echo -e "${YELLOW}[5] /query 多选题 (type=1)${NC}"
check "多选题 → 200"                POST    "$BASE/query" \
    '{"question":"以下哪些是编程语言？","options":["Python","筷子","Java","米饭"],"type":1}' \
    200 "success\":true"
check "多选题 返回 type=multiple"    POST    "$BASE/query" \
    '{"question":"以下哪些是编程语言？","options":["Python","筷子","Java","米饭"],"type":1}' \
    200 '"type":"multiple"'

# ----- /query 判断题 -----
echo ""
echo -e "${YELLOW}[6] /query 判断题 (type=4)${NC}"
check "判断题 → 200"                POST    "$BASE/query" \
    '{"question":"地球是圆的。","options":["正确","错误"],"type":4}' \
    200 "success\":true"
check "判断题 返回 type=judgement"   POST    "$BASE/query" \
    '{"question":"地球是圆的。","options":["正确","错误"],"type":4}' \
    200 '"type":"judgement"'

# ----- /query 填空题 -----
echo ""
echo -e "${YELLOW}[7] /query 填空题 (type=3)${NC}"
check "填空题 → 200"                POST    "$BASE/query" \
    '{"question":"水的化学式是____。","options":[],"type":3}' \
    200 "success\":true"
check "填空题 返回 type=completion"  POST    "$BASE/query" \
    '{"question":"水的化学式是____。","options":[],"type":3}' \
    200 '"type":"completion"'

# ============================================================
echo ""
echo -e "${CYAN}══════════════════════════════════════════════════${NC}"
echo -e "  通过: ${GREEN}${pass}${NC}  |  失败: ${RED}${fail}${NC}"
echo -e "${CYAN}══════════════════════════════════════════════════${NC}"

[ "$fail" -gt 0 ] && exit 1
exit 0
