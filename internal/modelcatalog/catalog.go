package modelcatalog

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	internalmodels "github.com/jordanhubbard/agenticorp/internal/models"
)

var (
	totalParamRe  = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)B`)
	activeParamRe = regexp.MustCompile(`(?i)A(\d+(?:\.\d+)?)B`)
)

// Catalog holds the recommended model list.
type Catalog struct {
	models []internalmodels.ModelSpec
}

func NewCatalog(models []internalmodels.ModelSpec) *Catalog {
	return &Catalog{models: models}
}

func DefaultCatalog() *Catalog {
	defaults := []internalmodels.ModelSpec{
		withParsed(internalmodels.ModelSpec{Name: "Qwen/Qwen3-Coder-480B-A35B-Instruct", Interactivity: "slow", MinVRAMGB: 320, SuggestedGPUClass: "multi-gpu", Rank: 1}),
		withParsed(internalmodels.ModelSpec{Name: "Qwen/Qwen3-Coder-30B-A3B-Instruct", Interactivity: "medium", MinVRAMGB: 48, SuggestedGPUClass: "A100-80GB", Rank: 2}),
		withParsed(internalmodels.ModelSpec{Name: "NVIDIA-Nemotron-3-Nano-30B-A3B-BF16", Interactivity: "fast", MinVRAMGB: 48, SuggestedGPUClass: "A100-80GB", Rank: 3}),
		withParsed(internalmodels.ModelSpec{Name: "Qwen2.5-Coder-32B-Instruct", Interactivity: "medium", MinVRAMGB: 48, SuggestedGPUClass: "A100-80GB", Rank: 4}),
		withParsed(internalmodels.ModelSpec{Name: "Qwen2.5-Coder-7B-Instruct", Interactivity: "fast", MinVRAMGB: 16, SuggestedGPUClass: "L40S", Rank: 5}),
	}

	return NewCatalog(defaults)
}

func withParsed(spec internalmodels.ModelSpec) internalmodels.ModelSpec {
	parsed := ParseModelName(spec.Name)
	if spec.Vendor == "" {
		spec.Vendor = parsed.Vendor
	}
	if spec.Family == "" {
		spec.Family = parsed.Family
	}
	if spec.TotalParamsB == 0 {
		spec.TotalParamsB = parsed.TotalParamsB
	}
	if spec.ActiveParamsB == 0 {
		spec.ActiveParamsB = parsed.ActiveParamsB
	}
	if spec.Precision == "" {
		spec.Precision = parsed.Precision
	}
	if !spec.Instruct {
		spec.Instruct = parsed.Instruct
	}
	return spec
}

// ParseModelName extracts metadata from a model name.
func ParseModelName(name string) internalmodels.ModelSpec {
	spec := internalmodels.ModelSpec{Name: name}
	parts := strings.Split(name, "/")
	if len(parts) > 1 {
		spec.Vendor = parts[0]
		name = parts[len(parts)-1]
	}
	nameParts := strings.Split(name, "-")
	if len(nameParts) > 0 {
		spec.Family = nameParts[0]
	}

	if matches := totalParamRe.FindAllStringSubmatch(name, -1); len(matches) > 0 {
		// prefer the last match that is not the active parameter token
		for i := len(matches) - 1; i >= 0; i-- {
			val := matches[i][1]
			if activeParamRe.MatchString(matches[i][0]) {
				continue
			}
			spec.TotalParamsB = parseFloat(val)
			break
		}
	}
	if match := activeParamRe.FindStringSubmatch(name); len(match) > 1 {
		spec.ActiveParamsB = parseFloat(match[1])
	}

	upper := strings.ToUpper(name)
	for _, prec := range []string{"BF16", "FP16", "FP8", "INT8", "INT4"} {
		if strings.Contains(upper, prec) {
			spec.Precision = prec
			break
		}
	}
	if strings.Contains(strings.ToLower(name), "instruct") {
		spec.Instruct = true
	}

	return spec
}

func parseFloat(val string) float64 {
	if val == "" {
		return 0
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0
	}
	return f
}

// List returns catalog models in rank order.
func (c *Catalog) List() []internalmodels.ModelSpec {
	if c == nil {
		return nil
	}
	models := make([]internalmodels.ModelSpec, len(c.models))
	copy(models, c.models)
	sort.SliceStable(models, func(i, j int) bool {
		return models[i].Rank < models[j].Rank
	})
	return models
}

// Score computes a heuristic score for a model spec.
func (c *Catalog) Score(spec internalmodels.ModelSpec) float64 {
	base := 100.0
	switch strings.ToLower(spec.Interactivity) {
	case "fast":
		base += 10
	case "medium":
		base += 5
	case "slow":
		base -= 5
	}
	if spec.TotalParamsB > 0 {
		base -= math.Log10(spec.TotalParamsB+1) * 10
	}
	if spec.Rank > 0 {
		base -= float64(spec.Rank)
	}
	return math.Round(base*100) / 100
}

// SelectBest matches available models and returns the top-ranked candidate.
func (c *Catalog) SelectBest(available []string) (*internalmodels.ModelSpec, float64, bool) {
	if c == nil {
		return nil, 0, false
	}
	availableSet := map[string]struct{}{}
	for _, a := range available {
		availableSet[strings.ToLower(a)] = struct{}{}
	}

	var best *internalmodels.ModelSpec
	bestScore := -math.MaxFloat64
	for _, spec := range c.List() {
		if _, ok := availableSet[strings.ToLower(spec.Name)]; !ok {
			continue
		}
		score := c.Score(spec)
		if best == nil || score > bestScore {
			copy := spec
			best = &copy
			bestScore = score
		}
	}

	if best == nil {
		return nil, 0, false
	}
	return best, bestScore, true
}

func (c *Catalog) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.models)
}

func (c *Catalog) Replace(models []internalmodels.ModelSpec) {
	if c == nil {
		return
	}
	for i := range models {
		if models[i].Name != "" {
			models[i] = withParsed(models[i])
		}
	}
	c.models = models
}

func (c *Catalog) Validate() error {
	if c == nil {
		return fmt.Errorf("catalog is nil")
	}
	if len(c.models) == 0 {
		return fmt.Errorf("catalog is empty")
	}
	for _, spec := range c.models {
		if spec.Name == "" {
			return fmt.Errorf("model name is required")
		}
	}
	return nil
}
