// Copyright (C) 2025 Jeff Rose
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

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
