package algorithm
import (
	"fmt"
	"hazop-safeguard-coverage/backend/internal/dto"
	"sort"
	"strings"
	"time"
)
type IndependenceResult struct {
	Retained     []SnapshotSafeguard
	Deduplicated []dto.DeduplicatedSafeguardResponse
	Rejected     []RejectedSafeguard
}
type RejectedSafeguard struct {
	ID     uint   `json:"id"`
	Reason string `json:"reason"`
}
func ResolveIndependence(safeguards []SnapshotSafeguard, referenceTime time.Time) IndependenceResult {
	groups := make(map[string][]SnapshotSafeguard)
	result := IndependenceResult{}
	for _, safeguard := range safeguards {
		key := strings.ToUpper(strings.TrimSpace(safeguard.IndependenceKey))
		if key == "" {
			result.Rejected = append(result.Rejected, RejectedSafeguard{ID: safeguard.ID, Reason: "missing independence key"})
			continue
		}
		if reason := ineligibleReason(safeguard, referenceTime); reason != "" {
			result.Rejected = append(result.Rejected, RejectedSafeguard{ID: safeguard.ID, Reason: reason})
			continue
		}
		groups[key] = append(groups[key], safeguard)
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		items := groups[key]
		sort.Slice(items, func(i, j int) bool {
			if items[i].Effectiveness == items[j].Effectiveness {
				return items[i].ID < items[j].ID
			}
			return items[i].Effectiveness > items[j].Effectiveness
		})
		result.Retained = append(result.Retained, items[0])
		if len(items) > 1 {
			ignored := make([]uint, 0, len(items)-1)
			for _, item := range items[1:] {
				ignored = append(ignored, item.ID)
			}
			result.Deduplicated = append(result.Deduplicated, dto.DeduplicatedSafeguardResponse{
				IndependenceKey: key, KeptID: items[0].ID, IgnoredIDs: ignored,
				Reason: "same independence_key on one cause-to-consequence path; highest effectiveness retained, then lowest ID",
			})
		}
	}
	sort.Slice(result.Rejected, func(i, j int) bool { return result.Rejected[i].ID < result.Rejected[j].ID })
	return result
}
func ineligibleReason(safeguard SnapshotSafeguard, referenceTime time.Time) string {
	if safeguard.LifecycleState != "active" {
		return fmt.Sprintf("lifecycle state %q is not active", safeguard.LifecycleState)
	}
	if safeguard.Effectiveness <= 0 || safeguard.Effectiveness > 1 {
		return "effectiveness is outside (0,1]"
	}
	if safeguard.LastVerifiedAt == nil {
		return "verification evidence is missing"
	}
	if safeguard.TestIntervalDays <= 0 {
		return "test interval is invalid"
	}
	expires := safeguard.LastVerifiedAt.AddDate(0, 0, safeguard.TestIntervalDays)
	if referenceTime.After(expires) {
		return fmt.Sprintf("verification expired at %s", expires.UTC().Format(time.RFC3339))
	}
	return ""
}
