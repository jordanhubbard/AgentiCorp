package provider

import (
	"testing"
)

func TestComplexityLevelString(t *testing.T) {
	tests := []struct {
		level    ComplexityLevel
		expected string
	}{
		{ComplexitySimple, "simple"},
		{ComplexityMedium, "medium"},
		{ComplexityComplex, "complex"},
		{ComplexityExtended, "extended"},
		{ComplexityLevel(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("ComplexityLevel(%d).String() = %q, want %q", tt.level, got, tt.expected)
		}
	}
}

func TestGetModelTier(t *testing.T) {
	tests := []struct {
		paramsB  float64
		expected ModelTier
	}{
		{0, TierSmall},
		{7, TierSmall},
		{10, TierMedium},
		{32, TierMedium},
		{50, TierLarge},
		{70, TierLarge},
		{200, TierXLarge},
		{480, TierXLarge},
	}

	for _, tt := range tests {
		if got := GetModelTier(tt.paramsB); got != tt.expected {
			t.Errorf("GetModelTier(%f) = %d, want %d", tt.paramsB, got, tt.expected)
		}
	}
}

func TestEstimateComplexitySimple(t *testing.T) {
	e := NewComplexityEstimator()

	simpleQueries := []struct {
		title       string
		description string
	}{
		{"Review the config file", "Check for syntax errors"},
		{"Validate JSON schema", "Make sure all fields are present"},
		{"Format code", "Run the linter on these files"},
		{"List all API endpoints", "Summarize available routes"},
		{"Fix typo in README", "Spelling error on line 42"},
		{"Remove unused imports", "Cleanup the file"},
	}

	for _, q := range simpleQueries {
		result := e.EstimateComplexity(q.title, q.description)
		if result != ComplexitySimple {
			t.Errorf("Expected ComplexitySimple for %q, got %s", q.title, result.String())
		}
	}
}

func TestEstimateComplexityMedium(t *testing.T) {
	e := NewComplexityEstimator()

	mediumQueries := []struct {
		title       string
		description string
	}{
		{"Implement user authentication", "Add login and logout endpoints"},
		{"Fix the bug in payment flow", "Users are getting double-charged"},
		{"Refactor the database layer", "Move from raw SQL to ORM"},
		{"Add unit tests for the API", "Cover all error cases"},
		{"Integrate with Stripe API", "Handle webhooks properly"},
	}

	for _, q := range mediumQueries {
		result := e.EstimateComplexity(q.title, q.description)
		if result != ComplexityMedium {
			t.Errorf("Expected ComplexityMedium for %q, got %s", q.title, result.String())
		}
	}
}

func TestEstimateComplexityComplex(t *testing.T) {
	e := NewComplexityEstimator()

	complexQueries := []struct {
		title       string
		description string
	}{
		{"Design the microservices architecture", "Evaluate trade-offs between monolith and services"},
		{"Architect the data pipeline", "Handle 1M events per second"},
		{"Plan the security review", "Analyze all attack vectors"},
		{"Design API versioning strategy", "Consider backward compatibility"},
		{"Evaluate database scalability options", "Compare sharding vs replication"},
	}

	for _, q := range complexQueries {
		result := e.EstimateComplexity(q.title, q.description)
		if result != ComplexityComplex {
			t.Errorf("Expected ComplexityComplex for %q, got %s", q.title, result.String())
		}
	}
}

func TestEstimateComplexityExtended(t *testing.T) {
	e := NewComplexityEstimator()

	extendedQueries := []struct {
		title       string
		description string
	}{
		{"Extended thinking session on architecture", "Need deep analysis of all components"},
		{"Root cause analysis of production outage", "Multi-step investigation required"},
		{"Comprehensive security audit", "Full audit of all systems"},
		{"Prove the algorithm is correct", "Formal verification needed"},
		{"Critical decision on infrastructure", "Irreversible change, high stakes"},
	}

	for _, q := range extendedQueries {
		result := e.EstimateComplexity(q.title, q.description)
		if result != ComplexityExtended {
			t.Errorf("Expected ComplexityExtended for %q, got %s", q.title, result.String())
		}
	}
}

func TestEstimateFromBeadType(t *testing.T) {
	e := NewComplexityEstimator()

	tests := []struct {
		beadType string
		expected ComplexityLevel
	}{
		{"chore", ComplexitySimple},
		{"docs", ComplexitySimple},
		{"style", ComplexitySimple},
		{"bug", ComplexityMedium},
		{"fix", ComplexityMedium},
		{"test", ComplexityMedium},
		{"feature", ComplexityMedium},
		{"enhancement", ComplexityMedium},
		{"design", ComplexityComplex},
		{"architecture", ComplexityComplex},
		{"rfc", ComplexityComplex},
		{"decision", ComplexityExtended},
		{"critical", ComplexityExtended},
		{"unknown", ComplexityMedium},
	}

	for _, tt := range tests {
		result := e.EstimateFromBeadType(tt.beadType)
		if result != tt.expected {
			t.Errorf("EstimateFromBeadType(%q) = %s, want %s", tt.beadType, result.String(), tt.expected.String())
		}
	}
}

func TestCombineEstimates(t *testing.T) {
	e := NewComplexityEstimator()

	tests := []struct {
		typeComplexity    ComplexityLevel
		contentComplexity ComplexityLevel
		expected          ComplexityLevel
	}{
		{ComplexitySimple, ComplexitySimple, ComplexitySimple},
		{ComplexitySimple, ComplexityComplex, ComplexityComplex},
		{ComplexityComplex, ComplexitySimple, ComplexityComplex},
		{ComplexityMedium, ComplexityExtended, ComplexityExtended},
	}

	for _, tt := range tests {
		result := e.CombineEstimates(tt.typeComplexity, tt.contentComplexity)
		if result != tt.expected {
			t.Errorf("CombineEstimates(%s, %s) = %s, want %s",
				tt.typeComplexity.String(), tt.contentComplexity.String(),
				result.String(), tt.expected.String())
		}
	}
}

func TestRequiredModelTier(t *testing.T) {
	tests := []struct {
		complexity ComplexityLevel
		expected   ModelTier
	}{
		{ComplexitySimple, TierSmall},
		{ComplexityMedium, TierMedium},
		{ComplexityComplex, TierLarge},
		{ComplexityExtended, TierXLarge},
	}

	for _, tt := range tests {
		result := RequiredModelTier(tt.complexity)
		if result != tt.expected {
			t.Errorf("RequiredModelTier(%s) = %d, want %d", tt.complexity.String(), result, tt.expected)
		}
	}
}

func TestIsModelSufficientForComplexity(t *testing.T) {
	tests := []struct {
		paramsB    float64
		complexity ComplexityLevel
		expected   bool
	}{
		// Small model (7B) capabilities
		{7, ComplexitySimple, true},
		{7, ComplexityMedium, false},
		{7, ComplexityComplex, false},
		{7, ComplexityExtended, false},

		// Medium model (32B) capabilities
		{32, ComplexitySimple, true},
		{32, ComplexityMedium, true},
		{32, ComplexityComplex, false},
		{32, ComplexityExtended, false},

		// Large model (70B) capabilities
		{70, ComplexitySimple, true},
		{70, ComplexityMedium, true},
		{70, ComplexityComplex, true},
		{70, ComplexityExtended, false},

		// XLarge model (480B) capabilities
		{480, ComplexitySimple, true},
		{480, ComplexityMedium, true},
		{480, ComplexityComplex, true},
		{480, ComplexityExtended, true},
	}

	for _, tt := range tests {
		result := IsModelSufficientForComplexity(tt.paramsB, tt.complexity)
		if result != tt.expected {
			t.Errorf("IsModelSufficientForComplexity(%.0fB, %s) = %v, want %v",
				tt.paramsB, tt.complexity.String(), result, tt.expected)
		}
	}
}

func TestRankProvidersForComplexity(t *testing.T) {
	s := NewScorer()

	// Setup providers with different model sizes
	s.UpdateProviderMetrics("small", 7, 100, 500, 0)    // 7B - TierSmall
	s.UpdateProviderMetrics("medium", 32, 100, 500, 0)  // 32B - TierMedium
	s.UpdateProviderMetrics("large", 70, 100, 500, 0)   // 70B - TierLarge
	s.UpdateProviderMetrics("xlarge", 480, 100, 500, 0) // 480B - TierXLarge

	providerIDs := []string{"small", "medium", "large", "xlarge"}

	// Simple task should prefer small model
	simpleRanked := s.RankProvidersForComplexity(providerIDs, ComplexitySimple)
	if simpleRanked[0] != "small" {
		t.Errorf("Simple task should prefer small model, got %s first", simpleRanked[0])
	}

	// Medium task should prefer medium model
	mediumRanked := s.RankProvidersForComplexity(providerIDs, ComplexityMedium)
	if mediumRanked[0] != "medium" {
		t.Errorf("Medium task should prefer medium model, got %s first", mediumRanked[0])
	}

	// Complex task should prefer large model
	complexRanked := s.RankProvidersForComplexity(providerIDs, ComplexityComplex)
	if complexRanked[0] != "large" {
		t.Errorf("Complex task should prefer large model, got %s first", complexRanked[0])
	}

	// Extended task should prefer xlarge model
	extendedRanked := s.RankProvidersForComplexity(providerIDs, ComplexityExtended)
	if extendedRanked[0] != "xlarge" {
		t.Errorf("Extended task should prefer xlarge model, got %s first", extendedRanked[0])
	}
}

func TestRankProvidersForComplexityFallback(t *testing.T) {
	s := NewScorer()

	// Setup providers without a perfect match
	s.UpdateProviderMetrics("medium", 32, 100, 500, 0)  // 32B - TierMedium
	s.UpdateProviderMetrics("xlarge", 480, 100, 500, 0) // 480B - TierXLarge

	providerIDs := []string{"medium", "xlarge"}

	// Simple task with no small model should fall back to medium
	simpleRanked := s.RankProvidersForComplexity(providerIDs, ComplexitySimple)
	if simpleRanked[0] != "medium" {
		t.Errorf("Simple task should fall back to smallest available (medium), got %s first", simpleRanked[0])
	}

	// Complex task with no large model should fall back to xlarge
	complexRanked := s.RankProvidersForComplexity(providerIDs, ComplexityComplex)
	if complexRanked[0] != "xlarge" {
		t.Errorf("Complex task should fall back to xlarge, got %s first", complexRanked[0])
	}
}
