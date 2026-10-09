package engine

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var varRegex = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_\.\-]+)\s*\}\}`)

// ResolvePath 从 ContextScope 中解析点路径变量 (例如: params.userId, step-1.output.id, sys.timestamp)
func (s *ContextScope) ResolvePath(path string) (interface{}, bool) {
	// 1. 系统内置动态变量
	if strings.HasPrefix(path, "sys.") {
		switch path {
		case "sys.timestamp":
			return time.Now().UnixMilli(), true
		case "sys.time_sec":
			return time.Now().Unix(), true
		case "sys.date":
			return time.Now().Format("2006-01-02"), true
		case "sys.datetime":
			return time.Now().Format("2006-01-02 15:04:05"), true
		case "sys.uuid":
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			return hex.EncodeToString(b), true
		}
	}

	// 2. 先尝试直接根据完整 key 查询 (针对平铺键)
	if val, ok := s.Get(path); ok {
		return val, true
	}

	// 3. 层级点路径解析 (a.b.c)
	parts := strings.Split(path, ".")
	if len(parts) <= 1 {
		return nil, false
	}

	rootVal, ok := s.Get(parts[0])
	if !ok {
		return nil, false
	}

	curr := rootVal
	for i := 1; i < len(parts); i++ {
		key := parts[i]
		if m, isMap := curr.(map[string]interface{}); isMap {
			var nextVal interface{}
			if nextVal, ok = m[key]; !ok {
				return nil, false
			}
			curr = nextVal
		} else {
			return nil, false
		}
	}

	return curr, true
}

// InterpolateValue 递归对配置值进行变量插值替换
func InterpolateValue(val interface{}, scope *ContextScope) interface{} {
	if val == nil || scope == nil {
		return val
	}

	switch v := val.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		// 边界情况：如果是整串恰好为一个变量定义，例如 "{{ params.amount }}"
		// 则尽可能保留原数据类型 (如 float64, int, bool)
		if strings.HasPrefix(trimmed, "{{") && strings.HasSuffix(trimmed, "}}") {
			matches := varRegex.FindStringSubmatch(trimmed)
			if len(matches) == 2 && matches[0] == trimmed {
				varName := strings.TrimSpace(matches[1])
				if resolved, found := scope.ResolvePath(varName); found {
					return resolved
				}
			}
		}

		// 文本内嵌变量替换 (如 "Bearer {{ params.token }}" 或 SQL 模板)
		return varRegex.ReplaceAllStringFunc(v, func(match string) string {
			subMatches := varRegex.FindStringSubmatch(match)
			if len(subMatches) < 2 {
				return match
			}
			varName := strings.TrimSpace(subMatches[1])
			if resolved, found := scope.ResolvePath(varName); found {
				return fmt.Sprintf("%v", resolved)
			}
			// 未解析到变量则保留原表达式
			return match
		})

	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for k, item := range v {
			resolvedKey := k
			if strings.Contains(k, "{{") {
				if sKey, ok := InterpolateValue(k, scope).(string); ok {
					resolvedKey = sKey
				}
			}
			result[resolvedKey] = InterpolateValue(item, scope)
		}
		return result

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = InterpolateValue(item, scope)
		}
		return result

	default:
		return v
	}
}

// InterpolateConfig 对节点的 Config 配置进行深度变量填充
func InterpolateConfig(config map[string]interface{}, scope *ContextScope) map[string]interface{} {
	if config == nil {
		return make(map[string]interface{})
	}
	res := InterpolateValue(config, scope)
	if m, ok := res.(map[string]interface{}); ok {
		return m
	}
	return config
}
