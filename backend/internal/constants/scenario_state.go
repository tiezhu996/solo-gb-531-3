package constants

type ScenarioState string

const (
	ScenarioDraft    ScenarioState = "draft"
	ScenarioAnalyzed ScenarioState = "analyzed"
	ScenarioVerified ScenarioState = "verified"
	ScenarioAccepted ScenarioState = "accepted"
	ScenarioRework   ScenarioState = "rework"
)

var scenarioTransitions = map[ScenarioState]map[ScenarioState]struct{}{
	ScenarioDraft:    {ScenarioAnalyzed: {}},
	ScenarioAnalyzed: {ScenarioVerified: {}, ScenarioRework: {}},
	ScenarioVerified: {ScenarioAccepted: {}, ScenarioRework: {}},
	ScenarioAccepted: {},
	ScenarioRework:   {ScenarioAnalyzed: {}},
}

func (s ScenarioState) Valid() bool {
	_, ok := scenarioTransitions[s]
	return ok
}

func CanTransitionScenario(from, to ScenarioState) bool {
	allowed, ok := scenarioTransitions[from]
	if !ok {
		return false
	}
	_, ok = allowed[to]
	return ok
}

func ScenarioStateValues() []string {
	return []string{"draft", "analyzed", "verified", "accepted", "rework"}
}
