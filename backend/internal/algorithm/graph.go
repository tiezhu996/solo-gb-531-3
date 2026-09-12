package algorithm
import (
	"fmt"
	"hazop-safeguard-coverage/backend/internal/model"
	"sort"
	"strings"
	"time"
)
const Version = "hazop-cover-v1.0.0"
type Snapshot struct {
	AlgorithmVersion string              `json:"algorithm_version"`
	ReferenceTime    time.Time           `json:"reference_time"`
	Node             SnapshotNode        `json:"node"`
	Scenario         SnapshotScenario    `json:"scenario"`
	Safeguards       []SnapshotSafeguard `json:"safeguards"`
}
type SnapshotNode struct {
	ID                uint    `json:"id"`
	NodeCode          string  `json:"node_code"`
	Name              string  `json:"name"`
	UnitName          string  `json:"unit_name"`
	Medium            string  `json:"medium"`
	DesignPressure    float64 `json:"design_pressure"`
	DesignTemperature float64 `json:"design_temperature"`
}
type SnapshotScenario struct {
	ID            uint   `json:"id"`
	Guideword     string `json:"guideword"`
	Parameter     string `json:"parameter"`
	Cause         string `json:"cause"`
	Consequence   string `json:"consequence"`
	Likelihood    int    `json:"likelihood"`
	Severity      int    `json:"severity"`
	ScenarioState string `json:"scenario_state"`
	Version       int    `json:"version"`
}
type SnapshotSafeguard struct {
	ID               uint       `json:"id"`
	Name             string     `json:"name"`
	Type             string     `json:"type"`
	IndependenceKey  string     `json:"independence_key"`
	Effectiveness    float64    `json:"effectiveness"`
	TestIntervalDays int        `json:"test_interval_days"`
	LastVerifiedAt   *time.Time `json:"last_verified_at"`
	LifecycleState   string     `json:"lifecycle_state"`
	EvidenceNote     string     `json:"evidence_note"`
}
type Graph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
	Paths []GraphPath `json:"paths"`
}
type GraphNode struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
}
type GraphEdge struct {
	From          string  `json:"from"`
	To            string  `json:"to"`
	Relation      string  `json:"relation"`
	SafeguardID   uint    `json:"safeguard_id,omitempty"`
	Effectiveness float64 `json:"effectiveness,omitempty"`
}
type GraphPath struct {
	ID          string `json:"id"`
	NodeCode    string `json:"node_code"`
	Cause       string `json:"cause"`
	Consequence string `json:"consequence"`
}
func NewSnapshot(node model.ProcessNode, scenario model.DeviationScenario, safeguards []model.Safeguard, reference time.Time) Snapshot {
	ordered := append([]model.Safeguard(nil), safeguards...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].IndependenceKey == ordered[j].IndependenceKey {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].IndependenceKey < ordered[j].IndependenceKey
	})
	snapshot := Snapshot{
		AlgorithmVersion: Version,
		ReferenceTime:    reference.UTC().Truncate(time.Second),
		Node: SnapshotNode{
			ID: node.ID, NodeCode: node.NodeCode, Name: node.Name, UnitName: node.UnitName,
			Medium: node.Medium, DesignPressure: node.DesignPressure, DesignTemperature: node.DesignTemperature,
		},
		Scenario: SnapshotScenario{
			ID: scenario.ID, Guideword: scenario.Guideword, Parameter: scenario.Parameter,
			Cause: scenario.Cause, Consequence: scenario.Consequence,
			Likelihood: scenario.Likelihood, Severity: scenario.Severity,
			ScenarioState: scenario.ScenarioState, Version: scenario.Version,
		},
	}
	for _, item := range ordered {
		var verified *time.Time
		if item.LastVerifiedAt != nil {
			value := item.LastVerifiedAt.UTC().Truncate(time.Second)
			verified = &value
		}
		snapshot.Safeguards = append(snapshot.Safeguards, SnapshotSafeguard{
			ID: item.ID, Name: item.Name, Type: item.SafeguardType,
			IndependenceKey: item.IndependenceKey, Effectiveness: item.Effectiveness,
			TestIntervalDays: item.TestIntervalDays, LastVerifiedAt: verified,
			LifecycleState: item.LifecycleState, EvidenceNote: item.EvidenceNote,
		})
	}
	return snapshot
}
func BuildGraph(snapshot Snapshot) Graph {
	causes := splitStatements(snapshot.Scenario.Cause)
	consequences := splitStatements(snapshot.Scenario.Consequence)
	graph := Graph{}
	graph.Nodes = append(graph.Nodes, GraphNode{
		ID: fmt.Sprintf("node-%d", snapshot.Node.ID), Kind: "process_node", Label: snapshot.Node.NodeCode,
	})
	for index, cause := range causes {
		causeID := fmt.Sprintf("cause-%02d", index+1)
		graph.Nodes = append(graph.Nodes, GraphNode{ID: causeID, Kind: "cause", Label: cause})
		graph.Edges = append(graph.Edges, GraphEdge{From: causeID, To: fmt.Sprintf("node-%d", snapshot.Node.ID), Relation: "initiates"})
	}
	for index, consequence := range consequences {
		consequenceID := fmt.Sprintf("consequence-%02d", index+1)
		graph.Nodes = append(graph.Nodes, GraphNode{ID: consequenceID, Kind: "consequence", Label: consequence})
		graph.Edges = append(graph.Edges, GraphEdge{From: fmt.Sprintf("node-%d", snapshot.Node.ID), To: consequenceID, Relation: "may_lead_to"})
	}
	for _, safeguard := range snapshot.Safeguards {
		safeguardID := fmt.Sprintf("safeguard-%d", safeguard.ID)
		graph.Nodes = append(graph.Nodes, GraphNode{ID: safeguardID, Kind: "safeguard", Label: safeguard.Name})
		for index := range consequences {
			graph.Edges = append(graph.Edges, GraphEdge{
				From: safeguardID, To: fmt.Sprintf("consequence-%02d", index+1),
				Relation: "mitigates", SafeguardID: safeguard.ID, Effectiveness: safeguard.Effectiveness,
			})
		}
	}
	pathIndex := 1
	for _, cause := range causes {
		for _, consequence := range consequences {
			graph.Paths = append(graph.Paths, GraphPath{
				ID: fmt.Sprintf("P-%03d", pathIndex), NodeCode: snapshot.Node.NodeCode,
				Cause: cause, Consequence: consequence,
			})
			pathIndex++
		}
	}
	return graph
}
func splitStatements(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ';' || r == '\n' || r == '|' || r == '。'
	})
	clean := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := strings.ToLower(part)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		clean = append(clean, part)
	}
	if len(clean) == 0 {
		return []string{"unspecified"}
	}
	return clean
}
