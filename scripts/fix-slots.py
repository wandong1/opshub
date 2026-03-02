#!/usr/bin/env python3
"""Fix Chinese slot names and remove orphaned #header templates in converted Vue files."""
import re
import glob
import os

KUBE_DIR = "/home/wandong/opus_project/src/opshub/web/src/views/kubernetes"

# Mapping of Chinese slot names to English
SLOT_NAME_MAP = {
    'col_节点名称': 'nodeName',
    'col_状态': 'status',
    'col_角色': 'role',
    'col_Kubelet版本': 'kubeletVersion',
    'col_标签': 'labels',
    'col_运行时间': 'uptime',
    'col_CPU': 'cpu',
    'col_内存': 'memory',
    'col_Pod数量': 'podCount',
    'col_调度': 'schedulable',
    'col_污点': 'taints',
    'col_访问模式': 'accessModes',
    'col_允许卷扩展': 'allowExpansion',
    'col_回收策略': 'reclaimPolicy',
    'col_绑定模式': 'bindingMode',
    'col_操作': 'actions',
    'col_类型': 'type',
    'col_名称': 'name',
    'col_命名空间': 'namespace',
    'col_集群IP': 'clusterIP',
    'col_端口': 'ports',
    'col_选择器': 'selector',
    'col_创建时间': 'createdAt',
    'col_年龄': 'age',
    'col_容量': 'capacity',
    'col_存储类': 'storageClass',
    'col_绑定PVC': 'boundPVC',
    'col_绑定PV': 'boundPV',
    'col_请求容量': 'requestCapacity',
    'col_描述': 'description',
    'col_值': 'value',
    'col_数据': 'data',
    'col_键': 'key',
    'col_规则': 'rules',
    'col_主机': 'host',
    'col_路径': 'path',
    'col_后端': 'backend',
    'col_TLS': 'tls',
    'col_副本': 'replicas',
    'col_就绪': 'ready',
    'col_镜像': 'image',
    'col_重启次数': 'restarts',
    'col_节点': 'node',
    'col_IP': 'ip',
    'col_资源': 'resources',
    'col_条件': 'conditions',
    'col_事件': 'events',
    'col_用户名': 'username',
    'col_角色绑定': 'roleBinding',
    'col_服务账号': 'serviceAccount',
    'col_密钥': 'secret',
    'col_目标': 'target',
    'col_当前': 'current',
    'col_最小': 'min',
    'col_最大': 'max',
    'col_期望': 'desired',
    'col_可用': 'available',
    'col_更新': 'updated',
    'col_策略': 'strategy',
    'col_调度策略': 'scheduleStrategy',
    'col_完成': 'completions',
    'col_持续时间': 'duration',
    'col_上次调度': 'lastSchedule',
    'col_暂停': 'suspend',
    'col_活跃': 'active',
    'col_成功': 'succeeded',
    'col_失败': 'failed',
}

def fix_file(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()

    original = content

    # 1. Remove orphaned #header templates (they don't work in Arco's column-based table)
    # Pattern: <template #header>..content..</template> inside a-table
    content = re.sub(
        r'\s*<template #header>\s*\n\s*<span class="header-with-icon">\s*\n\s*<[^>]+/>\s*\n\s*[^<]+\n\s*</span>\s*\n\s*</template>',
        '',
        content
    )

    # Also simpler header patterns
    content = re.sub(
        r'\s*<template #header>\s*\n.*?\n\s*</template>',
        '',
        content,
        flags=re.DOTALL
    )

    # 2. Replace Chinese slot names with English equivalents
    for cn_name, en_name in SLOT_NAME_MAP.items():
        content = content.replace(f"slotName: '{cn_name}'", f"slotName: '{en_name}'")
        content = content.replace(f'#{cn_name}=', f'#{en_name}=')
        content = content.replace(f'#{cn_name}"', f'#{en_name}"')

    # 3. Also fix any remaining col_XXX patterns not in the map
    # Find all col_XXX patterns and create simple English names
    remaining = re.findall(r"col_([^\s'\"=]+)", content)
    for name in set(remaining):
        if name not in [v.replace('col_', '') for v in SLOT_NAME_MAP.keys()]:
            # Create a simple slug
            en_name = 'col_' + str(hash(name) % 10000)
            # Don't replace if it's already English
            if not name.isascii():
                content = content.replace(f"col_{name}", en_name)

    if content != original:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(content)
        return True
    return False

# Process all files
vue_files = glob.glob(os.path.join(KUBE_DIR, '**/*.vue'), recursive=True)
fixed = 0
for filepath in sorted(vue_files):
    if fix_file(filepath):
        print(f"Fixed: {filepath}")
        fixed += 1

print(f"\nFixed {fixed} files.")
