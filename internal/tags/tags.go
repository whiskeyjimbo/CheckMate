package tags

func Deduplicate(tags []string) []string {
	seen := make(map[string]bool)
	deduped := make([]string, 0, len(tags))

	for _, tag := range tags {
		if !seen[tag] {
			seen[tag] = true
			deduped = append(deduped, tag)
		}
	}
	return deduped
}

func HasMatching(allTags, ruleTags []string) bool {
	if len(ruleTags) == 0 {
		return true
	}

	tagMap := make(map[string]bool)
	for _, tag := range allTags {
		tagMap[tag] = true
	}

	for _, ruleTag := range ruleTags {
		if tagMap[ruleTag] {
			return true
		}
	}
	return false
}

func MergeTags(tags1, tags2 []string) []string {
	merged := make([]string, 0, len(tags1)+len(tags2))
	merged = append(merged, tags1...)
	merged = append(merged, tags2...)
	return merged
}
