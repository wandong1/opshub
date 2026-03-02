#!/bin/bash
# Kafka 消费组消费监听脚本
# 目标: 192.168.0.158:9094 / Topic: test / 从最早开始 / 免密

BROKER="192.168.0.158:9094"
TOPIC="test"
GROUP="opshub-monitor-$$"

echo "========================================="
echo " Kafka 消费组消费监听"
echo " Broker:  $BROKER"
echo " Topic:   $TOPIC"
echo " Group:   $GROUP"
echo " 起始位置: earliest (--from-beginning)"
echo "========================================="
echo ""

# 检测可用的 Kafka 客户端工具
if command -v kcat &>/dev/null; then
    echo "[使用 kcat 消费]"
    echo "按 Ctrl+C 停止"
    echo "-----------------------------------------"
    kcat -b "$BROKER" -t "$TOPIC" -C -G "$GROUP" -o beginning \
         -f '[%T] P:%p O:%o | Key: %k | Value: %s\n'

elif command -v kafkacat &>/dev/null; then
    echo "[使用 kafkacat 消费]"
    echo "按 Ctrl+C 停止"
    echo "-----------------------------------------"
    kafkacat -b "$BROKER" -t "$TOPIC" -C -G "$GROUP" -o beginning \
             -f '[%T] P:%p O:%o | Key: %k | Value: %s\n'

elif command -v kafka-console-consumer.sh &>/dev/null; then
    echo "[使用 kafka-console-consumer.sh 消费]"
    echo "按 Ctrl+C 停止"
    echo "-----------------------------------------"
    kafka-console-consumer.sh \
        --bootstrap-server "$BROKER" \
        --topic "$TOPIC" \
        --group "$GROUP" \
        --from-beginning \
        --property print.timestamp=true \
        --property print.key=true \
        --property print.offset=true \
        --property print.partition=true \
        --property print.headers=true \
        --property key.separator=" | "

elif command -v kafka-console-consumer &>/dev/null; then
    echo "[使用 kafka-console-consumer 消费]"
    echo "按 Ctrl+C 停止"
    echo "-----------------------------------------"
    kafka-console-consumer \
        --bootstrap-server "$BROKER" \
        --topic "$TOPIC" \
        --group "$GROUP" \
        --from-beginning \
        --property print.timestamp=true \
        --property print.key=true \
        --property print.offset=true \
        --property print.partition=true \
        --property print.headers=true \
        --property key.separator=" | "

elif command -v docker &>/dev/null; then
    echo "[本地未找到 Kafka 工具，使用 Docker 容器消费]"
    echo "按 Ctrl+C 停止"
    echo "-----------------------------------------"
    docker run --rm --network host \
        bitnami/kafka:latest \
        kafka-console-consumer.sh \
            --bootstrap-server "$BROKER" \
            --topic "$TOPIC" \
            --group "$GROUP" \
            --from-beginning \
            --property print.timestamp=true \
            --property print.key=true \
            --property print.offset=true \
            --property print.partition=true \
            --property key.separator=" | "
else
    echo "[错误] 未找到可用的 Kafka 客户端工具"
    echo ""
    echo "请安装以下任一工具："
    echo "  1. kcat:                    apt install kcat / brew install kcat"
    echo "  2. kafka-console-consumer:  从 https://kafka.apache.org/downloads 下载"
    echo "  3. docker:                  脚本会自动使用 bitnami/kafka 镜像"
    exit 1
fi
