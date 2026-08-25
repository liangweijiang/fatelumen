# 管理员私有初始化

真实管理员初始化脚本必须放在本目录的 `private/` 子目录中，例如
`private/001-admin.sql`。该目录已被 Git 忽略，不能提交账号、bcrypt 密码哈希、
JWT 密钥或其他凭证。

脚本应当幂等：同一用户名重复执行时不重复创建账号。密码必须由 bcrypt 生成并仅以
哈希形式写入 `admin_users.password_hash`。
