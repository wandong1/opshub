-- 为站点表添加代理访问 Token 字段
-- 执行时间：2026-04-18

-- 添加 proxy_token 字段
ALTER TABLE websites
ADD COLUMN proxy_token VARCHAR(64) COMMENT '代理访问Token(UUID)';

-- 为现有站点生成 proxy_token
UPDATE websites
SET proxy_token = REPLACE(UUID(), '-', '')
WHERE proxy_token IS NULL OR proxy_token = '';

-- 添加唯一索引
ALTER TABLE websites
ADD UNIQUE INDEX idx_proxy_token (proxy_token);
