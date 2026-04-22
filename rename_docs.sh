#!/bin/bash

# 批量重命名 docs 目录下的 markdown 文件，添加中文名称

cd /Users/Zhuanz/golang_project/src/opshub/docs

# 核心文档
[ -f "deployment.md" ] && mv "deployment.md" "部署指南(deployment).md"
[ -f "cache-design.md" ] && mv "cache-design.md" "缓存设计方案(cache-design).md"
[ -f "cache-integration.md" ] && mv "cache-integration.md" "缓存集成(cache-integration).md"
[ -f "cache-summary.md" ] && mv "cache-summary.md" "缓存总结(cache-summary).md"
[ -f "cache-implementation-complete.md" ] && mv "cache-implementation-complete.md" "缓存实现完成(cache-implementation-complete).md"
[ -f "CACHE_SYSTEM_STATUS.md" ] && mv "CACHE_SYSTEM_STATUS.md" "缓存系统状态(CACHE_SYSTEM_STATUS).md"
[ -f "CACHE_INTEGRATION_CHECKLIST.md" ] && mv "CACHE_INTEGRATION_CHECKLIST.md" "缓存集成检查清单(CACHE_INTEGRATION_CHECKLIST).md"
[ -f "cache-integration-fixed.md" ] && mv "cache-integration-fixed.md" "缓存集成修复(cache-integration-fixed).md"
[ -f "cache-quick-integration.md" ] && mv "cache-quick-integration.md" "缓存快速集成(cache-quick-integration).md"
[ -f "CACHE_INTEGRATION_STEPS.md" ] && mv "CACHE_INTEGRATION_STEPS.md" "缓存集成步骤(CACHE_INTEGRATION_STEPS).md"
[ -f "CACHE_FINAL_STATUS.md" ] && mv "CACHE_FINAL_STATUS.md" "缓存最终状态(CACHE_FINAL_STATUS).md"
[ -f "CACHE_INTEGRATION_COMPLETE.md" ] && mv "CACHE_INTEGRATION_COMPLETE.md" "缓存集成完成(CACHE_INTEGRATION_COMPLETE).md"

# Agent 相关
[ -f "AGENT_OFFLINE_STATUS_FIX.md" ] && mv "AGENT_OFFLINE_STATUS_FIX.md" "Agent离线状态修复(AGENT_OFFLINE_STATUS_FIX).md"
[ -f "AGENT_STATUS_POLLING_FIX.md" ] && mv "AGENT_STATUS_POLLING_FIX.md" "Agent状态轮询修复(AGENT_STATUS_POLLING_FIX).md"
[ -f "agent-heartbeat-optimization.md" ] && mv "agent-heartbeat-optimization.md" "Agent心跳优化(agent-heartbeat-optimization).md"
[ -f "agent-heartbeat-quickstart.md" ] && mv "agent-heartbeat-quickstart.md" "Agent心跳快速开始(agent-heartbeat-quickstart).md"
[ -f "agent-heartbeat-summary.md" ] && mv "agent-heartbeat-summary.md" "Agent心跳总结(agent-heartbeat-summary).md"
[ -f "agent-heartbeat-checklist.md" ] && mv "agent-heartbeat-checklist.md" "Agent心跳检查清单(agent-heartbeat-checklist).md"
[ -f "agent-heartbeat-final-report.md" ] && mv "agent-heartbeat-final-report.md" "Agent心跳最终报告(agent-heartbeat-final-report).md"
[ -f "agent_ssl.md" ] && mv "agent_ssl.md" "Agent SSL配置(agent_ssl).md"
[ -f "agent-no-systemd-fix.md" ] && mv "agent-no-systemd-fix.md" "Agent无systemd修复(agent-no-systemd-fix).md"
[ -f "agent-tls-certificate-fix.md" ] && mv "agent-tls-certificate-fix.md" "Agent TLS证书修复(agent-tls-certificate-fix).md"
[ -f "agent-fix-summary.md" ] && mv "agent-fix-summary.md" "Agent修复总结(agent-fix-summary).md"
[ -f "agent-nat-solution.md" ] && mv "agent-nat-solution.md" "Agent NAT解决方案(agent-nat-solution).md"
[ -f "agent-nat-quickstart.md" ] && mv "agent-nat-quickstart.md" "Agent NAT快速开始(agent-nat-quickstart).md"
[ -f "agent-nat-implementation-summary.md" ] && mv "agent-nat-implementation-summary.md" "Agent NAT实现总结(agent-nat-implementation-summary).md"
[ -f "AGENT_BUILD_COMPLETED.md" ] && mv "AGENT_BUILD_COMPLETED.md" "Agent构建完成(AGENT_BUILD_COMPLETED).md"

# 探测相关
[ -f "probe-metrics-enhancement.md" ] && mv "probe-metrics-enhancement.md" "探测指标增强(probe-metrics-enhancement).md"
[ -f "probe-metrics.md" ] && mv "probe-metrics.md" "探测指标(probe-metrics).md"
[ -f "probe-agent-mode-fix.md" ] && mv "probe-agent-mode-fix.md" "探测Agent模式修复(probe-agent-mode-fix).md"
[ -f "probe-agent-logging.md" ] && mv "probe-agent-logging.md" "探测Agent日志(probe-agent-logging).md"
[ -f "agent-app-probe-implementation.md" ] && mv "agent-app-probe-implementation.md" "Agent应用探测实现(agent-app-probe-implementation).md"
[ -f "agent-native-probe-design.md" ] && mv "agent-native-probe-design.md" "Agent原生探测设计(agent-native-probe-design).md"
[ -f "agent-probe-implementation-status.md" ] && mv "agent-probe-implementation-status.md" "Agent探测实现状态(agent-probe-implementation-status).md"
[ -f "AGENT_PROBE_SUMMARY.md" ] && mv "AGENT_PROBE_SUMMARY.md" "Agent探测总结(AGENT_PROBE_SUMMARY).md"
[ -f "AGENT_PROBE_COMPLETED.md" ] && mv "AGENT_PROBE_COMPLETED.md" "Agent探测完成(AGENT_PROBE_COMPLETED).md"

# 站点管理
[ -f "website-management-implementation.md" ] && mv "website-management-implementation.md" "站点管理实现(website-management-implementation).md"
[ -f "website-proxy-integration-status.md" ] && mv "website-proxy-integration-status.md" "站点代理集成状态(website-proxy-integration-status).md"
[ -f "website-proxy-test-checklist.md" ] && mv "website-proxy-test-checklist.md" "站点代理测试清单(website-proxy-test-checklist).md"
[ -f "website-proxy-fix-summary.md" ] && mv "website-proxy-fix-summary.md" "站点代理修复总结(website-proxy-fix-summary).md"
[ -f "website-bugs-fix.md" ] && mv "website-bugs-fix.md" "站点Bug修复(website-bugs-fix).md"
[ -f "website-bugs-fix-summary.md" ] && mv "website-bugs-fix-summary.md" "站点Bug修复总结(website-bugs-fix-summary).md"
[ -f "website-bugs-fix-report.md" ] && mv "website-bugs-fix-report.md" "站点Bug修复报告(website-bugs-fix-report).md"
[ -f "website-display-issue.md" ] && mv "website-display-issue.md" "站点显示问题(website-display-issue).md"
[ -f "website-bugs-final-fix.md" ] && mv "website-bugs-final-fix.md" "站点Bug最终修复(website-bugs-final-fix).md"
[ -f "website-enhancements.md" ] && mv "website-enhancements.md" "站点功能增强(website-enhancements).md"
[ -f "website-ui-enhancement.md" ] && mv "website-ui-enhancement.md" "站点UI增强(website-ui-enhancement).md"
[ -f "website-password-copy-fix.md" ] && mv "website-password-copy-fix.md" "站点密码复制修复(website-password-copy-fix).md"
[ -f "website-bugs-fix-final.md" ] && mv "website-bugs-fix-final.md" "站点Bug最终修复2(website-bugs-fix-final).md"
[ -f "website-proxy-static-resources-fix.md" ] && mv "website-proxy-static-resources-fix.md" "站点代理静态资源修复(website-proxy-static-resources-fix).md"
[ -f "website-proxy-architecture-redesign.md" ] && mv "website-proxy-architecture-redesign.md" "站点代理架构重设计(website-proxy-architecture-redesign).md"

# 巡检相关
[ -f "inspection-implementation-summary.md" ] && mv "inspection-implementation-summary.md" "巡检实现总结(inspection-implementation-summary).md"
[ -f "inspection-fix-checklist.md" ] && mv "inspection-fix-checklist.md" "巡检修复检查清单(inspection-fix-checklist).md"
[ -f "inspection-page-optimization.md" ] && mv "inspection-page-optimization.md" "巡检页面优化(inspection-page-optimization).md"
[ -f "inspection-bug-fixes.md" ] && mv "inspection-bug-fixes.md" "巡检Bug修复(inspection-bug-fixes).md"
[ -f "inspection-additional-fixes.md" ] && mv "inspection-additional-fixes.md" "巡检额外修复(inspection-additional-fixes).md"
[ -f "inspection-test-checklist.md" ] && mv "inspection-test-checklist.md" "巡检测试清单(inspection-test-checklist).md"
[ -f "inspection-host-selector-fix.md" ] && mv "inspection-host-selector-fix.md" "巡检主机选择器修复(inspection-host-selector-fix).md"
[ -f "inspection-data-save-fix.md" ] && mv "inspection-data-save-fix.md" "巡检数据保存修复(inspection-data-save-fix).md"
[ -f "inspection-json-field-mismatch-fix.md" ] && mv "inspection-json-field-mismatch-fix.md" "巡检JSON字段不匹配修复(inspection-json-field-mismatch-fix).md"
[ -f "srehub-inspect-metrics.md" ] && mv "srehub-inspect-metrics.md" "SREHub巡检指标(srehub-inspect-metrics).md"

# 告警相关
[ -f "ALERT_SILENCE_IMPLEMENTATION.md" ] && mv "ALERT_SILENCE_IMPLEMENTATION.md" "告警静默实现(ALERT_SILENCE_IMPLEMENTATION).md"
[ -f "ALERT_AGENT_PROXY_QUICKSTART.md" ] && mv "ALERT_AGENT_PROXY_QUICKSTART.md" "告警Agent代理快速开始(ALERT_AGENT_PROXY_QUICKSTART).md"
[ -f "ALERT_DATASOURCE_AGENT_PROXY_IMPLEMENTATION.md" ] && mv "ALERT_DATASOURCE_AGENT_PROXY_IMPLEMENTATION.md" "告警数据源Agent代理实现(ALERT_DATASOURCE_AGENT_PROXY_IMPLEMENTATION).md"
[ -f "bugfix-silence-rule-inheritance.md" ] && mv "bugfix-silence-rule-inheritance.md" "静默规则继承Bug修复(bugfix-silence-rule-inheritance).md"
[ -f "feature-silence-rules-management.md" ] && mv "feature-silence-rules-management.md" "静默规则管理功能(feature-silence-rules-management).md"

# 数据源相关
[ -f "DATASOURCE_AGENT_PROXY_FIXES.md" ] && mv "DATASOURCE_AGENT_PROXY_FIXES.md" "数据源Agent代理修复(DATASOURCE_AGENT_PROXY_FIXES).md"
[ -f "AGENT_DATASOURCE_WORKFLOW_FIX.md" ] && mv "AGENT_DATASOURCE_WORKFLOW_FIX.md" "Agent数据源工作流修复(AGENT_DATASOURCE_WORKFLOW_FIX).md"
[ -f "AGENT_RELATION_BUG_FIX.md" ] && mv "AGENT_RELATION_BUG_FIX.md" "Agent关联Bug修复(AGENT_RELATION_BUG_FIX).md"
[ -f "DATASOURCE_DUAL_MODE_FIX.md" ] && mv "DATASOURCE_DUAL_MODE_FIX.md" "数据源双模式修复(DATASOURCE_DUAL_MODE_FIX).md"
[ -f "AGENT_DATASOURCE_UX_OPTIMIZATION.md" ] && mv "AGENT_DATASOURCE_UX_OPTIMIZATION.md" "Agent数据源UX优化(AGENT_DATASOURCE_UX_OPTIMIZATION).md"
[ -f "DATASOURCE_AGENT_TEST_AND_PROXY.md" ] && mv "DATASOURCE_AGENT_TEST_AND_PROXY.md" "数据源Agent测试和代理(DATASOURCE_AGENT_TEST_AND_PROXY).md"
[ -f "DATASOURCE_UNIFIED_DESIGN.md" ] && mv "DATASOURCE_UNIFIED_DESIGN.md" "数据源统一设计(DATASOURCE_UNIFIED_DESIGN).md"

# 代理相关
[ -f "AGENT_PROXY_FORWARDING_FIX.md" ] && mv "AGENT_PROXY_FORWARDING_FIX.md" "Agent代理转发修复(AGENT_PROXY_FORWARDING_FIX).md"
[ -f "AGENT_PROXY_MECHANISM.md" ] && mv "AGENT_PROXY_MECHANISM.md" "Agent代理机制(AGENT_PROXY_MECHANISM).md"
[ -f "AGENT_PROXY_IMPLEMENTATION.md" ] && mv "AGENT_PROXY_IMPLEMENTATION.md" "Agent代理实现(AGENT_PROXY_IMPLEMENTATION).md"

# 工作日志
[ -f "WORK_SUMMARY_2026-03-03.md" ] && mv "WORK_SUMMARY_2026-03-03.md" "工作总结2026-03-03(WORK_SUMMARY_2026-03-03).md"
[ -f "WORK_SUMMARY_FINAL_2026-03-03.md" ] && mv "WORK_SUMMARY_FINAL_2026-03-03.md" "最终工作总结2026-03-03(WORK_SUMMARY_FINAL_2026-03-03).md"
[ -f "work-log-2026-04.md" ] && mv "work-log-2026-04.md" "工作日志2026-04(work-log-2026-04).md"
[ -f "work-log-2026-04-07.md" ] && mv "work-log-2026-04-07.md" "工作日志2026-04-07(work-log-2026-04-07).md"
[ -f "final-summary-2026-04-07.md" ] && mv "final-summary-2026-04-07.md" "最终总结2026-04-07(final-summary-2026-04-07).md"

# 其他
[ -f "BUG_FIX_REPORT.md" ] && mv "BUG_FIX_REPORT.md" "Bug修复报告(BUG_FIX_REPORT).md"
[ -f "IMPLEMENTATION_SUMMARY.md" ] && mv "IMPLEMENTATION_SUMMARY.md" "实现总结(IMPLEMENTATION_SUMMARY).md"
[ -f "COMPLETION_REPORT.md" ] && mv "COMPLETION_REPORT.md" "完成报告(COMPLETION_REPORT).md"
[ -f "REDIS_PASSWORD_FIX.md" ] && mv "REDIS_PASSWORD_FIX.md" "Redis密码修复(REDIS_PASSWORD_FIX).md"
[ -f "record1.md" ] && mv "record1.md" "项目记录(record1).md"
[ -f "api-key-implementation.md" ] && mv "api-key-implementation.md" "API密钥实现(api-key-implementation).md"
[ -f "api-key-testing-guide.md" ] && mv "api-key-testing-guide.md" "API密钥测试指南(api-key-testing-guide).md"
[ -f "api-key-verification-report.md" ] && mv "api-key-verification-report.md" "API密钥验证报告(api-key-verification-report).md"

# 处理 plugin-development 子目录
if [ -d "plugin-development" ]; then
    cd plugin-development
    [ -f "README.md" ] && mv "README.md" "插件开发文档(README).md"
    [ -f "quick-start.md" ] && mv "quick-start.md" "快速开始(quick-start).md"
    [ -f "development-guide.md" ] && mv "development-guide.md" "开发指南(development-guide).md"
    [ -f "api-reference.md" ] && mv "api-reference.md" "API参考(api-reference).md"
    [ -f "advanced-topics.md" ] && mv "advanced-topics.md" "高级主题(advanced-topics).md"
    cd ..
fi

# 处理 plugins 子目录
if [ -d "plugins" ]; then
    cd plugins
    [ -f "kubernetes.md" ] && mv "kubernetes.md" "Kubernetes插件(kubernetes).md"
    [ -f "monitor.md" ] && mv "monitor.md" "监控插件(monitor).md"
    [ -f "nginx.md" ] && mv "nginx.md" "Nginx插件(nginx).md"
    [ -f "ssl-cert.md" ] && mv "ssl-cert.md" "SSL证书插件(ssl-cert).md"
    [ -f "task.md" ] && mv "task.md" "任务插件(task).md"
    cd ..
fi

echo "✅ 重命名完成！"
