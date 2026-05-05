#!/bin/bash

# 测试告警规则导入功能

# 1. 创建测试 YAML 文件
cat > /tmp/test_rule.yaml <<'EOF'
- name: 测试规则
  description: "这是一个测试规则"
  assetgroupid: 0
  rulegroupid: 0
  datasourceid: 1
  datasourceids: '[1]'
  expr: 'up'
  queryexpr: 'up'
  conditions: '[{"logic": "AND", "value": 0, "operator": "=="}]'
  evalinterval: 15
  duration: 60s
  severity: warning
  labels: ""
  annotations: '{"title":"测试告警","description":"这是测试"}'
  enabled: false
  notifyonresolve: true
EOF

echo "测试文件已创建: /tmp/test_rule.yaml"
echo "文件内容:"
cat /tmp/test_rule.yaml
echo ""
echo "---"
echo ""

# 2. 使用 curl 测试导入
echo "开始测试导入..."
curl -X POST http://localhost:9876/api/v1/alert/rules/import \
  -H "Authorization: Bearer $(cat ~/.opshub_token 2>/dev/null || echo 'YOUR_TOKEN_HERE')" \
  -F "file=@/tmp/test_rule.yaml" \
  -v

echo ""
echo "---"
echo "查看最近的日志:"
tail -50 logs/app.log | grep -E "导入|规则|test_rule|测试规则"
