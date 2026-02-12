package provider

import (
	"regexp"
	"strings"
)

// ComplexityLevel represents the estimated complexity of a task/query.
type ComplexityLevel int

const (
	ComplexitySimple   ComplexityLevel = 1 // Review, check, validate, format
	ComplexityMedium   ComplexityLevel = 2 // Implement, fix, refactor, test
	ComplexityComplex  ComplexityLevel = 3 // Design, architect, analyze deeply
	ComplexityExtended ComplexityLevel = 4 // Extended thinking, multi-step reasoning
)

func (c ComplexityLevel) String() string {
	switch c {
	case ComplexitySimple:
		return "simple"
	case ComplexityMedium:
		return "medium"
	case ComplexityComplex:
		return "complex"
	case ComplexityExtended:
		return "extended"
	default:
		return "unknown"
	}
}

// ModelTier represents the capability tier of a model based on size.
type ModelTier int

const (
	TierSmall  ModelTier = 1 // 1-10B params
	TierMedium ModelTier = 2 // 10-50B params
	TierLarge  ModelTier = 3 // 50-200B params
	TierXLarge ModelTier = 4 // 200B+ params
)

// GetModelTier returns the tier for a given model size.
func GetModelTier(paramsB float64) ModelTier {
	switch {
	case paramsB >= 200:
		return TierXLarge
	case paramsB >= 50:
		return TierLarge
	case paramsB >= 10:
		return TierMedium
	default:
		return TierSmall
	}
}

// ComplexityEstimator analyzes queries/tasks to estimate their complexity.
type ComplexityEstimator struct {
	// Pattern matchers for different complexity levels
	simplePatterns   []*regexp.Regexp
	mediumPatterns   []*regexp.Regexp
	complexPatterns  []*regexp.Regexp
	extendedPatterns []*regexp.Regexp
}

// NewComplexityEstimator creates a new complexity estimator with default patterns.
func NewComplexityEstimator() *ComplexityEstimator {
	return &ComplexityEstimator{
		simplePatterns: compilePatterns([]string{
			`(?i)\breview\b`,
			`(?i)\bcheck\b`,
			`(?i)\bvalidate\b`,
			`(?i)\bformat\b`,
			`(?i)\blint\b`,
			`(?i)\bverify\b`,
			`(?i)\blist\b`,
			`(?i)\bsummarize\b`,
			`(?i)\bcount\b`,
			`(?i)\brename\b`,
			`(?i)\bupdate.*comment`,
			`(?i)\badd.*comment`,
			`(?i)\btypo\b`,         // Simple: typo fixes
			`(?i)\bspelling`,
			`(?i)\bcleanup\b`,
			`(?i)\bremove.*unused`,
		}),
		mediumPatterns: compilePatterns([]string{
			`(?i)\bimplement\b`,
			`(?i)\brefactor\b`,
			`(?i)\btest\b`,
			`(?i)\badd.*feature`,
			`(?i)\bcreate.*function`,
			`(?i)\bwrite.*code`,
			`(?i)\bdebug\b`,
			`(?i)\boptimize\b`,
			`(?i)\bintegrate\b`,
			`(?i)\bmigrate\b`,
			`(?i)\bconvert\b`,
			`(?i)\bupgrade\b`,
			`(?i)\bfix\b(?!.*typo)`, // Fix but not "fix typo" (that's simple)
		}),
		complexPatterns: compilePatterns([]string{
			`(?i)\bdesign\b`,
			`(?i)\barchitect\b`,
			`(?i)\banalyze\b`,
			`(?i)\bevaluate\b`,
			`(?i)\bplan\b`,
			`(?i)\bstrateg`,
			`(?i)\bcomplex\b`,
			`(?i)\bsystem.*design`,
			`(?i)\bapi.*design`,
			`(?i)\bdatabase.*design`,
			`(?i)\bsecurity.*review`,
			`(?i)\bperformance.*analysis`,
			`(?i)\bscalability`,
			`(?i)\btrade.?off`,
		}),
		extendedPatterns: compilePatterns([]string{
			`(?i)\bextended.*think`,
			`(?i)\bdeep.*analysis`,
			`(?i)\bcomprehensive`,    // Comprehensive anything = extended
			`(?i)\bfull.*audit`,
			`(?i)\baudit`,            // Audit = extended
			`(?i)\broot.*cause`,
			`(?i)\bmulti.?step`,
			`(?i)\bchain.*of.*thought`,
			`(?i)\breason.*through`,
			`(?i)\bexplain.*detail`,
			`(?i)\bprove\b`,
			`(?i)\bverify.*correct`,
			`(?i)\bcritical.*decision`,
			`(?i)\bhigh.*stakes`,
			`(?i)\birreversible`,
			`(?i)\bformal.*verif`,    // Formal verification
		}),
	}
}

func compilePatterns(patterns []string) []*regexp.Regexp {
	result := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			result = append(result, re)
		}
	}
	return result
}

// EstimateComplexity analyzes a task title and description to estimate complexity.
func (e *ComplexityEstimator) EstimateComplexity(title, description string) ComplexityLevel {
	text := strings.ToLower(title + " " + description)

	// Count matches for each level
	scores := map[ComplexityLevel]int{
		ComplexitySimple:   0,
		ComplexityMedium:   0,
		ComplexityComplex:  0,
		ComplexityExtended: 0,
	}

	for _, re := range e.extendedPatterns {
		if re.MatchString(text) {
			scores[ComplexityExtended]++
		}
	}
	for _, re := range e.complexPatterns {
		if re.MatchString(text) {
			scores[ComplexityComplex]++
		}
	}
	for _, re := range e.mediumPatterns {
		if re.MatchString(text) {
			scores[ComplexityMedium]++
		}
	}
	for _, re := range e.simplePatterns {
		if re.MatchString(text) {
			scores[ComplexitySimple]++
		}
	}

	// Consider context length as a complexity indicator
	wordCount := len(strings.Fields(text))
	if wordCount > 500 {
		scores[ComplexityComplex]++
	} else if wordCount > 200 {
		scores[ComplexityMedium]++
	}

	// Return highest scoring level (with tie-breaker favoring simpler)
	if scores[ComplexityExtended] >= 2 {
		return ComplexityExtended
	}
	if scores[ComplexityComplex] >= 2 || (scores[ComplexityComplex] >= 1 && scores[ComplexityMedium] == 0) {
		return ComplexityComplex
	}
	if scores[ComplexityMedium] >= 1 {
		return ComplexityMedium
	}
	if scores[ComplexitySimple] >= 1 {
		return ComplexitySimple
	}

	// Default to medium if no patterns matched
	return ComplexityMedium
}

// EstimateFromBeadType returns a baseline complexity based on bead type.
func (e *ComplexityEstimator) EstimateFromBeadType(beadType string) ComplexityLevel {
	switch strings.ToLower(beadType) {
	case "chore", "docs", "style":
		return ComplexitySimple
	case "bug", "fix", "test":
		return ComplexityMedium
	case "feature", "enhancement":
		return ComplexityMedium
	case "design", "architecture", "rfc":
		return ComplexityComplex
	case "decision", "critical":
		return ComplexityExtended
	default:
		return ComplexityMedium
	}
}

// CombineEstimates combines type-based and content-based complexity estimates.
func (e *ComplexityEstimator) CombineEstimates(typeComplexity, contentComplexity ComplexityLevel) ComplexityLevel {
	// Take the higher of the two estimates
	if contentComplexity > typeComplexity {
		return contentComplexity
	}
	return typeComplexity
}

// RequiredModelTier returns the minimum model tier needed for a complexity level.
func RequiredModelTier(complexity ComplexityLevel) ModelTier {
	switch complexity {
	case ComplexitySimple:
		return TierSmall
	case ComplexityMedium:
		return TierMedium
	case ComplexityComplex:
		return TierLarge
	case ComplexityExtended:
		return TierXLarge
	default:
		return TierMedium
	}
}

// IsModelSufficientForComplexity checks if a model is capable enough for a task.
func IsModelSufficientForComplexity(paramsB float64, complexity ComplexityLevel) bool {
	modelTier := GetModelTier(paramsB)
	requiredTier := RequiredModelTier(complexity)
	return modelTier >= requiredTier
}
