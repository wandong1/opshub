#!/bin/bash

echo "=========================================="
echo "实时监控告警规则导入日志"
echo "=========================================="
echo ""
echo "请在浏览器中执行导入操作..."
echo ""

# 实时监控日志，过滤导入相关内容
tail -f logs/app.log | grep --line-buffered -E "导入|规则|import|Import|IMPORT|解析|Parse|创建|更新|失败|成功"
