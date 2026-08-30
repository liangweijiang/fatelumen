# 基础数据初始化包

本目录保存可重复恢复的固定基础数据快照，不保存业务数据或用户数据。

## 文件

- `geo_base_20260830.sql.gz`：国家、城市、四语名称、经纬度和 IANA 时区。
- `annual_calendar_20260830.sql`：2026—2085 公共干支日历，一个完整六十甲子周期。
- `manifest.sha256`：初始化文件的 SHA-256 校验值。

表结构先按 `backend/migrations` 执行，再导入数据文件。地区导入前应确保
`geo_countries` 和 `geo_cities` 为空；干支日历使用年份主键，可安全地以
`INSERT IGNORE` 方式导入。

## 当前快照

| 数据 | 数量 | 范围/说明 |
|---|---:|---|
| 国家/地区 | 246 | 排除废止或没有可用城市的代码 |
| 城市 | 229,758 | GeoNames 固定快照，包含经纬度与时区 |
| 中国城市 | 10,324 | 仅保留存在真实中文名称的记录 |
| 干支年份 | 60 | 2026—2085，六十个干支不重复 |

任何后台修正或导入规则调整，都必须在同一次 Git 提交中更新对应数据文件、
本清单的数量以及 `manifest.sha256`，保证代码、数据库结构和初始化快照一致。

## 导入示例

```bash
gzip -dc backend/seed/geo_base_20260830.sql.gz | mysql --default-character-set=utf8mb4 -u USER -p DATABASE
mysql --default-character-set=utf8mb4 -u USER -p DATABASE < backend/seed/annual_calendar_20260830.sql
```
