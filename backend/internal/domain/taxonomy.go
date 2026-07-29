package domain

import "strings"

type TagDefinition struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

const (
	TopicTagPhysicsTrack    = "物理方向"
	TopicTagHistoryTrack    = "历史方向"
	TopicTagChemistry       = "化学重要性"
	TopicTagSelectionTiming = "选科时间线"
	TopicTagScoreGrowth     = "提分方法"
	TopicTagParentGuide     = "家长选科"
)

var topicTags = []TagDefinition{
	{Value: TopicTagPhysicsTrack, Label: "物理方向组合怎么选"},
	{Value: TopicTagHistoryTrack, Label: "历史方向就业前景"},
	{Value: TopicTagChemistry, Label: "化学到底有多重要"},
	{Value: TopicTagSelectionTiming, Label: "高二选科时间线"},
	{Value: TopicTagScoreGrowth, Label: "选科后如何提分"},
	{Value: TopicTagParentGuide, Label: "家长如何帮孩子选科"},
}

var subjectTags = []TagDefinition{
	{Value: "物化生", Label: "物理 + 化学 + 生物"},
	{Value: "物化政", Label: "物理 + 化学 + 政治"},
	{Value: "物化地", Label: "物理 + 化学 + 地理"},
	{Value: "物生政", Label: "物理 + 生物 + 政治"},
	{Value: "物生地", Label: "物理 + 生物 + 地理"},
	{Value: "物政地", Label: "物理 + 政治 + 地理"},
	{Value: "史化生", Label: "历史 + 化学 + 生物"},
	{Value: "史化政", Label: "历史 + 化学 + 政治"},
	{Value: "史化地", Label: "历史 + 化学 + 地理"},
	{Value: "史生政", Label: "历史 + 生物 + 政治"},
	{Value: "史生地", Label: "历史 + 生物 + 地理"},
	{Value: "史政地", Label: "历史 + 政治 + 地理"},
}

var topicTagsBySlug = map[string]string{
	"physics-track-how-to-choose": TopicTagPhysicsTrack,
	"physics-combo":               TopicTagPhysicsTrack,
	"history-track-careers":       TopicTagHistoryTrack,
	"history-careers":             TopicTagHistoryTrack,
	"is-chemistry-important":      TopicTagChemistry,
	"chemistry-importance":        TopicTagChemistry,
	"grade-eleven-timeline":       TopicTagSelectionTiming,
	"grade-one-timeline":          TopicTagSelectionTiming,
	"after-selection-score-up":    TopicTagScoreGrowth,
	"score-improvement":           TopicTagScoreGrowth,
	"parents-guide":               TopicTagParentGuide,
}

func TopicTags() []TagDefinition {
	return append([]TagDefinition(nil), topicTags...)
}

func SubjectTags() []TagDefinition {
	return append([]TagDefinition(nil), subjectTags...)
}

func TopicTagForSlug(slug string) (string, bool) {
	tag, ok := topicTagsBySlug[slug]
	return tag, ok
}

func SubjectTagForChoice(track SubjectTrack, electives []Subject) (string, bool) {
	if len(electives) != 2 || electives[0] == electives[1] {
		return "", false
	}
	prefix := map[SubjectTrack]string{TrackPhysics: "物", TrackHistory: "史"}[track]
	if prefix == "" {
		return "", false
	}
	labels := map[Subject]string{
		SubjectChemistry: "化",
		SubjectBiology:   "生",
		SubjectPolitics:  "政",
		SubjectGeography: "地",
	}
	first, second := labels[electives[0]], labels[electives[1]]
	if first == "" || second == "" {
		return "", false
	}
	order := "化生政地"
	if strings.Index(order, first) > strings.Index(order, second) {
		first, second = second, first
	}
	tag := prefix + first + second
	for _, definition := range subjectTags {
		if definition.Value == tag {
			return tag, true
		}
	}
	return "", false
}

func IsControlledTag(tag string) bool {
	for _, definitions := range [][]TagDefinition{topicTags, subjectTags} {
		for _, definition := range definitions {
			if definition.Value == tag {
				return true
			}
		}
	}
	return false
}
