package hardener

import (
	"regexp"
	"sort"

	"github.com/alexandrmotologa/toolcraft/internal/domain"
)

// PruneParameters filters out redundant and low-value parameters using regex patterns and priority ranking.
func PruneParameters(params []domain.ParameterDefinition, patterns []*regexp.Regexp, maxParams int) ([]domain.ParameterDefinition, []domain.ParameterDefinition) {
	var retained []domain.ParameterDefinition
	var discarded []domain.ParameterDefinition

	for _, param := range params {
		// Path parameters must always be retained for HTTP route resolution
		if param.In == "path" {
			retained = append(retained, param)
			continue
		}

		// Check against pruning regex patterns
		isPruned := false
		for _, pat := range patterns {
			if pat.MatchString(param.Name) {
				param.DiscardReason = "matches redundant parameter pattern: " + pat.String()
				discarded = append(discarded, param)
				isPruned = true
				break
			}
		}
		if isPruned {
			continue
		}

		retained = append(retained, param)
	}

	// If retained parameters exceed maxParams, rank and cap
	if maxParams > 0 && len(retained) > maxParams {
		// Score and sort retained parameters
		sort.SliceStable(retained, func(i, j int) bool {
			scoreI := scoreParameter(retained[i])
			scoreJ := scoreParameter(retained[j])
			return scoreI > scoreJ
		})

		capped := make([]domain.ParameterDefinition, 0, maxParams)
		for i, param := range retained {
			if i < maxParams || param.Required {
				capped = append(capped, param)
			} else {
				param.DiscardReason = "exceeds maximum parameter cap per tool"
				discarded = append(discarded, param)
			}
		}
		retained = capped
	}

	return retained, discarded
}

// scoreParameter assigns a priority weight to determine retention order.
func scoreParameter(p domain.ParameterDefinition) int {
	score := 0
	if p.In == "path" {
		return 1000
	}
	if p.Required {
		score += 500
	}
	switch p.Name {
	case "id", "status", "query", "search", "filter", "limit":
		score += 100
	}
	if p.In == "query" {
		score += 50
	}
	if p.In == "body" {
		score += 40
	}
	if len(p.Description) > 0 {
		score += 10
	}
	return score
}
