package resource

import "strings"

// Query DSL 解析辅助：从 query string 解析筛选/排序/分页。

const (
	FilterGTE = "__gte"
	FilterLTE = "__lte"
	FilterGT  = "__gt"
	FilterLT  = "__lt"
	FilterIn  = "__in"
	FilterLike = "__like"
)

// FilterSuffix 返回筛选后缀类型：空字符串 = 精确匹配。
func FilterSuffix(key string) (field string, op string) {
	for _, suf := range []string{FilterGTE, FilterLTE, FilterGT, FilterLT, FilterIn, FilterLike} {
		if strings.HasSuffix(key, suf) {
			return strings.TrimSuffix(key, suf), suf
		}
	}
	return key, ""
}
