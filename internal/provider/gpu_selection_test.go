package provider

import (
	"testing"

	internalmodels "github.com/jordanhubbard/loom/internal/models"
)

func TestSelectGPU(t *testing.T) {
	tests := []struct {
		name          string
		modelSpec     *internalmodels.ModelSpec
		constraints   *internalmodels.GPUConstraints
		availableGPUs []GPUInfo
		expectGPUID   string
		shouldSelect  bool
	}{
		{
			name: "No constraints or GPUs",
			modelSpec: &internalmodels.ModelSpec{
				MinVRAMGB: 48,
			},
			constraints:   nil,
			availableGPUs: nil,
			expectGPUID:   "",
			shouldSelect:  false,
		},
		{
			name: "Select GPU with sufficient VRAM",
			modelSpec: &internalmodels.ModelSpec{
				MinVRAMGB: 48,
			},
			constraints: &internalmodels.GPUConstraints{
				MinVRAMGB: 48,
			},
			availableGPUs: []GPUInfo{
				{ID: "gpu0", Name: "NVIDIA A100-40GB", VRAMGByte: 40, Arch: "ampere", InUse: false},
				{ID: "gpu1", Name: "NVIDIA A100-80GB", VRAMGByte: 80, Arch: "ampere", InUse: false},
			},
			expectGPUID:  "gpu1",
			shouldSelect: true,
		},
		{
			name: "Skip GPUs in use",
			modelSpec: &internalmodels.ModelSpec{
				MinVRAMGB: 40,
			},
			constraints: &internalmodels.GPUConstraints{
				MinVRAMGB: 40,
			},
			availableGPUs: []GPUInfo{
				{ID: "gpu0", Name: "NVIDIA A100-80GB", VRAMGByte: 80, Arch: "ampere", InUse: true},
				{ID: "gpu1", Name: "NVIDIA A100-80GB", VRAMGByte: 80, Arch: "ampere", InUse: false},
			},
			expectGPUID:  "gpu1",
			shouldSelect: true,
		},
		{
			name: "Filter by architecture",
			modelSpec: &internalmodels.ModelSpec{
				MinVRAMGB: 40,
			},
			constraints: &internalmodels.GPUConstraints{
				MinVRAMGB:       40,
				RequiredGPUArch: "hopper",
			},
			availableGPUs: []GPUInfo{
				{ID: "gpu0", Name: "NVIDIA A100-80GB", VRAMGByte: 80, Arch: "ampere", InUse: false},
				{ID: "gpu1", Name: "NVIDIA H100", VRAMGByte: 80, Arch: "hopper", InUse: false},
			},
			expectGPUID:  "gpu1",
			shouldSelect: true,
		},
		{
			name: "Prefer specific GPU class",
			modelSpec: &internalmodels.ModelSpec{
				MinVRAMGB:         40,
				SuggestedGPUClass: "A100-80GB",
			},
			constraints: &internalmodels.GPUConstraints{
				PreferredClass: "A100-80GB",
			},
			availableGPUs: []GPUInfo{
				{ID: "gpu0", Name: "NVIDIA L40S", VRAMGByte: 48, Arch: "ada", InUse: false},
				{ID: "gpu1", Name: "NVIDIA A100-80GB", VRAMGByte: 80, Arch: "ampere", InUse: false},
			},
			expectGPUID:  "gpu1",
			shouldSelect: true,
		},
		{
			name: "No GPU meets VRAM requirement",
			modelSpec: &internalmodels.ModelSpec{
				MinVRAMGB: 80,
			},
			constraints: &internalmodels.GPUConstraints{
				MinVRAMGB: 80,
			},
			availableGPUs: []GPUInfo{
				{ID: "gpu0", Name: "NVIDIA L40S", VRAMGByte: 48, Arch: "ada", InUse: false},
			},
			expectGPUID:  "",
			shouldSelect: false,
		},
		{
			name: "Filter by allowed GPU IDs",
			modelSpec: &internalmodels.ModelSpec{
				MinVRAMGB: 40,
			},
			constraints: &internalmodels.GPUConstraints{
				AllowedGPUIDs: []string{"gpu1", "gpu2"},
			},
			availableGPUs: []GPUInfo{
				{ID: "gpu0", Name: "NVIDIA A100-80GB", VRAMGByte: 80, Arch: "ampere", InUse: false},
				{ID: "gpu1", Name: "NVIDIA A100-80GB", VRAMGByte: 80, Arch: "ampere", InUse: false},
			},
			expectGPUID:  "gpu1",
			shouldSelect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gpuID, reason := SelectGPU(tt.modelSpec, tt.constraints, tt.availableGPUs)

			if tt.shouldSelect {
				if gpuID == "" {
					t.Errorf("Expected GPU selection but got empty ID. Reason: %s", reason)
				}
				if gpuID != tt.expectGPUID {
					t.Errorf("Expected GPU %s, got %s. Reason: %s", tt.expectGPUID, gpuID, reason)
				}
			} else {
				if gpuID != "" {
					t.Errorf("Expected no GPU selection but got %s. Reason: %s", gpuID, reason)
				}
			}
		})
	}
}

func TestInferGPUConstraintsFromModel(t *testing.T) {
	tests := []struct {
		name              string
		modelSpec         *internalmodels.ModelSpec
		expectConstraints bool
		expectMinVRAM     int
		expectClass       string
	}{
		{
			name:              "Nil model spec",
			modelSpec:         nil,
			expectConstraints: false,
		},
		{
			name: "Model with VRAM and GPU class",
			modelSpec: &internalmodels.ModelSpec{
				MinVRAMGB:         48,
				SuggestedGPUClass: "A100-80GB",
			},
			expectConstraints: true,
			expectMinVRAM:     48,
			expectClass:       "A100-80GB",
		},
		{
			name: "Model with only VRAM",
			modelSpec: &internalmodels.ModelSpec{
				MinVRAMGB: 16,
			},
			expectConstraints: true,
			expectMinVRAM:     16,
			expectClass:       "",
		},
		{
			name: "Model with no GPU requirements",
			modelSpec: &internalmodels.ModelSpec{
				MinVRAMGB: 0,
			},
			expectConstraints: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constraints := InferGPUConstraintsFromModel(tt.modelSpec)

			if tt.expectConstraints {
				if constraints == nil {
					t.Error("Expected constraints but got nil")
					return
				}
				if constraints.MinVRAMGB != tt.expectMinVRAM {
					t.Errorf("Expected MinVRAMGB=%d, got %d", tt.expectMinVRAM, constraints.MinVRAMGB)
				}
				if constraints.PreferredClass != tt.expectClass {
					t.Errorf("Expected PreferredClass=%s, got %s", tt.expectClass, constraints.PreferredClass)
				}
			} else {
				if constraints != nil {
					t.Errorf("Expected nil constraints but got %+v", constraints)
				}
			}
		})
	}
}
