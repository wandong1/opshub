-- 为站点表添加代理配置字段
-- 执行时间：2026-04-18

ALTER TABLE websites
ADD COLUMN proxy_strategy VARCHAR(20) DEFAULT 'hybrid' COMMENT '代理策略 minimal/standard/hybrid/aggressive',
ADD COLUMN proxy_whitelist TEXT COMMENT '白名单路径(JSON数组)',
ADD COLUMN proxy_blacklist TEXT COMMENT '黑名单路径(JSON数组)',
ADD COLUMN inject_script TINYINT DEFAULT 1 COMMENT '是否注入拦截脚本',
ADD COLUMN rewrite_html TINYINT DEFAULT 1 COMMENT '是否重写HTML',
ADD COLUMN rewrite_css TINYINT DEFAULT 1 COMMENT '是否重写CSS',
ADD COLUMN rewrite_js TINYINT DEFAULT 0 COMMENT '是否重写JS(保守)';

-- 为现有站点设置默认值
UPDATE websites
SET
    proxy_strategy = 'hybrid',
    inject_script = 1,
    rewrite_html = 1,
    rewrite_css = 1,
    rewrite_js = 0
WHERE proxy_strategy IS NULL;
