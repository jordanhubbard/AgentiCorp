package orgchart

import (
	"testing"

	"github.com/jordanhubbard/loom/pkg/models"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager returned nil")
	}

	template := m.GetDefaultTemplate()
	if template == nil {
		t.Fatal("Default template is nil")
	}
	if !template.IsTemplate {
		t.Error("Default template should have IsTemplate=true")
	}
	if len(template.Positions) == 0 {
		t.Error("Default template should have positions")
	}
}

func TestCreateForProject(t *testing.T) {
	m := NewManager()

	chart, err := m.CreateForProject("proj-123", "Test Project")
	if err != nil {
		t.Fatalf("CreateForProject failed: %v", err)
	}

	if chart.ProjectID != "proj-123" {
		t.Errorf("Expected project ID 'proj-123', got '%s'", chart.ProjectID)
	}
	if chart.IsTemplate {
		t.Error("Project org chart should not be a template")
	}
	if len(chart.Positions) == 0 {
		t.Error("Project org chart should have positions from template")
	}

	// Verify positions are cloned (not shared)
	template := m.GetDefaultTemplate()
	if len(chart.Positions) != len(template.Positions) {
		t.Error("Project org chart should have same number of positions as template")
	}

	// All positions should start empty
	for _, pos := range chart.Positions {
		if len(pos.AgentIDs) != 0 {
			t.Errorf("Position %s should start with no agents", pos.RoleName)
		}
	}
}

func TestCreateForProjectEmpty(t *testing.T) {
	m := NewManager()

	_, err := m.CreateForProject("", "Test")
	if err == nil {
		t.Error("CreateForProject with empty project ID should fail")
	}
}

func TestCreateForProjectIdempotent(t *testing.T) {
	m := NewManager()

	chart1, _ := m.CreateForProject("proj-123", "Test Project")
	chart2, _ := m.CreateForProject("proj-123", "Test Project")

	if chart1.ID != chart2.ID {
		t.Error("Creating same project twice should return same org chart")
	}
}

func TestAssignAgent(t *testing.T) {
	m := NewManager()
	if _, err := m.CreateForProject("proj-123", "Test"); err != nil {
		t.Fatalf("CreateForProject failed: %v", err)
	}

	err := m.AssignAgent("proj-123", "pos-ceo", "agent-1")
	if err != nil {
		t.Fatalf("AssignAgent failed: %v", err)
	}

	chart, _ := m.GetByProject("proj-123")
	pos := chart.GetPositionByID("pos-ceo")
	if !pos.HasAgent("agent-1") {
		t.Error("Agent should be assigned to position")
	}
}

func TestAssignAgentMaxCapacity(t *testing.T) {
	m := NewManager()
	if _, err := m.CreateForProject("proj-123", "Test"); err != nil {
		t.Fatalf("CreateForProject failed: %v", err)
	}

	// CEO position has MaxInstances=1
	_ = m.AssignAgent("proj-123", "pos-ceo", "agent-1")
	err := m.AssignAgent("proj-123", "pos-ceo", "agent-2")
	if err == nil {
		t.Error("Assigning second agent to CEO position should fail")
	}
}

func TestAssignAgentToRole(t *testing.T) {
	m := NewManager()
	if _, err := m.CreateForProject("proj-123", "Test"); err != nil {
		t.Fatalf("CreateForProject failed: %v", err)
	}

	err := m.AssignAgentToRole("proj-123", "product-manager", "agent-1")
	if err != nil {
		t.Fatalf("AssignAgentToRole failed: %v", err)
	}

	chart, _ := m.GetByProject("proj-123")
	pos := chart.GetPositionByRole("product-manager")
	if !pos.HasAgent("agent-1") {
		t.Error("Agent should be assigned to product-manager position")
	}
}

func TestUnassignAgent(t *testing.T) {
	m := NewManager()
	if _, err := m.CreateForProject("proj-123", "Test"); err != nil {
		t.Fatalf("CreateForProject failed: %v", err)
	}
	if err := m.AssignAgent("proj-123", "pos-ceo", "agent-1"); err != nil {
		t.Fatalf("AssignAgent failed: %v", err)
	}

	err := m.UnassignAgent("proj-123", "pos-ceo", "agent-1")
	if err != nil {
		t.Fatalf("UnassignAgent failed: %v", err)
	}

	chart, _ := m.GetByProject("proj-123")
	pos := chart.GetPositionByID("pos-ceo")
	if pos.HasAgent("agent-1") {
		t.Error("Agent should be unassigned from position")
	}
}

func TestRemoveAgentFromAll(t *testing.T) {
	m := NewManager()
	if _, err := m.CreateForProject("proj-123", "Test"); err != nil {
		t.Fatalf("CreateForProject failed: %v", err)
	}

	// Assign agent to multiple positions (PM has unlimited capacity)
	if err := m.AssignAgentToRole("proj-123", "product-manager", "agent-1"); err != nil {
		t.Fatalf("AssignAgentToRole failed: %v", err)
	}
	if err := m.AssignAgentToRole("proj-123", "qa-engineer", "agent-1"); err != nil {
		t.Fatalf("AssignAgentToRole failed: %v", err)
	}

	err := m.RemoveAgentFromAll("proj-123", "agent-1")
	if err != nil {
		t.Fatalf("RemoveAgentFromAll failed: %v", err)
	}

	chart, _ := m.GetByProject("proj-123")
	for _, pos := range chart.Positions {
		if pos.HasAgent("agent-1") {
			t.Errorf("Agent should be removed from position %s", pos.RoleName)
		}
	}
}

func TestGetPositionsForAgent(t *testing.T) {
	m := NewManager()
	if _, err := m.CreateForProject("proj-123", "Test"); err != nil {
		t.Fatalf("CreateForProject failed: %v", err)
	}
	if err := m.AssignAgentToRole("proj-123", "product-manager", "agent-1"); err != nil {
		t.Fatalf("AssignAgentToRole failed: %v", err)
	}
	if err := m.AssignAgentToRole("proj-123", "qa-engineer", "agent-1"); err != nil {
		t.Fatalf("AssignAgentToRole failed: %v", err)
	}

	positions := m.GetPositionsForAgent("proj-123", "agent-1")
	if len(positions) != 2 {
		t.Errorf("Expected 2 positions, got %d", len(positions))
	}
}

func TestAddPosition(t *testing.T) {
	m := NewManager()
	if _, err := m.CreateForProject("proj-123", "Test"); err != nil {
		t.Fatalf("CreateForProject failed: %v", err)
	}

	newPos := models.Position{
		ID:           "pos-custom",
		RoleName:     "custom-role",
		PersonaPath:  "custom/role",
		Required:     false,
		MaxInstances: 2,
	}

	err := m.AddPosition("proj-123", newPos)
	if err != nil {
		t.Fatalf("AddPosition failed: %v", err)
	}

	chart, _ := m.GetByProject("proj-123")
	pos := chart.GetPositionByRole("custom-role")
	if pos == nil {
		t.Error("Custom position should exist")
	}
}

func TestRemovePosition(t *testing.T) {
	m := NewManager()
	if _, err := m.CreateForProject("proj-123", "Test"); err != nil {
		t.Fatalf("CreateForProject failed: %v", err)
	}

	err := m.RemovePosition("proj-123", "pos-hk")
	if err != nil {
		t.Fatalf("RemovePosition failed: %v", err)
	}

	chart, _ := m.GetByProject("proj-123")
	pos := chart.GetPositionByID("pos-hk")
	if pos != nil {
		t.Error("Position should be removed")
	}
}

func TestDeleteForProject(t *testing.T) {
	m := NewManager()
	if _, err := m.CreateForProject("proj-123", "Test"); err != nil {
		t.Fatalf("CreateForProject failed: %v", err)
	}

	err := m.DeleteForProject("proj-123")
	if err != nil {
		t.Fatalf("DeleteForProject failed: %v", err)
	}

	_, err = m.GetByProject("proj-123")
	if err == nil {
		t.Error("GetByProject should fail after deletion")
	}
}

func TestOrgChartGetRequiredPositions(t *testing.T) {
	m := NewManager()
	chart, _ := m.CreateForProject("proj-123", "Test")

	required := chart.GetRequiredPositions()
	if len(required) == 0 {
		t.Error("Should have required positions")
	}

	for _, pos := range required {
		if !pos.Required {
			t.Errorf("Position %s should be required", pos.RoleName)
		}
	}
}

func TestOrgChartAllRequiredFilled(t *testing.T) {
	m := NewManager()
	chart, _ := m.CreateForProject("proj-123", "Test")

	if chart.AllRequiredFilled() {
		t.Error("No positions are filled yet, should return false")
	}

	// Fill all required positions
	for _, pos := range chart.GetRequiredPositions() {
		if err := m.AssignAgent("proj-123", pos.ID, "agent-"+pos.RoleName); err != nil {
			t.Fatalf("AssignAgent failed: %v", err)
		}
	}

	chart, _ = m.GetByProject("proj-123")
	if !chart.AllRequiredFilled() {
		t.Error("All required positions are filled, should return true")
	}
}

func TestOrgChartGetAllAgentIDs(t *testing.T) {
	m := NewManager()
	if _, err := m.CreateForProject("proj-123", "Test"); err != nil {
		t.Fatalf("CreateForProject failed: %v", err)
	}
	if err := m.AssignAgentToRole("proj-123", "ceo", "agent-1"); err != nil {
		t.Fatalf("AssignAgentToRole failed: %v", err)
	}
	if err := m.AssignAgentToRole("proj-123", "product-manager", "agent-2"); err != nil {
		t.Fatalf("AssignAgentToRole failed: %v", err)
	}
	if err := m.AssignAgentToRole("proj-123", "qa-engineer", "agent-2"); err != nil { // Same agent in two roles
		t.Fatalf("AssignAgentToRole failed: %v", err)
	}

	chart, _ := m.GetByProject("proj-123")
	agents := chart.GetAllAgentIDs()

	if len(agents) != 2 {
		t.Errorf("Expected 2 unique agents, got %d", len(agents))
	}
}

func TestPositionStatus(t *testing.T) {
	pos := models.Position{
		ID:           "pos-test",
		RoleName:     "test",
		MaxInstances: 2,
		AgentIDs:     []string{},
	}

	if pos.Status() != "vacant" {
		t.Errorf("Expected 'vacant', got '%s'", pos.Status())
	}

	pos.AgentIDs = []string{"agent-1"}
	if pos.Status() != "partial" {
		t.Errorf("Expected 'partial', got '%s'", pos.Status())
	}

	pos.AgentIDs = []string{"agent-1", "agent-2"}
	if pos.Status() != "filled" {
		t.Errorf("Expected 'filled', got '%s'", pos.Status())
	}
}
