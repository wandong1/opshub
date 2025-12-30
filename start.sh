#!/bin/bash

# 快速启动脚本

echo "====================================="
echo "  OpsHub RBAC 系统启动脚本"
echo "====================================="
echo ""

# 检查并启动后端
echo "1. 检查后端服务..."
if [ ! -f "./bin/opshub" ]; then
    echo "   后端未编译，正在编译..."
    go build -o bin/opshub main.go
fi

echo "   启动后端服务..."
./bin/opshub server &
BACKEND_PID=$!
echo "   后端服务已启动 (PID: $BACKEND_PID)"
echo "   地址: http://localhost:9876"
echo ""

# 等待后端启动
sleep 2

# 启动前端
echo "2. 启动前端服务..."
cd web
echo "   安装依赖（如果需要）..."
if [ ! -d "node_modules" ]; then
    npm install
fi

echo "   启动前端开发服务器..."
npm run dev &
FRONTEND_PID=$!
echo "   前端服务已启动 (PID: $FRONTEND_PID)"
echo "   地址: http://localhost:5173"
echo ""

echo "====================================="
echo "  系统启动完成！"
echo "====================================="
echo ""
echo "后端地址: http://localhost:9876"
echo "前端地址: http://localhost:5173"
echo "API文档: http://localhost:9876/swagger/index.html"
echo ""
echo "按 Ctrl+C 停止所有服务"
echo ""

# 等待用户中断
trap "echo ''; echo '正在停止服务...'; kill $BACKEND_PID $FRONTEND_PID 2>/dev/null; exit 0" INT TERM

wait
