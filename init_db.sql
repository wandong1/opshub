-- 禁用外键检查
SET FOREIGN_KEY_CHECKS = 0;

-- 删除所有RBAC相关的表（注意：会删除所有数据）
DROP TABLE IF EXISTS `sys_role_menus`;
DROP TABLE IF EXISTS `sys_role_menu`;
DROP TABLE IF EXISTS `sys_user_roles`;
DROP TABLE IF EXISTS `sys_user_role`;
DROP TABLE IF EXISTS `sys_menu`;
DROP TABLE IF EXISTS `sys_department`;
DROP TABLE IF EXISTS `sys_role`;
DROP TABLE IF EXISTS `sys_user`;

-- 恢复外键检查
SET FOREIGN_KEY_CHECKS = 1;
