package service

import "strings"

func filterXimoAIModelPlazaModels(channel *Channel, models []SupportedModel) []SupportedModel {
	type modelKey struct {
		platform string
		name     string
	}
	mappingSources := make(map[modelKey]struct{})
	mappedTargets := make(map[modelKey]struct{})
	if channel != nil {
		for platform, mapping := range channel.ModelMapping {
			platform = strings.ToLower(strings.TrimSpace(platform))
			for source, target := range mapping {
				source = strings.TrimSpace(source)
				target = strings.TrimSpace(target)
				if source == "" {
					continue
				}
				if _, wildcard := splitWildcardSuffix(source); wildcard {
					continue
				}
				sourceKey := modelKey{platform: platform, name: strings.ToLower(source)}
				mappingSources[sourceKey] = struct{}{}
				if target == "" || strings.EqualFold(source, target) {
					continue
				}
				if _, wildcard := splitWildcardSuffix(target); wildcard {
					continue
				}
				mappedTargets[modelKey{platform: platform, name: strings.ToLower(target)}] = struct{}{}
			}
		}
	}

	out := make([]SupportedModel, 0, len(models))
	for i := range models {
		if models[i].Pricing == nil {
			continue
		}
		key := modelKey{
			platform: strings.ToLower(strings.TrimSpace(models[i].Platform)),
			name:     strings.ToLower(strings.TrimSpace(models[i].Name)),
		}
		if _, isTarget := mappedTargets[key]; isTarget {
			if _, isSource := mappingSources[key]; !isSource {
				continue
			}
		}
		out = append(out, models[i])
	}
	return out
}
