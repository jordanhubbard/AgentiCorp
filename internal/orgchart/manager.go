package orgchart

import (
	"fmt"
	"sync"
	"time"

	"github.com/jordanhubbard/loom/pkg/models"
)

// Manager manages org charts for projects
type Manager struct {
	charts   map[string]*models.OrgChart // key: project ID or template ID
	template *models.OrgChart            // default template
	mu       sync.RWMutex
}

// NewManager creates a new org chart manager with the default template
func NewManager() *Manager {
	m := &Manager{
		charts: make(map[string]*models.OrgChart),
	}
	m.template = m.createDefaultTemplate()
	return m
}

// createDefaultTemplate builds the system-wide default org chart template
func (m *Manager) createDefaultTemplate() *models.OrgChart {
	return &models.OrgChart{
		ID:         "orgchart-default-template",
		ProjectID:  "",
		Name:       "Default",
		Positions:  models.DefaultOrgChartPositions(),
		IsTemplate: true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// GetDefaultTemplate returns the default org chart template
func (m *Manager) GetDefaultTemplate() *models.OrgChart {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.template
}

// CreateForProject creates a new org chart for a project, cloned from the default template
func (m *Manager) CreateForProject(projectID, projectName string) (*models.OrgChart, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if projectID == "" {
		return nil, fmt.Errorf("project ID is required")
	}

	// Check if already exists
	if existing, ok := m.charts[projectID]; ok {
		return existing, nil
	}

	// Clone positions from template
	positions := make([]models.Position, len(m.template.Positions))
	for i, p := range m.template.Positions {
		positions[i] = models.Position{
			ID:           p.ID,
			RoleName:     p.RoleName,
			PersonaPath:  p.PersonaPath,
			Required:     p.Required,
			MaxInstances: p.MaxInstances,
			AgentIDs:     []string{}, // Start empty
			ReportsTo:    p.ReportsTo,
		}
	}

	chart := &models.OrgChart{
		ID:         fmt.Sprintf("orgchart-%s", projectID),
		ProjectID:  projectID,
		Name:       fmt.Sprintf("%s Org Chart", projectName),
		Positions:  positions,
		IsTemplate: false,
		ParentID:   m.template.ID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	m.charts[projectID] = chart
	return chart, nil
}

// GetByProject retrieves the org chart for a project
func (m *Manager) GetByProject(projectID string) (*models.OrgChart, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chart, ok := m.charts[projectID]
	if !ok {
		return nil, fmt.Errorf("org chart not found for project: %s", projectID)
	}
	return chart, nil
}

// AssignAgent assigns an agent to a position in a project's org chart
func (m *Manager) AssignAgent(projectID, positionID, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	chart, ok := m.charts[projectID]
	if !ok {
		return fmt.Errorf("org chart not found for project: %s", projectID)
	}

	pos := chart.GetPositionByID(positionID)
	if pos == nil {
		return fmt.Errorf("position not found: %s", positionID)
	}

	if pos.HasAgent(agentID) {
		return nil // Already assigned
	}

	if !pos.CanAddAgent() {
		return fmt.Errorf("position %s is at max capacity", pos.RoleName)
	}

	pos.AgentIDs = append(pos.AgentIDs, agentID)
	chart.UpdatedAt = time.Now()
	return nil
}

// AssignAgentToRole assigns an agent to a position by role name
func (m *Manager) AssignAgentToRole(projectID, roleName, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	chart, ok := m.charts[projectID]
	if !ok {
		return fmt.Errorf("org chart not found for project: %s", projectID)
	}

	pos := chart.GetPositionByRole(roleName)
	if pos == nil {
		return fmt.Errorf("position not found for role: %s", roleName)
	}

	if pos.HasAgent(agentID) {
		return nil // Already assigned
	}

	if !pos.CanAddAgent() {
		return fmt.Errorf("position %s is at max capacity", pos.RoleName)
	}

	pos.AgentIDs = append(pos.AgentIDs, agentID)
	chart.UpdatedAt = time.Now()
	return nil
}

// UnassignAgent removes an agent from a position
func (m *Manager) UnassignAgent(projectID, positionID, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	chart, ok := m.charts[projectID]
	if !ok {
		return fmt.Errorf("org chart not found for project: %s", projectID)
	}

	pos := chart.GetPositionByID(positionID)
	if pos == nil {
		return fmt.Errorf("position not found: %s", positionID)
	}

	// Remove agent from position
	for i, id := range pos.AgentIDs {
		if id == agentID {
			pos.AgentIDs = append(pos.AgentIDs[:i], pos.AgentIDs[i+1:]...)
			chart.UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("agent %s not found in position %s", agentID, positionID)
}

// RemoveAgentFromAll removes an agent from all positions in a project's org chart
func (m *Manager) RemoveAgentFromAll(projectID, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	chart, ok := m.charts[projectID]
	if !ok {
		return fmt.Errorf("org chart not found for project: %s", projectID)
	}

	changed := false
	for i := range chart.Positions {
		for j, id := range chart.Positions[i].AgentIDs {
			if id == agentID {
				chart.Positions[i].AgentIDs = append(
					chart.Positions[i].AgentIDs[:j],
					chart.Positions[i].AgentIDs[j+1:]...,
				)
				changed = true
				break
			}
		}
	}

	if changed {
		chart.UpdatedAt = time.Now()
	}
	return nil
}

// GetPositionsForAgent returns all positions an agent is assigned to
func (m *Manager) GetPositionsForAgent(projectID, agentID string) []models.Position {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chart, ok := m.charts[projectID]
	if !ok {
		return nil
	}

	var positions []models.Position
	for _, p := range chart.Positions {
		if p.HasAgent(agentID) {
			positions = append(positions, p)
		}
	}
	return positions
}

// AddPosition adds a new position to a project's org chart
func (m *Manager) AddPosition(projectID string, position models.Position) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	chart, ok := m.charts[projectID]
	if !ok {
		return fmt.Errorf("org chart not found for project: %s", projectID)
	}

	// Check for duplicate ID or role name
	for _, p := range chart.Positions {
		if p.ID == position.ID {
			return fmt.Errorf("position ID already exists: %s", position.ID)
		}
		if p.RoleName == position.RoleName {
			return fmt.Errorf("position role already exists: %s", position.RoleName)
		}
	}

	if position.AgentIDs == nil {
		position.AgentIDs = []string{}
	}

	chart.Positions = append(chart.Positions, position)
	chart.UpdatedAt = time.Now()
	return nil
}

// RemovePosition removes a position from a project's org chart
func (m *Manager) RemovePosition(projectID, positionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	chart, ok := m.charts[projectID]
	if !ok {
		return fmt.Errorf("org chart not found for project: %s", projectID)
	}

	for i, p := range chart.Positions {
		if p.ID == positionID {
			chart.Positions = append(chart.Positions[:i], chart.Positions[i+1:]...)
			chart.UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("position not found: %s", positionID)
}

// DeleteForProject removes the org chart for a project
func (m *Manager) DeleteForProject(projectID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.charts[projectID]; !ok {
		return fmt.Errorf("org chart not found for project: %s", projectID)
	}

	delete(m.charts, projectID)
	return nil
}

// ListAll returns all org charts (excluding template)
func (m *Manager) ListAll() []*models.OrgChart {
	m.mu.RLock()
	defer m.mu.RUnlock()

	charts := make([]*models.OrgChart, 0, len(m.charts))
	for _, c := range m.charts {
		charts = append(charts, c)
	}
	return charts
}

// ImportFromLegacy creates an org chart from an existing project's agent list
// This is used for migration from the old flat agent list model
func (m *Manager) ImportFromLegacy(projectID, projectName string, agentRoles map[string]string) (*models.OrgChart, error) {
	chart, err := m.CreateForProject(projectID, projectName)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Map agents to positions based on their role/persona
	for agentID, roleName := range agentRoles {
		pos := chart.GetPositionByRole(roleName)
		if pos != nil && pos.CanAddAgent() {
			pos.AgentIDs = append(pos.AgentIDs, agentID)
		}
	}

	chart.UpdatedAt = time.Now()
	return chart, nil
}
