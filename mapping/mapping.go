package mapping

import (
	"fmt"
	"os"
	"regexp"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	osm "github.com/omniscale/go-osm"
	"github.com/omniscale/imposm3/log"
	"github.com/omniscale/imposm3/mapping/config"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type orderedDestTable struct {
	DestTable
	order int
}

type TagTableMapping map[Key]map[Value][]orderedDestTable

func (tt TagTableMapping) addFromMapping(mapping config.KeyValues, table DestTable) {
	for key, vals := range mapping {
		for _, v := range vals {
			vals, ok := tt[Key(key)]
			tbl := orderedDestTable{DestTable: table, order: v.Order}
			if ok {
				vals[Value(v.Value)] = append(vals[Value(v.Value)], tbl)
			} else {
				tt[Key(key)] = make(map[Value][]orderedDestTable)
				tt[Key(key)][Value(v.Value)] = append(tt[Key(key)][Value(v.Value)], tbl)
			}
		}
	}
}

func (tt TagTableMapping) asTagMap() tagMap {
	result := make(tagMap)
	for k, vals := range tt {
		result[k] = make(map[Value]struct{})
		for v := range vals {
			result[k][v] = struct{}{}
		}
	}
	return result
}

type DestTable struct {
	Name       string
	SubMapping string
}

type TableType string

func (tt *TableType) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case "":
		return errors.New("missing table type")
	case `"point"`:
		*tt = PointTable
	case `"linestring"`:
		*tt = LineStringTable
	case `"polygon"`:
		*tt = PolygonTable
	case `"geometry"`:
		*tt = GeometryTable
	case `"point_or_polygon"`:
		*tt = PointOrPolygonTable
	case `"relation"`:
		*tt = RelationTable
	case `"relation_member"`:
		*tt = RelationMemberTable
	}
	return errors.New("unknown type " + string(data))
}

const (
	PolygonTable        TableType = "polygon"
	LineStringTable     TableType = "linestring"
	PointTable          TableType = "point"
	GeometryTable       TableType = "geometry"
	PointOrPolygonTable TableType = "point_or_polygon"
	RelationTable       TableType = "relation"
	RelationMemberTable TableType = "relation_member"
)

type Mapping struct {
	Conf                  config.Mapping
	PointMatcher          NodeMatcher
	LineStringMatcher     WayMatcher
	PolygonMatcher        RelWayMatcher
	RelationMatcher       RelationMatcher
	RelationMemberMatcher RelationMatcher
}

func FromFile(filename string) (*Mapping, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return New(b)
}

func New(b []byte) (*Mapping, error) {
	mapping := Mapping{}
	err := yaml.UnmarshalStrict(b, &mapping.Conf)
	if err != nil {
		return nil, err
	}

	err = mapping.prepare()
	if err != nil {
		return nil, err
	}

	err = mapping.createMatcher()
	if err != nil {
		return nil, err
	}
	return &mapping, nil
}

func (m *Mapping) prepare() error {
	for name, t := range m.Conf.Tables {
		t.Name = name
		if t.OldFields != nil {
			// todo deprecate 'fields'
			t.Columns = t.OldFields
		}
		if t.Type == "" {
			return errors.Errorf("missing type for table %s", name)
		}

		if TableType(t.Type) == GeometryTable {
			if t.Mapping != nil || t.Mappings != nil {
				return errors.Errorf("table with type:geometry requires type_mappings for table %s", name)
			}
		}
	}

	for name, t := range m.Conf.GeneralizedTables {
		t.Name = name
	}

	for _, includeRegex := range m.Conf.Tags.IncludeRegex {
		if _, err := regexp.Compile(includeRegex); err != nil {
			return errors.Wrapf(err, "invalid tags.include_regex pattern %q", includeRegex)
		}
	}
	return nil
}

func (m *Mapping) createMatcher() error {
	var err error
	m.PointMatcher, err = m.pointMatcher()
	if err != nil {
		return err
	}
	m.LineStringMatcher, err = m.lineStringMatcher()
	if err != nil {
		return err
	}
	m.PolygonMatcher, err = m.polygonMatcher()
	if err != nil {
		return err
	}
	m.RelationMatcher, err = m.relationMatcher()
	if err != nil {
		return err
	}
	m.RelationMemberMatcher, err = m.relationMemberMatcher()
	if err != nil {
		return err
	}
	return nil
}

type tableElementMultiValues map[string]map[Key]struct{}

func (m *Mapping) multiValues(tableType TableType) tableElementMultiValues {
	options := make(tableElementMultiValues)

	for name, t := range m.Conf.Tables {
		if !tableMatchesType(t, tableType) {
			continue
		}

		multiValues := make(map[Key]struct{})

		for _, key := range t.MultiValues {
			multiValues[Key(key)] = struct{}{}
		}

		options[name] = multiValues
	}
	return options
}

func (m *Mapping) splitValuesForTableKey(t *config.Table, key string) bool {
	if len(t.MultiValues) == 0 {
		return false
	}

	for _, k := range t.MultiValues {
		if k == "__any__" || string(k) == key {
			return true
		}
	}

	return false
}

func (m *Mapping) multiValueKeysForFilters(tableTypes ...TableType) (map[Key]bool, bool) {
	keys := make(map[Key]bool)
	any := false
	for _, t := range m.Conf.Tables {
		match := false

		for _, tableType := range tableTypes {
			if tableMatchesType(t, tableType) {
				match = true
				break
			}
		}

		if !match {
			continue
		}

		for _, key := range t.MultiValues {
			if key == "__any__" {
				any = true
				continue
			}
			keys[Key(key)] = true
		}
	}

	return keys, any
}

func (m *Mapping) mappings(tableType TableType, mappings TagTableMapping) {
	for name, t := range m.Conf.Tables {
		if !tableMatchesType(t, tableType) {
			continue
		}
		mappings.addFromMapping(t.Mapping, DestTable{Name: name})

		for subMappingName, subMapping := range t.Mappings {
			mappings.addFromMapping(subMapping.Mapping, DestTable{Name: name, SubMapping: subMappingName})
		}

		addTypeMapping := func(typeMapping config.TypeMapping) {
			if typeMapping.Mapping != nil {
				mappings.addFromMapping(typeMapping.Mapping, DestTable{Name: name})
			}
			for subMappingName, subMapping := range typeMapping.Mappings {
				mappings.addFromMapping(subMapping.Mapping, DestTable{Name: name, SubMapping: subMappingName})
			}
		}

		switch tableType {
		case PointTable:
			addTypeMapping(t.TypeMappings.Points)
		case LineStringTable:
			addTypeMapping(t.TypeMappings.LineStrings)
		case PolygonTable:
			addTypeMapping(t.TypeMappings.Polygons)
		}
		addTypeMapping(t.TypeMappings.Any)
	}
}

func (m *Mapping) tables(tableType TableType) (map[string]*rowBuilder, error) {
	var err error
	result := make(map[string]*rowBuilder)
	for name, t := range m.Conf.Tables {
		if tableMatchesType(t, tableType) {
			result[name], err = makeRowBuilder(t)
			if err != nil {
				return nil, errors.Wrapf(err, "creating row builder for %s", name)
			}

		}
	}
	return result, nil
}

func makeRowBuilder(tbl *config.Table) (*rowBuilder, error) {
	result := rowBuilder{}

	for _, mappingColumn := range tbl.Columns {
		column := valueBuilder{}
		column.key = Key(mappingColumn.Key)

		columnType, err := MakeColumnType(mappingColumn)
		if err != nil {
			return nil, errors.Wrapf(err, "creating column %s", mappingColumn.Name)
		}
		if mappingColumn.GeometryTransform != "" && (columnType.Name == "geometry" || columnType.Name == "validated_geometry") {
			normalized, err := normalizeGeometryTransform(mappingColumn.GeometryTransform)
			if err != nil {
				return nil, errors.Wrapf(err, "column %s", mappingColumn.Name)
			}
			columnType.Func = makeGeometryTransformFunc(normalized)
		}
		column.colType = *columnType
		result.columns = append(result.columns, column)
	}
	return &result, nil
}

func MakeColumnType(c *config.Column) (*ColumnType, error) {
	columnType, ok := AvailableColumnTypes[c.Type]
	if !ok {
		return nil, errors.Errorf("unhandled type %s", c.Type)
	}

	if columnType.MakeFunc != nil {
		makeValue, err := columnType.MakeFunc(c.Name, columnType, *c)
		if err != nil {
			return nil, err
		}
		columnType = ColumnType{columnType.Name, columnType.GoType, makeValue, nil, nil, columnType.FromMember}
	}
	columnType.FromMember = c.FromMember
	return &columnType, nil
}

func (m *Mapping) extraTags(tableType TableType, tags map[Key]bool) {
	for _, t := range m.Conf.Tables {
		if !tableMatchesType(t, tableType) {
			continue
		}

		for _, col := range t.Columns {
			if col.Key != "" {
				tags[Key(col.Key)] = true
			}
			for _, k := range col.Keys {
				tags[Key(k)] = true
			}
		}

		if t.Filters != nil && t.Filters.ExcludeTags != nil {
			for _, keyVal := range *t.Filters.ExcludeTags {
				tags[Key(keyVal[0])] = true
			}
		}

		if tableType == PolygonTable || tableType == RelationTable || tableType == RelationMemberTable {
			if t.RelationTypes != nil {
				tags["type"] = true
			}
		}
	}
	for _, k := range m.Conf.Tags.Include {
		tags[Key(k)] = true
	}

	// always include area tag for closed-way handling
	tags["area"] = true
}

type elementFilter func(tags osm.Tags, key Key, elemType string, closed bool) bool

type tableElementFilters map[string][]elementFilter

func (m *Mapping) addTypedFilters(tableType TableType, filters tableElementFilters) {
	var areaTags map[Key]struct{}
	var linearTags map[Key]struct{}
	if m.Conf.Areas.AreaTags != nil {
		areaTags = make(map[Key]struct{})
		for _, tag := range m.Conf.Areas.AreaTags {
			areaTags[Key(tag)] = struct{}{}
		}
	}
	if m.Conf.Areas.LinearTags != nil {
		linearTags = make(map[Key]struct{})
		for _, tag := range m.Conf.Areas.LinearTags {
			linearTags[Key(tag)] = struct{}{}
		}
	}

	for name, t := range m.Conf.Tables {
		if !tableMatchesType(t, tableType) {
			continue
		}
		if TableType(t.Type) == LineStringTable && areaTags != nil {
			f := func(tags osm.Tags, key Key, elemType string, closed bool) bool {
				if closed {
					if tags["area"] == "yes" {
						return false
					}
					if tags["area"] != "no" {
						if _, ok := areaTags[key]; ok {
							return false
						}
					}
				}
				return true
			}
			filters[name] = append(filters[name], f)
		}
		if (TableType(t.Type) == PolygonTable || (TableType(t.Type) == PointOrPolygonTable && tableType == PolygonTable)) && linearTags != nil {
			f := func(tags osm.Tags, key Key, elemType string, closed bool) bool {
				if closed && tags["area"] == "no" {
					return false
				}
				if tags["area"] != "yes" {
					if _, ok := linearTags[key]; ok {
						return false
					}
				}
				return true
			}
			filters[name] = append(filters[name], f)
		}
	}
}

func (m *Mapping) addRelationFilters(tableType TableType, filters tableElementFilters) {
	for name, t := range m.Conf.Tables {
		if t.RelationTypes != nil {
			relTypes := t.RelationTypes
			f := func(tags osm.Tags, key Key, elemType string, closed bool) bool {
				if v, ok := tags["type"]; ok {
					for _, rtype := range relTypes {
						if v == rtype {
							return true
						}
					}
				}
				return false
			}
			filters[name] = append(filters[name], f)
		} else {
			if TableType(t.Type) == PolygonTable || TableType(t.Type) == PointOrPolygonTable {
				// standard multipolygon handling (boundary and land_area are for backwards compatibility)
				f := func(tags osm.Tags, key Key, elemType string, closed bool) bool {
					if v, ok := tags["type"]; ok {
						if v == "multipolygon" || v == "boundary" || v == "land_area" {
							return true
						}
					}
					return false
				}
				filters[name] = append(filters[name], f)
			}
		}
	}
}

func tableMatchesType(t *config.Table, tableType TableType) bool {
	ttype := TableType(t.Type)
	if ttype == GeometryTable || ttype == tableType {
		return true
	}
	if ttype == PointOrPolygonTable && (tableType == PointTable || tableType == PolygonTable) {
		return true
	}
	return false
}

func (m *Mapping) addFilters(filters tableElementFilters) {
	for name, t := range m.Conf.Tables {
		if t.Filters == nil {
			continue
		}
		if t.Filters.ExcludeTags != nil {
			log.Println("[warn]: exclude_tags filter is deprecated and will be removed. See require and reject filter.")
			for _, filterKeyVal := range *t.Filters.ExcludeTags {
				// Convert `exclude_tags`` filter to `reject` filter !
				keyname := filterKeyVal[0]
				vararr := []config.OrderedValue{
					{
						Value: config.Value(filterKeyVal[1]),
						Order: 1,
					},
				}
				filters[name] = append(filters[name], makeFiltersFunction(name, false, true, keyname, vararr, m.splitValuesForTableKey(t, keyname)))

			}
		}

		if t.Filters.Require != nil {
			for keyname, vararr := range t.Filters.Require {
				filters[name] = append(filters[name], makeFiltersFunction(name, true, false, string(keyname), vararr, m.splitValuesForTableKey(t, string(keyname))))
			}
		}

		if t.Filters.Reject != nil {
			for keyname, vararr := range t.Filters.Reject {
				filters[name] = append(filters[name], makeFiltersFunction(name, false, true, string(keyname), vararr, m.splitValuesForTableKey(t, string(keyname))))
			}
		}

		if t.Filters.RequireRegexp != nil {
			for keyname, re := range t.Filters.RequireRegexp {
				filters[name] = append(filters[name], makeRegexpFiltersFunction(name, true, false, string(keyname), re, m.splitValuesForTableKey(t, string(keyname))))
			}
		}

		if t.Filters.RejectRegexp != nil {
			for keyname, re := range t.Filters.RejectRegexp {
				filters[name] = append(filters[name], makeRegexpFiltersFunction(name, false, true, string(keyname), re, m.splitValuesForTableKey(t, string(keyname))))
			}
		}

		if t.Filters.Filter != "" {
			program, err := expr.Compile(
				t.Filters.Filter,
				expr.Env(filterExprEnv{}),
				expr.AsBool(),
			)
			if err != nil {
				panic(fmt.Sprintf("invalid filter expression for table %s: %v", name, err))
			}
			filters[name] = append(filters[name], makeExprFilterFunction(program))
		}

	}
}

func findValueInOrderedValue(v config.Value, list []config.OrderedValue) bool {
	for _, item := range list {
		if item.Value == v {
			return true
		}
	}
	return false
}

func makeRegexpFiltersFunction(tablename string, virtualTrue bool, virtualFalse bool, vKeyname string, vRegexp string, splitValues bool) func(tags osm.Tags, key Key, elemType string, closed bool) bool {
	// Compile regular expression,  if not valid regexp --> panic !
	r := regexp.MustCompile(vRegexp)
	return func(tags osm.Tags, key Key, elemType string, closed bool) bool {
		if v, ok := tags[vKeyname]; ok {
			if !splitValues {
				if r.MatchString(v) {
					return virtualTrue
				}
				return virtualFalse
			}
			for _, value := range splitTagValues(v) {
				if r.MatchString(value) {
					return virtualTrue
				}
			}
		}
		return virtualFalse
	}
}

func makeFiltersFunction(tablename string, virtualTrue bool, virtualFalse bool, vKeyname string, vVararr []config.OrderedValue, splitValues bool) func(tags osm.Tags, key Key, elemType string, closed bool) bool {

	if findValueInOrderedValue("__nil__", vVararr) {
		log.Println("[warn] Filter value '__nil__' is not supported ! (tablename:" + tablename + ")")
	}

	if findValueInOrderedValue("__any__", vVararr) {
		if len(vVararr) > 1 {
			log.Println("[warn] Multiple filter value with '__any__' keywords is not valid! (tablename:" + tablename + ")")
		}
		return func(tags osm.Tags, key Key, elemType string, closed bool) bool {
			if _, ok := tags[vKeyname]; ok {
				return virtualTrue
			}
			return virtualFalse
		}
	} else if len(vVararr) == 1 {
		return func(tags osm.Tags, key Key, elemType string, closed bool) bool {
			if v, ok := tags[vKeyname]; ok {
				if !splitValues {
					if config.Value(v) == vVararr[0].Value {
						return virtualTrue
					}
					return virtualFalse
				}
				for _, value := range splitTagValues(v) {
					if config.Value(value) == vVararr[0].Value {
						return virtualTrue
					}
				}
			}
			return virtualFalse
		}
	}

	return func(tags osm.Tags, key Key, elemType string, closed bool) bool {
		if v, ok := tags[vKeyname]; ok {
			if !splitValues {
				if findValueInOrderedValue(config.Value(v), vVararr) {
					return virtualTrue
				}
				return virtualFalse
			}
			for _, value := range splitTagValues(v) {
				if findValueInOrderedValue(config.Value(value), vVararr) {
					return virtualTrue
				}
			}
		}
		return virtualFalse
	}
}

type filterExprEnv struct {
	Tags   map[string]string `expr:"tags"`
	Type   string            `expr:"type"`
	Closed bool              `expr:"closed"`
}

func makeExprFilterFunction(program *vm.Program) func(tags osm.Tags, key Key, elemType string, closed bool) bool {
	return func(tags osm.Tags, key Key, elemType string, closed bool) bool {
		result, err := expr.Run(program, filterExprEnv{Tags: tags, Type: elemType, Closed: closed})
		if err != nil {
			return false
		}
		accepted, ok := result.(bool)
		if !ok {
			return false
		}
		return accepted
	}
}
