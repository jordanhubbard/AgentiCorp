package workflow

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// WorkflowDefinition represents a workflow definition from YAML
type WorkflowDefinition struct {
	ID           string                   `yaml:"id"`
	Name         string                   `yaml:"name"`
	Description  string                   `yaml:"description"`
	WorkflowType string                   `yaml:"workflow_type"`
	IsDefault    bool                     `yaml:"is_default"`
	Nodes        []WorkflowNodeDefinition `yaml:"nodes"`
	Edges        []WorkflowEdgeDefinition `yaml:"edges"`
}

// WorkflowNodeDefinition represents a node definition from YAML
type WorkflowNodeDefinition struct {
	NodeKey        string            `yaml:"node_key"`
	NodeType       string            `yaml:"node_type"`
	RoleRequired   string            `yaml:"role_required"`
	PersonaHint    string            `yaml:"persona_hint"`
	MaxAttempts    int               `yaml:"max_attempts"`
	TimeoutMinutes int               `yaml:"timeout_minutes"`
	Instructions   string            `yaml:"instructions"`
	Metadata       map[string]string `yaml:"metadata,omitempty"`
}

// WorkflowEdgeDefinition represents an edge definition from YAML
type WorkflowEdgeDefinition struct {
	FromNodeKey string `yaml:"from_node_key"`
	ToNodeKey   string `yaml:"to_node_key"`
	Condition   string `yaml:"condition"`
	Priority    int    `yaml:"priority"`
}

// LoadWorkflowFromFile loads a workflow definition from a YAML file
func LoadWorkflowFromFile(filepath string) (*Workflow, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	var def WorkflowDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	return convertDefinitionToWorkflow(&def), nil
}

// LoadDefaultWorkflows loads all default workflow definitions from a directory
func LoadDefaultWorkflows(dir string) ([]*Workflow, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}

	var workflows []*Workflow
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".yaml" {
			continue
		}

		path := filepath.Join(dir, file.Name())
		wf, err := LoadWorkflowFromFile(path)
		if err != nil {
			log.Printf("[Workflow] Warning: failed to load %s: %v", file.Name(), err)
			continue
		}

		workflows = append(workflows, wf)
		log.Printf("[Workflow] Loaded workflow: %s (%s)", wf.Name, wf.ID)
	}

	return workflows, nil
}

// convertDefinitionToWorkflow converts a YAML definition to a Workflow model
func convertDefinitionToWorkflow(def *WorkflowDefinition) *Workflow {
	now := time.Now()
	wf := &Workflow{
		ID:           def.ID,
		Name:         def.Name,
		Description:  def.Description,
		WorkflowType: def.WorkflowType,
		IsDefault:    def.IsDefault,
		ProjectID:    "", // Empty for global defaults
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Convert nodes
	for _, nodeDef := range def.Nodes {
		node := WorkflowNode{
			ID:             fmt.Sprintf("wfn-%s", uuid.New().String()[:8]),
			WorkflowID:     wf.ID,
			NodeKey:        nodeDef.NodeKey,
			NodeType:       NodeType(nodeDef.NodeType),
			RoleRequired:   nodeDef.RoleRequired,
			PersonaHint:    nodeDef.PersonaHint,
			MaxAttempts:    nodeDef.MaxAttempts,
			TimeoutMinutes: nodeDef.TimeoutMinutes,
			Instructions:   nodeDef.Instructions,
			Metadata:       nodeDef.Metadata,
			CreatedAt:      now,
		}
		if node.Metadata == nil {
			node.Metadata = map[string]string{}
		}
		wf.Nodes = append(wf.Nodes, node)
	}

	// Convert edges
	for _, edgeDef := range def.Edges {
		edge := WorkflowEdge{
			ID:          fmt.Sprintf("wfe-%s", uuid.New().String()[:8]),
			WorkflowID:  wf.ID,
			FromNodeKey: edgeDef.FromNodeKey,
			ToNodeKey:   edgeDef.ToNodeKey,
			Condition:   EdgeCondition(edgeDef.Condition),
			Priority:    edgeDef.Priority,
			CreatedAt:   now,
		}
		wf.Edges = append(wf.Edges, edge)
	}

	return wf
}

// InstallDefaultWorkflows loads and installs default workflows into the database
func InstallDefaultWorkflows(db Database, workflowsDir string) error {
	workflows, err := LoadDefaultWorkflows(workflowsDir)
	if err != nil {
		return fmt.Errorf("failed to load default workflows: %w", err)
	}

	for _, wf := range workflows {
		// Insert workflow
		if err := db.UpsertWorkflow(wf); err != nil {
			log.Printf("[Workflow] Warning: failed to upsert workflow %s: %v", wf.ID, err)
			continue
		}

		// Insert nodes
		for _, node := range wf.Nodes {
			if err := db.UpsertWorkflowNode(&node); err != nil {
				log.Printf("[Workflow] Warning: failed to upsert node %s: %v", node.NodeKey, err)
			}
		}

		// Insert edges
		for _, edge := range wf.Edges {
			if err := db.UpsertWorkflowEdge(&edge); err != nil {
				log.Printf("[Workflow] Warning: failed to upsert edge: %v", err)
			}
		}

		log.Printf("[Workflow] Installed default workflow: %s", wf.Name)
	}

	return nil
}
