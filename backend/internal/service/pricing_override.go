package service

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"sort"
	"strings"

	"github.com/yuchen0x1/sub3api-relay/internal/pkg/logger"
)

// validateCustomPricingFiles 要求每个已配置且存在的 fallback/override 文件可读且为 JSON
// 对象，任一不满足即返回带路径的错误；文件不存在视为该层为空，属合法状态。
// 在服务初始化时调用，让"配置了但写坏了"的文件在启动日志里立刻暴露，
// 而不是等到运行期被静默跳过。
func (s *PricingService) validateCustomPricingFiles() error {
	if s == nil || s.cfg == nil {
		return nil
	}
	for _, path := range []string{s.cfg.Pricing.FallbackFile, s.cfg.Pricing.OverrideFile} {
		p := strings.TrimSpace(path)
		if p == "" {
			continue
		}
		body, err := os.ReadFile(p)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		var entries map[string]json.RawMessage
		if err := json.Unmarshal(body, &entries); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
	}
	return nil
}

// loadPricingOverrideEntries 读取 override 文件的原始条目。未配置返回 nil；
// 读取或解析失败打日志并跳过，不影响目录加载。
func (s *PricingService) loadPricingOverrideEntries() map[string]json.RawMessage {
	if s == nil || s.cfg == nil {
		return nil
	}
	path := strings.TrimSpace(s.cfg.Pricing.OverrideFile)
	if path == "" {
		return nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		logger.LegacyPrintf("service.pricing", "[Pricing] Warning: override merge skipped: %v", err)
		return nil
	}
	var entries map[string]json.RawMessage
	if err := json.Unmarshal(body, &entries); err != nil {
		logger.LegacyPrintf("service.pricing", "[Pricing] Warning: override merge skipped: %v", err)
		return nil
	}
	return entries
}

// applyPricingOverrides 把 override 文件的条目逐字段修补进原始目录数据。目录与回退
// 层先合并为"基础价"，override 只做字段级浅覆盖（值为 null 的字段删除该字段）。
func (s *PricingService) applyPricingOverrides(rawData map[string]json.RawMessage) map[string]json.RawMessage {
	overrides := s.loadPricingOverrideEntries()
	if len(overrides) == 0 {
		return rawData
	}
	if rawData == nil {
		rawData = make(map[string]json.RawMessage, len(overrides))
	}
	for name, patch := range overrides {
		base, ok := rawData[name]
		if !ok {
			rawData[name] = patch
			continue
		}
		merged, ok := mergePricingOverrideEntry(base, patch)
		if !ok {
			continue
		}
		rawData[name] = merged
	}
	return rawData
}

// mergePricingOverrideEntry 在 JSON 字段层浅合并：patch 字段覆盖 base 同名字段，
// 值为 null 的 patch 字段从结果中删除，base 为空时结果即 patch 本身。
// patch 不是 JSON 对象时返回 ok=false。
func mergePricingOverrideEntry(base, patch json.RawMessage) (json.RawMessage, bool) {
	var patchFields map[string]any
	if err := json.Unmarshal(patch, &patchFields); err != nil || patchFields == nil {
		return nil, false
	}
	merged := make(map[string]any, len(patchFields))
	if len(base) > 0 {
		// base 非对象时忽略，仅以 patch 为准。
		if err := json.Unmarshal(base, &merged); err != nil {
			merged = make(map[string]any, len(patchFields))
		}
	}
	for k, v := range patchFields {
		if v == nil {
			delete(merged, k)
			continue
		}
		merged[k] = v
	}
	out, err := json.Marshal(merged)
	if err != nil {
		return nil, false
	}
	return out, true
}

// mergeOverrideOnlyModels 把 override 中目录/回退两层都不存在的模型作为独立条目并入
// （条目须自带价格字段才能通过有效性过滤），并对最终仍未生效的条目打 WARN：
// 模型名拼错、或纯补丁条目落在不存在的模型上时会被静默丢弃，让"已改价/已关阶梯"
// 的运营预期与实际计费脱节，这里是唯一的哨兵。
func (s *PricingService) mergeOverrideOnlyModels(data map[string]*LiteLLMModelPricing) map[string]*LiteLLMModelPricing {
	overrides := s.loadPricingOverrideEntries()
	if len(overrides) == 0 {
		return data
	}
	if data == nil {
		data = make(map[string]*LiteLLMModelPricing)
	}
	leftover := make(map[string]json.RawMessage)
	for name, patch := range overrides {
		if _, ok := data[name]; !ok {
			leftover[name] = patch
		}
	}
	if len(leftover) == 0 {
		return data
	}
	// 复用主解析路径（含有效性过滤）；applyPricingOverrides 对已存在条目做的
	// 自我修补是幂等的，不会二次改值。
	if body, err := json.Marshal(leftover); err == nil {
		if parsed, err := s.parsePricingData(body); err == nil {
			maps.Copy(data, parsed)
		}
	}
	var missing []string
	for name := range leftover {
		if _, ok := data[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return data
	}
	sort.Strings(missing)
	logger.LegacyPrintf("service.pricing", "[Pricing] Warning: override had no effect for %d model(s): %s (unknown model name, or patch-only entry without price fields)", len(missing), strings.Join(missing, ", "))
	return data
}
