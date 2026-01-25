package mapping

import (
	"sort"
	"strings"

	osm "github.com/omniscale/go-osm"
	"github.com/omniscale/imposm3/geom"
)

func (m *Mapping) pointMatcher() (NodeMatcher, error) {
	mappings := make(TagTableMapping)
	m.mappings(PointTable, mappings)
	filters := make(tableElementFilters)
	m.addFilters(filters)
	m.addTypedFilters(PointTable, filters)
	tables, err := m.tables(PointTable)
	return &tagMatcher{
		mappings:    mappings,
		filters:     filters,
		tables:      tables,
		multiValues: m.multiValues(PointTable),
		matchAreas:  false,
	}, err
}

func (m *Mapping) lineStringMatcher() (WayMatcher, error) {
	mappings := make(TagTableMapping)
	m.mappings(LineStringTable, mappings)
	filters := make(tableElementFilters)
	m.addFilters(filters)
	m.addTypedFilters(LineStringTable, filters)
	tables, err := m.tables(LineStringTable)
	return &tagMatcher{
		mappings:    mappings,
		filters:     filters,
		tables:      tables,
		multiValues: m.multiValues(LineStringTable),
		matchAreas:  false,
	}, err
}

func (m *Mapping) polygonMatcher() (RelWayMatcher, error) {
	mappings := make(TagTableMapping)
	m.mappings(PolygonTable, mappings)
	filters := make(tableElementFilters)
	m.addFilters(filters)
	m.addTypedFilters(PolygonTable, filters)
	relFilters := make(tableElementFilters)
	m.addRelationFilters(PolygonTable, relFilters)
	tables, err := m.tables(PolygonTable)
	return &tagMatcher{
		mappings:    mappings,
		filters:     filters,
		tables:      tables,
		relFilters:  relFilters,
		multiValues: m.multiValues(PolygonTable),
		matchAreas:  true,
	}, err
}

func (m *Mapping) relationMatcher() (RelationMatcher, error) {
	mappings := make(TagTableMapping)
	m.mappings(RelationTable, mappings)
	filters := make(tableElementFilters)
	m.addFilters(filters)
	m.addTypedFilters(PolygonTable, filters)
	m.addTypedFilters(RelationTable, filters)
	relFilters := make(tableElementFilters)
	m.addRelationFilters(RelationTable, relFilters)
	tables, err := m.tables(RelationTable)
	return &tagMatcher{
		mappings:    mappings,
		filters:     filters,
		tables:      tables,
		relFilters:  relFilters,
		multiValues: m.multiValues(RelationTable),
		matchAreas:  true,
	}, err
}

func (m *Mapping) relationMemberMatcher() (RelationMatcher, error) {
	mappings := make(TagTableMapping)
	m.mappings(RelationMemberTable, mappings)
	filters := make(tableElementFilters)
	m.addFilters(filters)
	m.addTypedFilters(RelationMemberTable, filters)
	relFilters := make(tableElementFilters)
	m.addRelationFilters(RelationMemberTable, relFilters)
	tables, err := m.tables(RelationMemberTable)
	return &tagMatcher{
		mappings:    mappings,
		filters:     filters,
		tables:      tables,
		relFilters:  relFilters,
		multiValues: m.multiValues(RelationMemberTable),
		matchAreas:  true,
	}, err
}

type NodeMatcher interface {
	MatchNode(node *osm.Node) []Match
}

type WayMatcher interface {
	MatchWay(way *osm.Way) []Match
}

type RelationMatcher interface {
	MatchRelation(rel *osm.Relation) []Match
}

type RelWayMatcher interface {
	WayMatcher
	RelationMatcher
}

type Match struct {
	Key     string
	Value   string
	Table   DestTable
	builder *rowBuilder
}

func (m *Match) Row(elem *osm.Element, geom *geom.Geometry) []any {
	return m.builder.MakeRow(elem, geom, *m)
}

func (m *Match) MemberRow(rel *osm.Relation, member *osm.Member, memberIndex int, geom *geom.Geometry) []any {
	return m.builder.MakeMemberRow(rel, member, memberIndex, geom, *m)
}

type tagMatcher struct {
	mappings    TagTableMapping
	tables      map[string]*rowBuilder
	filters     tableElementFilters
	relFilters  tableElementFilters
	multiValues tableElementMultiValues
	matchAreas  bool
}

func (tm *tagMatcher) MatchNode(node *osm.Node) []Match {
	return tm.match(node.Tags, "point", false)
}

func (tm *tagMatcher) MatchWay(way *osm.Way) []Match {
	var matches []Match
	if tm.matchAreas {
		if way.IsClosed() {
			if way.Tags["area"] != "no" {
				matches = tm.match(way.Tags, "way", true)
			}
		}
	} else {
		if way.IsClosed() {
			if way.Tags["area"] != "yes" {
				matches = tm.match(way.Tags, "way", true)
			}
		} else {
			matches = tm.match(way.Tags, "way", false)
		}
	}
	return matches
}

func (tm *tagMatcher) MatchRelation(rel *osm.Relation) []Match {
	matches := tm.match(rel.Tags, "relation", true)
	return matches
}

type orderedMatch struct {
	Match
	order int
}

func (tm *tagMatcher) match(tags osm.Tags, elemType string, closed bool) []Match {
	type tableKeyMatches struct {
		order   int
		matches []orderedMatch
	}

	tables := make(map[DestTable]map[Key]*tableKeyMatches)

	addTableMatch := func(k, v string, t orderedDestTable) {
		keyMatches, ok := tables[t.DestTable]

		if !ok {
			keyMatches = make(map[Key]*tableKeyMatches)
			tables[t.DestTable] = keyMatches
		}

		entry, ok := keyMatches[Key(k)]

		if !ok {
			entry = &tableKeyMatches{order: t.order}
			keyMatches[Key(k)] = entry
		} else if t.order < entry.order {
			entry.order = t.order
		}

		entry.matches = append(entry.matches, orderedMatch{
			Match: Match{
				Key:     k,
				Value:   v,
				Table:   t.DestTable,
				builder: tm.tables[t.Name],
			},
			order: t.order,
		})
	}

	if values, ok := tm.mappings[Key("__any__")]; ok {
		for _, t := range values["__any__"] {
			addTableMatch("__any__", "__any__", t)
		}
	}

	for k, v := range tags {
		values, ok := tm.mappings[Key(k)]

		if ok {
			if tbls, ok := values["__any__"]; ok {
				for _, t := range tbls {
					addTableMatch(k, v, t)
				}
			}

			if tbls, ok := values[Value(v)]; ok {
				for _, t := range tbls {
					addTableMatch(k, v, t)
				}
			}

			if strings.Contains(v, ";") {
				for _, val := range splitTagValues(v) {
					if tbls, ok := values[Value(val)]; ok {
						for _, t := range tbls {
							if tm.multiValuesForTableKey(t.Name, Key(k)) {
								addTableMatch(k, val, t)
							}
						}
					}
				}
			}
		}
	}

	var matches []Match
	for t, keyMatches := range tables {
		var selected *tableKeyMatches
		for key, entry := range keyMatches {
			if !tm.multiValuesForTableKey(t.Name, key) {
				entry.matches = reduceMatches(entry.matches)
				if len(entry.matches) == 0 {
					continue
				}
				entry.order = entry.matches[0].order
			}
			if selected == nil || entry.order < selected.order {
				selected = entry
			}
		}
		if selected == nil || len(selected.matches) == 0 {
			continue
		}

		sort.SliceStable(selected.matches, func(i, j int) bool {
			return selected.matches[i].order < selected.matches[j].order
		})

		match := selected.matches[0].Match
		filters, ok := tm.filters[t.Name]
		filteredOut := false
		if ok {
			for _, filter := range filters {
				if !filter(tags, Key(match.Key), elemType, closed) {
					filteredOut = true
					if isDebugBusStop(tags) {
						log.Printf("[info] debug node filtered by table filter for %s (key=%s closed=%t relation=%t)", t.Name, match.Key, closed, relation)
					}
					break
				}
			}
		}
		if elemType == "relation" && !filteredOut {
			filters, ok := tm.relFilters[t.Name]
			if ok {
				for _, filter := range filters {
					if !filter(tags, Key(match.Key), elemType, closed) {
						filteredOut = true
						if isDebugBusStop(tags) {
							log.Printf("[info] debug node filtered by relation filter for %s (key=%s closed=%t)", t.Name, match.Key, closed)
						}
						break
					}
				}
			}
		}

		if !filteredOut {
			for _, selectedMatch := range selected.matches {
				matches = append(matches, selectedMatch.Match)
			}
		}
	}
	return matches
}

func (tm *tagMatcher) multiValuesForTableKey(tableName string, key Key) bool {
	if multiValues, ok := tm.multiValues[tableName]; ok {
		if _, ok := multiValues["__any__"]; ok {
			return true
		}
		_, ok := multiValues[key]

		return ok
	}
	return false
}

func reduceMatches(matches []orderedMatch) []orderedMatch {
	if len(matches) == 0 {
		return matches
	}
	best := matches[0]
	for i := 1; i < len(matches); i++ {
		if matches[i].order < best.order {
			best = matches[i]
		}
	}
	return []orderedMatch{best}
}

type valueBuilder struct {
	key     Key
	colType ColumnType
}

func (v *valueBuilder) Value(elem *osm.Element, geom *geom.Geometry, match Match) any {
	if v.colType.Func != nil {
		return v.colType.Func(elem.Tags[string(v.key)], elem, geom, match)
	}
	return nil
}

func (v *valueBuilder) MemberValue(rel *osm.Relation, member *osm.Member, memberIndex int, geom *geom.Geometry, match Match) any {
	if v.colType.Func != nil {
		if v.colType.FromMember {
			if member.Element == nil {
				return nil
			}
			return v.colType.Func(member.Element.Tags[string(v.key)], member.Element, geom, match)
		}
		return v.colType.Func(rel.Tags[string(v.key)], &rel.Element, geom, match)
	}
	if v.colType.MemberFunc != nil {
		return v.colType.MemberFunc(rel, member, memberIndex, match)
	}
	return nil
}

type rowBuilder struct {
	columns []valueBuilder
}

func (r *rowBuilder) MakeRow(elem *osm.Element, geom *geom.Geometry, match Match) []any {
	var row []any
	for _, column := range r.columns {
		row = append(row, column.Value(elem, geom, match))
	}
	return row
}

func (r *rowBuilder) MakeMemberRow(rel *osm.Relation, member *osm.Member, memberIndex int, geom *geom.Geometry, match Match) []any {
	var row []any
	for _, column := range r.columns {
		row = append(row, column.MemberValue(rel, member, memberIndex, geom, match))
	}
	return row
}
