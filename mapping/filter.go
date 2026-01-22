package mapping

import (
	"path"
	"regexp"
	"strings"

	osm "github.com/omniscale/go-osm"
	"github.com/omniscale/imposm3/mapping/config"
)

type TagFilterer interface {
	Filter(tags *osm.Tags)
}

func (m *Mapping) NodeTagFilter() TagFilterer {
	if m.Conf.Tags.LoadAll {
		return newExcludeFilter(m.Conf.Tags.Exclude)
	}
	mappings := make(TagTableMapping)
	m.mappings(PointTable, mappings)
	tags := make(map[Key]bool)
	m.extraTags(PointTable, tags)
	m.extraTags(RelationMemberTable, tags)
	splitKeys, splitAny := m.multiValueKeysForFilters(PointTable, RelationMemberTable)
	return &tagFilter{
		mappings:       mappings.asTagMap(),
		extraTags:      tags,
		splitKeys:      splitKeys,
		splitAny:       splitAny,
		includeRegexps: compileRegexps(m.Conf.Tags.IncludeRegex),
	}
}

func (m *Mapping) WayTagFilter() TagFilterer {
	if m.Conf.Tags.LoadAll {
		return newExcludeFilter(m.Conf.Tags.Exclude)
	}
	mappings := make(TagTableMapping)
	m.mappings(LineStringTable, mappings)
	m.mappings(PolygonTable, mappings)
	tags := make(map[Key]bool)
	m.extraTags(LineStringTable, tags)
	m.extraTags(PolygonTable, tags)
	m.extraTags(RelationMemberTable, tags)
	splitKeys, splitAny := m.multiValueKeysForFilters(LineStringTable, PolygonTable, RelationMemberTable)
	return &tagFilter{
		mappings:       mappings.asTagMap(),
		extraTags:      tags,
		splitKeys:      splitKeys,
		splitAny:       splitAny,
		includeRegexps: compileRegexps(m.Conf.Tags.IncludeRegex),
	}
}

func (m *Mapping) RelationTagFilter() TagFilterer {
	if m.Conf.Tags.LoadAll {
		return newExcludeFilter(m.Conf.Tags.Exclude)
	}
	mappings := make(TagTableMapping)
	// do not filter out type tag for common relations
	mappings["type"] = map[Value][]orderedDestTable{
		"multipolygon": {},
		"boundary":     {},
		"land_area":    {},
	}
	m.mappings(LineStringTable, mappings)
	m.mappings(PolygonTable, mappings)
	m.mappings(RelationTable, mappings)
	m.mappings(RelationMemberTable, mappings)
	tags := make(map[Key]bool)
	m.extraTags(LineStringTable, tags)
	m.extraTags(PolygonTable, tags)
	m.extraTags(RelationTable, tags)
	m.extraTags(RelationMemberTable, tags)
	splitKeys, splitAny := m.multiValueKeysForFilters(LineStringTable, PolygonTable, RelationTable, RelationMemberTable)
	return &tagFilter{
		mappings:       mappings.asTagMap(),
		extraTags:      tags,
		splitKeys:      splitKeys,
		splitAny:       splitAny,
		includeRegexps: compileRegexps(m.Conf.Tags.IncludeRegex),
	}
}

type tagMap map[Key]map[Value]struct{}

type tagFilter struct {
	mappings       tagMap
	extraTags      map[Key]bool
	splitKeys      map[Key]bool
	splitAny       bool
	includeRegexps []*regexp.Regexp
}

func (f *tagFilter) Filter(tags *osm.Tags) {
	if tags == nil {
		return
	}
	for k, v := range *tags {
		values, ok := f.mappings[Key(k)]
		splitValues := f.splitAny || f.splitKeys[Key(k)]
		if ok {
			if _, ok := values["__any__"]; ok {
				continue
			} else if mappingValueMatches(values, v, splitValues) {
				continue
			} else if _, ok := f.extraTags[Key(k)]; !ok {
				if f.matchesIncludeRegex(k) {
					continue
				}
				delete(*tags, k)
			}
		} else if _, ok := f.extraTags[Key(k)]; !ok {
			if f.matchesIncludeRegex(k) {
				continue
			}
			delete(*tags, k)
		}
	}
}

func (f *tagFilter) matchesIncludeRegex(k string) bool {
	for _, includeRegexp := range f.includeRegexps {
		if includeRegexp.MatchString(k) {
			return true
		}
	}
	return false
}

func compileRegexps(patterns []string) []*regexp.Regexp {
	result := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		result = append(result, regexp.MustCompile(pattern))
	}
	return result
}

type excludeFilter struct {
	keys    map[Key]struct{}
	matches []string
}

func newExcludeFilter(tags []config.Key) *excludeFilter {
	f := excludeFilter{
		keys:    make(map[Key]struct{}),
		matches: make([]string, 0),
	}
	for _, t := range tags {
		if strings.ContainsAny(string(t), "?*[") {
			f.matches = append(f.matches, string(t))
		} else {
			f.keys[Key(t)] = struct{}{}
		}
	}
	return &f
}

func (f *excludeFilter) Filter(tags *osm.Tags) {
	for k := range *tags {
		if _, ok := f.keys[Key(k)]; ok {
			delete(*tags, k)
		} else if f.matches != nil {
			for _, exkey := range f.matches {
				if ok, _ := path.Match(exkey, k); ok {
					delete(*tags, k)
					break
				}
			}
		}
	}
}
