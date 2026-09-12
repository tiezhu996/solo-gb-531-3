package algorithm
import (
	"fmt"
	"hazop-safeguard-coverage/backend/internal/dto"
	"math"
	"sort"
)
const coveredThreshold = 0.5
type ScoreResult struct {
	CoverageScore  float64
	Paths          []dto.CoveragePathResponse
	UncoveredPaths []dto.CoveragePathResponse
	Steps          []dto.ScoreStepResponse
	RiskBefore     string
	RiskAfter      string
}
func CalculateScore(graph Graph, scenario SnapshotScenario, safeguards []SnapshotSafeguard, rejected []RejectedSafeguard) ScoreResult {
	ordered := append([]SnapshotSafeguard(nil), safeguards...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].IndependenceKey == ordered[j].IndependenceKey {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].IndependenceKey < ordered[j].IndependenceKey
	})
	combined := 0.0
	remainingProbability := 1.0
	steps := make([]dto.ScoreStepResponse, 0, len(ordered)+len(rejected)+2)
	stepNumber := 1
	for _, safeguard := range ordered {
		before := 1 - remainingProbability
		remainingProbability *= 1 - safeguard.Effectiveness
		combined = 1 - remainingProbability
		steps = append(steps, dto.ScoreStepResponse{
			Step: stepNumber, Rule: "independent-layer-combination",
			Input:        fmt.Sprintf("safeguard=%d key=%s effectiveness=%.4f", safeguard.ID, safeguard.IndependenceKey, safeguard.Effectiveness),
			Contribution: round4(combined - before), RunningScore: round4(combined * 100),
			Explanation: "Independent effectiveness is combined as 1 - product(1 - effectiveness).",
		})
		stepNumber++
	}
	for _, item := range rejected {
		steps = append(steps, dto.ScoreStepResponse{
			Step: stepNumber, Rule: "eligibility-filter", Input: fmt.Sprintf("safeguard=%d", item.ID),
			Contribution: 0, RunningScore: round4(combined * 100), Explanation: item.Reason,
		})
		stepNumber++
	}
	paths := make([]dto.CoveragePathResponse, 0, len(graph.Paths))
	ids := make([]uint, 0, len(ordered))
	keys := make([]string, 0, len(ordered))
	for _, safeguard := range ordered {
		ids = append(ids, safeguard.ID)
		keys = append(keys, safeguard.IndependenceKey)
	}
	covered := combined >= coveredThreshold
	for _, path := range graph.Paths {
		reason := fmt.Sprintf("combined independent protection %.2f%% meets %.0f%% threshold", combined*100, coveredThreshold*100)
		if !covered {
			reason = fmt.Sprintf("combined independent protection %.2f%% is below %.0f%% threshold", combined*100, coveredThreshold*100)
		}
		pathResult := dto.CoveragePathResponse{
			PathID: path.ID, NodeCode: path.NodeCode, Cause: path.Cause, Consequence: path.Consequence,
			SafeguardIDs: append([]uint(nil), ids...), IndependenceKeys: append([]string(nil), keys...),
			CombinedProtection: round4(combined), Covered: covered, Reason: reason,
		}
		paths = append(paths, pathResult)
	}
	uncovered := make([]dto.CoveragePathResponse, 0)
	for _, path := range paths {
		if !path.Covered {
			uncovered = append(uncovered, path)
		}
	}
	coverageScore := round4(combined * 100)
	initialRisk := scenario.Likelihood * scenario.Severity
	residualRisk := int(math.Ceil(float64(initialRisk) * (1 - combined)))
	if residualRisk < 1 {
		residualRisk = 1
	}
	steps = append(steps, dto.ScoreStepResponse{
		Step: stepNumber, Rule: "residual-risk", Input: fmt.Sprintf("initial=%d protection=%.4f", initialRisk, combined),
		Contribution: 0, RunningScore: coverageScore,
		Explanation: fmt.Sprintf("Residual risk index is ceil(%d * (1 - %.4f)) = %d.", initialRisk, combined, residualRisk),
	})
	return ScoreResult{
		CoverageScore: coverageScore, Paths: paths, UncoveredPaths: uncovered, Steps: steps,
		RiskBefore: dto.RiskRank(initialRisk), RiskAfter: dto.RiskRank(residualRisk),
	}
}
func round4(value float64) float64 { return math.Round(value*10000) / 10000 }
