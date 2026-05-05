-- 告警治理功能：去重、分组、抑制
-- 创建时间：2026-04-27

-- 1. 告警去重规则表
CREATE TABLE IF NOT EXISTS alert_dedup_rules (
  id INT PRIMARY KEY AUTO_INCREMENT,
  subscription_id INT NOT NULL COMMENT '关联订阅ID',
  name VARCHAR(100) NOT NULL COMMENT '规则名称',
  enabled TINYINT(1) DEFAULT 1 COMMENT '是否启用',

  -- 去重配置
  fingerprint_keys JSON COMMENT '指纹字段: ["severity", "ruleName", "instance"]',
  dedup_window INT DEFAULT 600 COMMENT '去重时间窗口(秒)，默认10分钟',

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  INDEX idx_subscription (subscription_id),
  INDEX idx_enabled (enabled),
  FOREIGN KEY (subscription_id) REFERENCES alert_subscriptions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警去重规则表';

-- 2. 告警指纹记录表（用于去重判断）
CREATE TABLE IF NOT EXISTS alert_fingerprints (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  subscription_id INT NOT NULL COMMENT '订阅ID',
  fingerprint VARCHAR(64) NOT NULL COMMENT 'SHA256指纹',
  rule_name VARCHAR(200) NOT NULL COMMENT '规则名称',
  severity VARCHAR(20) NOT NULL COMMENT '告警级别',
  labels TEXT COMMENT 'JSON标签',
  first_seen_at TIMESTAMP NOT NULL COMMENT '首次出现时间',
  last_seen_at TIMESTAMP NOT NULL COMMENT '最后出现时间',
  occurrence_count INT DEFAULT 1 COMMENT '出现次数',
  last_sent_at TIMESTAMP NULL COMMENT '最后发送时间',

  INDEX idx_fingerprint (subscription_id, fingerprint),
  INDEX idx_last_seen (last_seen_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警指纹记录表';

-- 3. 告警分组规则表
CREATE TABLE IF NOT EXISTS alert_group_rules (
  id INT PRIMARY KEY AUTO_INCREMENT,
  subscription_id INT NOT NULL COMMENT '关联订阅ID',
  name VARCHAR(100) NOT NULL COMMENT '规则名称',
  enabled TINYINT(1) DEFAULT 1 COMMENT '是否启用',

  -- 分组配置
  group_by JSON COMMENT '分组字段: ["severity", "ruleName"]',
  group_wait INT DEFAULT 30 COMMENT '分组等待时间(秒)，收集告警后统一发送',
  group_interval INT DEFAULT 300 COMMENT '分组发送间隔(秒)',
  max_group_size INT DEFAULT 20 COMMENT '单组最大告警数',

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  INDEX idx_subscription (subscription_id),
  INDEX idx_enabled (enabled),
  FOREIGN KEY (subscription_id) REFERENCES alert_subscriptions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警分组规则表';

-- 4. 分组告警缓存表
CREATE TABLE IF NOT EXISTS alert_group_cache (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  group_rule_id INT NOT NULL COMMENT '分组规则ID',
  subscription_id INT NOT NULL COMMENT '订阅ID',
  group_key VARCHAR(255) NOT NULL COMMENT '分组键: severity=critical,ruleName=CPU高',
  alerts JSON COMMENT '缓存的告警事件ID列表: [123, 124, 125]',
  first_alert_at TIMESTAMP NOT NULL COMMENT '首个告警时间',
  last_alert_at TIMESTAMP NOT NULL COMMENT '最后告警时间',
  alert_count INT DEFAULT 1 COMMENT '告警数量',
  sent TINYINT(1) DEFAULT 0 COMMENT '是否已发送',
  sent_at TIMESTAMP NULL COMMENT '发送时间',

  INDEX idx_group_sent (group_rule_id, sent),
  INDEX idx_first_alert (first_alert_at),
  INDEX idx_subscription (subscription_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='分组告警缓存表';

-- 5. 告警抑制规则表
CREATE TABLE IF NOT EXISTS alert_inhibit_rules (
  id INT PRIMARY KEY AUTO_INCREMENT,
  subscription_id INT NOT NULL COMMENT '关联订阅ID',
  name VARCHAR(100) NOT NULL COMMENT '规则名称',
  enabled TINYINT(1) DEFAULT 1 COMMENT '是否启用',

  -- 源告警（抑制源）
  source_matchers JSON COMMENT '源告警匹配条件: {"severity": "critical", "ruleName": "节点宕机"}',

  -- 目标告警（被抑制）
  target_matchers JSON COMMENT '目标告警匹配条件: {"severity": "warning", "ruleName": "服务不可用"}',

  -- 抑制条件（必须相等的标签）
  equal_labels JSON COMMENT '必须相等的标签: ["instance", "cluster"]',

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  INDEX idx_subscription (subscription_id),
  INDEX idx_enabled (enabled),
  FOREIGN KEY (subscription_id) REFERENCES alert_subscriptions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警抑制规则表';
