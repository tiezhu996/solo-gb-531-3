package constants

// DeviationGuideword identifies the standard HAZOP deviation prompt.
type DeviationGuideword string

const (
	GuidewordNo      DeviationGuideword = "no"
	GuidewordMore    DeviationGuideword = "more"
	GuidewordLess    DeviationGuideword = "less"
	GuidewordReverse DeviationGuideword = "reverse"
	GuidewordOther   DeviationGuideword = "other"
)

var deviationGuidewords = map[DeviationGuideword]struct{}{
	GuidewordNo: {}, GuidewordMore: {}, GuidewordLess: {},
	GuidewordReverse: {}, GuidewordOther: {},
}

func (g DeviationGuideword) Valid() bool {
	_, ok := deviationGuidewords[g]
	return ok
}

func DeviationGuidewordValues() []string {
	return []string{string(GuidewordNo), string(GuidewordMore), string(GuidewordLess), string(GuidewordReverse), string(GuidewordOther)}
}
