package constants

type CoverageState string

const (
	CoverageQueued    CoverageState = "queued"
	CoverageRunning   CoverageState = "running"
	CoverageCompleted CoverageState = "completed"
	CoverageFailed    CoverageState = "failed"
	CoverageConfirmed CoverageState = "confirmed"
	CoverageVoided    CoverageState = "voided"
)

var coverageTransitions = map[CoverageState]map[CoverageState]struct{}{
	CoverageQueued:    {CoverageRunning: {}},
	CoverageRunning:   {CoverageCompleted: {}, CoverageFailed: {}},
	CoverageCompleted: {CoverageConfirmed: {}, CoverageVoided: {}},
	CoverageFailed:    {CoverageVoided: {}},
	CoverageConfirmed: {CoverageVoided: {}},
	CoverageVoided:    {},
}

func (s CoverageState) Valid() bool {
	_, ok := coverageTransitions[s]
	return ok
}

func CanTransitionCoverage(from, to CoverageState) bool {
	allowed, ok := coverageTransitions[from]
	if !ok {
		return false
	}
	_, ok = allowed[to]
	return ok
}

func CoverageStateValues() []string {
	return []string{"queued", "running", "completed", "failed", "confirmed", "voided"}
}
