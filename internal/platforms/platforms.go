// Package platforms holds the platform database used by userhunt.
//
// Each Platform describes how to detect whether a given username exists on a
// particular site. The full list is embedded into the binary so that the tool
// requires no external configuration.
package platforms

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed platforms.json
var rawJSON []byte

// CheckType determines how userhunt verifies whether a username exists.
type CheckType string

const (
	// CheckStatus means HTTP 200 = exists, 404 = does not exist, anything
	// else is treated as an unreliable/error result.
	CheckStatus CheckType = "status"
	// CheckContent means we always fetch the body and look for marker
	// substrings declared on the platform.
	CheckContent CheckType = "content"
)

// Platform describes a single site that userhunt can probe.
//
// URL must contain the literal `{}` placeholder which is replaced with the
// target username at probe time. For CheckContent platforms, ExistsContent
// takes priority over NotExistsContent: if any ExistsContent substring is
// found, the account is reported as existing.
type Platform struct {
	Name             string    `json:"name"`
	URL              string    `json:"url"`
	Category         string    `json:"category"`
	CheckType        CheckType `json:"check_type"`
	NotExistsContent []string  `json:"not_exists_content,omitempty"`
	ExistsContent    []string  `json:"exists_content,omitempty"`
	NSFW             bool      `json:"nsfw,omitempty"`
}

// BuildURL substitutes the username placeholder in p.URL.
func (p Platform) BuildURL(username string) string {
	return strings.ReplaceAll(p.URL, "{}", username)
}

type fileShape struct {
	Platforms []Platform `json:"platforms"`
}

// Load parses the embedded platforms.json and validates every entry.
func Load() ([]Platform, error) {
	var f fileShape
	if err := json.Unmarshal(rawJSON, &f); err != nil {
		return nil, fmt.Errorf("decode platforms: %w", err)
	}
	if len(f.Platforms) == 0 {
		return nil, fmt.Errorf("embedded platform list is empty")
	}
	for i, p := range f.Platforms {
		if p.Name == "" {
			return nil, fmt.Errorf("platform #%d: missing name", i)
		}
		if !strings.Contains(p.URL, "{}") {
			return nil, fmt.Errorf("platform %q: URL missing {} placeholder", p.Name)
		}
		switch p.CheckType {
		case CheckStatus, CheckContent:
		default:
			return nil, fmt.Errorf("platform %q: invalid check_type %q", p.Name, p.CheckType)
		}
		if p.CheckType == CheckContent && len(p.NotExistsContent) == 0 && len(p.ExistsContent) == 0 {
			return nil, fmt.Errorf("platform %q: content check requires markers", p.Name)
		}
	}
	sort.SliceStable(f.Platforms, func(i, j int) bool {
		return strings.ToLower(f.Platforms[i].Name) < strings.ToLower(f.Platforms[j].Name)
	})
	return f.Platforms, nil
}

// Filter narrows a platform slice by category and NSFW preference.
// Empty category means "all". includeNSFW=false removes NSFW entries.
func Filter(all []Platform, category string, includeNSFW bool) []Platform {
	category = strings.ToLower(strings.TrimSpace(category))
	out := make([]Platform, 0, len(all))
	for _, p := range all {
		if !includeNSFW && p.NSFW {
			continue
		}
		if category != "" && strings.ToLower(p.Category) != category {
			continue
		}
		out = append(out, p)
	}
	return out
}

// Categories returns the unique category names across all platforms, sorted.
func Categories(all []Platform) []string {
	set := make(map[string]struct{}, 16)
	for _, p := range all {
		if p.Category == "" {
			continue
		}
		set[p.Category] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}
