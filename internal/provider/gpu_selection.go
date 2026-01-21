package provider

import (
	"fmt"
	"strings"

	internalmodels "github.com/jordanhubbard/agenticorp/internal/models"
)

// GPUInfo represents information about an available GPU
type GPUInfo struct {
	ID       string `json:"id"`        // Device ID or identifier
	Name     string `json:"name"`      // GPU model name
	VRAMGByte int     `json:"vram_gb"`   // VRAM in GB
	Arch     string `json:"arch"`      // Architecture (ampere, hopper, ada, etc.)
	InUse    bool   `json:"in_use"`    // Whether currently allocated
}

// SelectGPU selects an appropriate GPU for a model based on constraints
// Returns the selected GPU ID and explanation, or empty string if no selection needed
func SelectGPU(modelSpec *internalmodels.ModelSpec, constraints *internalmodels.GPUConstraints, availableGPUs []GPUInfo) (string, string) {
	// If no constraints or no GPUs available, no selection
	if constraints == nil || len(availableGPUs) == 0 {
		return "", "No GPU constraints or GPU list provided"
	}
	
	// If model spec is nil, we can't make intelligent decisions
	if modelSpec == nil {
		return "", "No model specification for GPU selection"
	}
	
	// Filter GPUs based on constraints
	var candidates []GPUInfo
	
	for _, gpu := range availableGPUs {
		// Skip GPUs in use
		if gpu.InUse {
			continue
		}
		
		// Check VRAM requirement (from model or constraints)
		requiredVRAM := modelSpec.MinVRAMGB
		if constraints.MinVRAMGB > requiredVRAM {
			requiredVRAM = constraints.MinVRAMGB
		}
		if requiredVRAM > 0 && gpu.VRAMGByte < requiredVRAM {
			continue
		}
		
		// Check architecture requirement
		if constraints.RequiredGPUArch != "" {
			if !strings.EqualFold(gpu.Arch, constraints.RequiredGPUArch) {
				continue
			}
		}
		
		// Check allowed GPU IDs
		if len(constraints.AllowedGPUIDs) > 0 {
			found := false
			for _, allowedID := range constraints.AllowedGPUIDs {
				if gpu.ID == allowedID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		candidates = append(candidates, gpu)
	}
	
	// No candidates found
	if len(candidates) == 0 {
		return "", fmt.Sprintf("No GPU meets requirements (VRAM: %dGB, Arch: %s)", 
			modelSpec.MinVRAMGB, constraints.RequiredGPUArch)
	}
	
	// Select best candidate
	// Priority: 1) Preferred class match, 2) Most VRAM, 3) First available
	var selected *GPUInfo
	
	// Try to match preferred class
	if constraints.PreferredClass != "" {
		for i := range candidates {
			if strings.Contains(candidates[i].Name, constraints.PreferredClass) {
				selected = &candidates[i]
				break
			}
		}
	}
	
	// Try to match suggested GPU class from model
	if selected == nil && modelSpec.SuggestedGPUClass != "" {
		for i := range candidates {
			if strings.Contains(candidates[i].Name, modelSpec.SuggestedGPUClass) {
				selected = &candidates[i]
				break
			}
		}
	}
	
	// Fall back to GPU with most VRAM
	if selected == nil {
		selected = &candidates[0]
		for i := range candidates {
			if candidates[i].VRAMGByte > selected.VRAMGByte {
				selected = &candidates[i]
			}
		}
	}
	
	reason := fmt.Sprintf("Selected %s (VRAM: %dGB, Arch: %s) for model requiring %dGB", 
		selected.Name, selected.VRAMGByte, selected.Arch, modelSpec.MinVRAMGB)
	
	return selected.ID, reason
}

// InferGPUConstraintsFromModel creates GPU constraints based on model requirements
func InferGPUConstraintsFromModel(modelSpec *internalmodels.ModelSpec) *internalmodels.GPUConstraints {
	if modelSpec == nil {
		return nil
	}
	
	constraints := &internalmodels.GPUConstraints{
		MinVRAMGB:      modelSpec.MinVRAMGB,
		PreferredClass: modelSpec.SuggestedGPUClass,
	}
	
	// Only return if there are actual constraints
	if constraints.MinVRAMGB == 0 && constraints.PreferredClass == "" {
		return nil
	}
	
	return constraints
}
