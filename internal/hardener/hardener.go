package hardener

import (
	"fmt"
	"math"
	"regexp"

	"github.com/alexandrmotologa/toolcraft/internal/config"
	"github.com/alexandrmotologa/toolcraft/internal/domain"
)

// Harden processes an input specification through the hardening and optimization pipeline.
func Harden(spec *domain.Spec, cfg config.HardeningConfig) (*domain.Spec, *domain.HardeningReport, error) {
	compiledPatterns := make([]*regexp.Regexp, 0, len(cfg.PruneParameters))
	for _, p := range cfg.PruneParameters {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid pruning regex pattern '%s': %w", p, err)
		}
		compiledPatterns = append(compiledPatterns, re)
	}

	hardenedSpec := &domain.Spec{
		Title:       spec.Title,
		Description: spec.Description,
		Version:     spec.Version,
		BaseURL:     spec.BaseURL,
		Servers:     spec.Servers,
		Tools:       make([]domain.ToolDefinition, 0),
	}

	report := &domain.HardeningReport{
		OriginalOperations: len(spec.Tools),
	}

	rawTokens := 0
	hardenedTokens := 0

	for _, tool := range spec.Tools {
		// Calculate raw tokens estimate for unpruned tool
		rawTokens += estimateToolTokens(&tool)
		report.OriginalParameters += len(tool.Parameters)
		if tool.RequestBody != nil {
			report.OriginalParameters += len(tool.RequestBody.Properties)
		}

		// Enforce read-only if configured
		if cfg.ReadOnly && !tool.ReadOnly {
			continue // Mutating methods are blocked/excluded in read-only mode
		}

		hardenedTool := tool

		// Prune query/header parameters
		retainedParams, discardedParams := PruneParameters(tool.Parameters, compiledPatterns, cfg.MaxParametersPerTool)
		report.PrunedParametersCount += len(discardedParams)

		// Validate parameter types
		for i := range retainedParams {
			ValidateParameter(&retainedParams[i])
		}
		hardenedTool.Parameters = retainedParams

		// Prune and validate request body properties if present
		if tool.RequestBody != nil && len(tool.RequestBody.Properties) > 0 {
			retainedBody, discardedBody := PruneParameters(tool.RequestBody.Properties, compiledPatterns, cfg.MaxParametersPerTool)
			report.PrunedParametersCount += len(discardedBody)
			for i := range retainedBody {
				ValidateParameter(&retainedBody[i])
			}
			hardenedTool.RequestBody.Properties = retainedBody
		}

		// Enrich tool description for LLM tool selection
		if cfg.EnrichDescriptions {
			hardenedTool.Description = EnhanceDescription(&hardenedTool)
		}

		hardenedTokens += estimateToolTokens(&hardenedTool)
		report.HardenedParameters += len(hardenedTool.Parameters)
		if hardenedTool.RequestBody != nil {
			report.HardenedParameters += len(hardenedTool.RequestBody.Properties)
		}

		hardenedSpec.Tools = append(hardenedSpec.Tools, hardenedTool)
	}

	report.HardenedOperations = len(hardenedSpec.Tools)
	report.EstimatedRawTokens = rawTokens
	report.EstimatedHardenedTokens = hardenedTokens

	if rawTokens > 0 {
		diff := float64(rawTokens - hardenedTokens)
		report.ReductionPercent = math.Round((diff/float64(rawTokens))*1000) / 10
	}

	return hardenedSpec, report, nil
}

// estimateToolTokens estimates LLM context prompt tokens based on tool schema size in MCP JSON Schema.
func estimateToolTokens(tool *domain.ToolDefinition) int {
	// Base MCP tool wrapper: name, description, inputSchema structure
	tokens := 40 + len(tool.ID)/4 + len(tool.Description)/4

	for _, p := range tool.Parameters {
		tokens += 25 + len(p.Name)/4 + len(p.Description)/4 + len(p.Type)/4
		for _, e := range p.Enum {
			tokens += len(e)/4 + 2
		}
	}

	if tool.RequestBody != nil {
		tokens += 20
		for _, p := range tool.RequestBody.Properties {
			tokens += 25 + len(p.Name)/4 + len(p.Description)/4 + len(p.Type)/4
		}
	}

	return tokens
}
