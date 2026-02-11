package mapping

import (
	"os"
	"testing"

	osm "github.com/omniscale/go-osm"
)

func TestFilters_require(t *testing.T) {
	filterTest(
		t,
		`
tables:
  admin:
    fields:
    - name: id
      type: id
    - key: admin_level
      name: admin_level
      type: integer
    filters:
      require:
        boundary: ["administrative","maritime"]
    mapping:
      admin_level: ['2','4']
    type: linestring
`,
		// Accept
		[]osm.Tags{
			{"admin_level": "2", "boundary": "administrative"},
			{"admin_level": "2", "boundary": "maritime"},
			{"admin_level": "4", "boundary": "administrative", "name": "N4"},
			{"admin_level": "4", "boundary": "maritime", "name": "N4"},
		},
		// Reject
		[]osm.Tags{
			{"admin_level": "0", "boundary": "administrative"},
			{"admin_level": "1", "boundary": "administrative"},
			{"admin_level": "2", "boundary": "postal_code"},
			{"admin_level": "2", "boundary": ""},
			{"admin_level": "2", "boundary": "__nil__"},
			{"admin_level": "4", "boundary": "census"},
			{"admin_level": "3", "boundary": "administrative", "name": "NX"},
			{"admin_level": "2"},
			{"admin_level": "4"},
			{"admin_level": "❤"},
			{"admin_level": "__any__", "boundary": "__any__"},
			{"boundary": "administrative"},
			{"boundary": "maritime"},
			{"name": "maritime"},
		},
	)
}

func TestFilters_require2(t *testing.T) {
	// same as above, but mapping and filters are swapped
	filterTest(
		t,
		`
tables:
  admin:
    fields:
    - name: id
      type: id
    - key: admin_level
      name: admin_level
      type: integer
    filters:
      require:
        admin_level: ["2","4"]
    mapping:
      boundary:
      - administrative
      - maritime
    type: linestring
`,
		// Accept
		[]osm.Tags{
			{"admin_level": "2", "boundary": "administrative"},
			{"admin_level": "2", "boundary": "maritime"},
			{"admin_level": "4", "boundary": "administrative", "name": "N4"},
			{"admin_level": "4", "boundary": "maritime", "name": "N4"},
		},
		// Reject
		[]osm.Tags{
			{"admin_level": "0", "boundary": "administrative"},
			{"admin_level": "1", "boundary": "administrative"},
			{"admin_level": "2", "boundary": "postal_code"},
			{"admin_level": "2", "boundary": ""},
			{"admin_level": "2", "boundary": "__nil__"},
			{"admin_level": "4", "boundary": "census"},
			{"admin_level": "3", "boundary": "administrative", "name": "NX"},
			{"admin_level": "2"},
			{"admin_level": "4"},
			{"admin_level": "❤"},
			{"admin_level": "__any__", "boundary": "__any__"},
			{"boundary": "administrative"},
			{"boundary": "maritime"},
			{"name": "maritime"},
		},
	)
}
func TestFilters_building(t *testing.T) {

	filterTest(
		t,
		`
tables:
  buildings:
    fields:
    - name: id
      type: id
    - key: building
      name: building
      type: string
    filters:
      reject:
        building: ["no","none"]
      require_regexp:
        'addr:housenumber': '^\d+[a-zA-Z,]*$'
        building: '^[a-z_]+$'
    mapping:
      building:
      - __any__
    type: linestring
`,
		// Accept
		[]osm.Tags{
			{"building": "yes", "addr:housenumber": "1a"},
			{"building": "house", "addr:housenumber": "131"},
			{"building": "residential", "addr:housenumber": "21"},
			{"building": "garage", "addr:housenumber": "0"},
			{"building": "hut", "addr:housenumber": "99999999"},
			{"building": "_", "addr:housenumber": "333"},

			{"building": "__any__", "addr:housenumber": "333"},
			{"building": "__nil__", "addr:housenumber": "333"},
			{"building": "y", "addr:housenumber": "1abcdefg"},
			{"building": "tower_block", "addr:housenumber": "1A"},
			{"building": "shed", "name": "N4", "addr:housenumber": "1AAA"},
			{"building": "office", "name": "N4", "addr:housenumber": "0XYAB,"},
		},
		// Reject
		[]osm.Tags{
			{"building": "yes", "addr:housenumber": "aaaaa-number"},
			{"building": "house", "addr:housenumber": "1-3a"},
			{"building": "house", "addr:housenumber": "❤"},
			{"building": "house", "addr:housenumber": "two"},
			{"building": "residential", "addr:housenumber": "x21"},

			{"building": "", "addr:housenumber": "111"},

			{"building": "no"},
			{"building": "no", "addr:housenumber": "1a"},
			{"building": "No", "addr:housenumber": "1a"},
			{"building": "NO", "addr:housenumber": "1a"},
			{"building": "none"},
			{"building": "none", "addr:housenumber": "0"},
			{"building": "nONe", "addr:housenumber": "0"},
			{"building": "No"},
			{"building": "NO"},
			{"building": "NONe"},
			{"building": "Garage"},
			{"building": "Hut"},
			{"building": "Farm"},
			{"building": "tower-block"},
			{"building": "❤"},
			{"building": "Ümlåütê"},
			{"building": "木"},
			{"building": "SheD", "name": "N4"},
			{"building": "oFFice", "name": "N4"},
			{"admin_level": "2"},
			{"admin_level": "4"},
			{"boundary": "administrative"},
			{"boundary": "maritime"},
			{"name": "maritime"},
		},
	)
}

func TestFilters_highway_with_name(t *testing.T) {
	filterTest(
		t,
		`
tables:
  highway:
    fields:
    - name: id
      type: id
    - key: highway
      name: highway
      type: string
    - key: name
      name: name
      type: string
    filters:
      require:
        name: ["__any__"]
      reject:
        highway: ["no","none"]
    mapping:
      highway:
      - __any__
    type: linestring
`,
		// Accept
		[]osm.Tags{
			{"highway": "residential", "name": "N1"},
			{"highway": "service", "name": "N2"},
			{"highway": "track", "name": "N3"},
			{"highway": "unclassified", "name": "N4"},
			{"highway": "path", "name": "N5"},
			{"highway": "", "name": "🌍🌎🌏"},
			{"highway": "_", "name": "N6"},
			{"highway": "y", "name": "N7"},
			{"highway": "tower_block", "name": "N8"},
			{"highway": "shed", "name": "N9"},
			{"highway": "office", "name": "N10"},
			{"highway": "SheD", "name": "N11"},
			{"highway": "oFFice", "name": "N12"},
			{"highway": "❤", "name": "❤"},
			{"highway": "Ümlåütê", "name": "Ümlåütê"},
			{"highway": "木", "name": "木"},
		},
		// Reject
		[]osm.Tags{
			{"highway": "no", "name": "N1"},
			{"highway": "none", "name": "N2"},
			{"highway": "yes"},
			{"highway": "no"},
			{"highway": "none"},
			{"highway": "No"},
			{"highway": "NO"},
			{"highway": "NONe"},
			{"highway": "Garage"},
			{"highway": "residential"},
			{"highway": "path"},
			{"highway": "tower-block"},
			{"highway": "❤"},
			{"highway": "Ümlåütê"},
			{"highway": "木"},
			{"admin_level": "2"},
			{"admin_level": "4"},
			{"boundary": "administrative"},
			{"boundary": "maritime"},
			{"name": "maritime"},
		},
	)
}

func TestFilters_waterway_with_name(t *testing.T) {
	filterTest(
		t,
		`
tables:
  waterway:
    fields:
    - name: id
      type: id
    - key: waterway
      name: waterway
      type: string
    - key: name
      name: name
      type: string
    filters:
      require:
        name: ["__any__"]
        waterway:
        - stream
        - river
        - canal
        - drain
        - ditch
      reject:
        fixme: ['__any__']
        amenity: ['__any__']
        shop: ['__any__']
        building: ['__any__']
        tunnel: ['yes']
      reject_regexp:
        level: '^\D+.*$'
    mapping:
      waterway:
      - __any__
    type: linestring
`,
		// Accept
		[]osm.Tags{
			{"waterway": "stream", "name": "N1"},
			{"waterway": "river", "name": "N2"},
			{"waterway": "canal", "name": "N3"},
			{"waterway": "drain", "name": "N4"},
			{"waterway": "ditch", "name": "N5"},

			{"waterway": "stream", "name": "N1", "tunnel": "no"},
			{"waterway": "river", "name": "N2", "boat": "no"},
			{"waterway": "canal", "name": "N3"},
			{"waterway": "ditch", "name": "N4", "level": "3"},

			{"waterway": "stream", "name": "__any__"},
			{"waterway": "stream", "name": "__nil__"},

			{"waterway": "stream", "name": "❤"},
			{"waterway": "stream", "name": "木"},
			{"waterway": "stream", "name": "Ümlåütê"},
		},
		// Reject
		[]osm.Tags{
			{"waterway": "ditch", "name": "N1", "fixme": "incomplete"},
			{"waterway": "stream", "name": "N1", "amenity": "parking"},
			{"waterway": "river", "name": "N2", "shop": "hairdresser"},
			{"waterway": "canal", "name": "N3", "building": "house"},
			{"waterway": "drain", "name": "N1 tunnel", "tunnel": "yes"},

			{"waterway": "river", "name": "N4", "level": "unknown"},
			{"waterway": "ditch", "name": "N4", "level": "primary"},

			{"waterway": "path", "name": "N5"},
			{"waterway": "_", "name": "N6"},
			{"waterway": "y", "name": "N7"},
			{"waterway": "tower_block", "name": "N8"},
			{"waterway": "shed", "name": "N9"},
			{"waterway": "office", "name": "N10"},
			{"waterway": "SheD", "name": "N11"},
			{"waterway": "oFFice", "name": "N12"},
			{"waterway": "❤", "name": "❤"},
			{"waterway": "Ümlåütê", "name": "Ümlåütê"},
			{"waterway": "木", "name": "木"},
			{"waterway": "no", "name": "N1"},
			{"waterway": "none", "name": "N2"},

			{"waterway": "yes"},
			{"waterway": "no"},
			{"waterway": "none"},
			{"waterway": "tower-block"},
			{"waterway": "❤"},
			{"waterway": "Ümlåütê"},
			{"waterway": "木"},

			{"waterway": "__nil__", "name": "__nil__"},
			{"waterway": "__any__", "name": "__nil__"},

			{"waterway": "stream", "name": "__any__", "shop": "__any__"},
			{"waterway": "stream", "name": "__nil__", "shop": "__any__"},
			{"waterway": "stream", "name": "__any__", "shop": "__nil__"},
			{"waterway": "stream", "name": "__nil__", "shop": "__nil__"},
			{"waterway": "stream", "name": "__any__", "shop": ""},
			{"waterway": "stream", "name": "__nil__", "shop": ""},

			{"admin_level": "2"},
			{"admin_level": "4"},
			{"boundary": "administrative"},
			{"boundary": "maritime"},
			{"name": "maritime"},
		},
	)
}

func TestFilters_exclude_tags(t *testing.T) {
	filterTest(
		t,
		`
tables:
  exclude_tags:
    _comment:  Allways Empty !
    fields:
    - name: id
      type: id
    - key: waterway
      name: waterway
      type: string
    - key: name
      name: name
      type: string
    filters:
      require:
        waterway:
         - stream
      exclude_tags:
      - ['waterway', 'river']
      - ['waterway', 'canal']
      - ['waterway', 'drain']
      - ['waterway', 'ditch']
    mapping:
      waterway:
      - __any__
    type: linestring
`,
		// Accept
		[]osm.Tags{
			{"waterway": "stream", "name": "N1"},
			{"waterway": "stream", "name": "N1", "tunnel": "no"},
			{"waterway": "stream", "name": "N1", "amenity": "parking"},
		},
		// Reject
		[]osm.Tags{
			{"waterway": "river", "name": "N2"},
			{"waterway": "canal", "name": "N3"},
			{"waterway": "drain", "name": "N4"},
			{"waterway": "ditch", "name": "N5"},

			{"waterway": "river", "name": "N2", "boat": "no"},
			{"waterway": "canal", "name": "N3"},
			{"waterway": "ditch", "name": "N4", "level": "3"},

			{"waterway": "ditch", "name": "N1", "fixme": "incomplete"},
			{"waterway": "river", "name": "N2", "shop": "hairdresser"},
			{"waterway": "canal", "name": "N3", "building": "house"},
			{"waterway": "drain", "name": "N1 tunnel", "tunnel": "yes"},

			{"waterway": "river", "name": "N4", "level": "unknown"},
			{"waterway": "ditch", "name": "N4", "level": "primary"},

			{"waterway": "path", "name": "N5"},
			{"waterway": "_", "name": "N6"},
			{"waterway": "y", "name": "N7"},
			{"waterway": "tower_block", "name": "N8"},
			{"waterway": "shed", "name": "N9"},
			{"waterway": "office", "name": "N10"},
			{"waterway": "SheD", "name": "N11"},
			{"waterway": "oFFice", "name": "N12"},
			{"waterway": "❤", "name": "❤"},
			{"waterway": "Ümlåütê", "name": "Ümlåütê"},
			{"waterway": "木", "name": "木"},

			{"waterway": "no", "name": "N1"},
			{"waterway": "none", "name": "N2"},
			{"waterway": "yes"},
			{"waterway": "no"},
			{"waterway": "none"},
			{"waterway": "tower-block"},
			{"waterway": "❤"},
			{"waterway": "Ümlåütê"},
			{"waterway": "木"},
			{"admin_level": "2"},
			{"admin_level": "4"},
			{"boundary": "administrative"},
			{"boundary": "maritime"},
			{"name": "maritime"},
		},
	)
}

func TestFilters_expression(t *testing.T) {
	filterTest(
		t,
		`
tables:
  named_roads:
    fields:
    - name: id
      type: id
    - key: highway
      name: highway
      type: string
    filters:
      filter: 'type == "way" and tags["name"] != ""'
    mapping:
      highway:
      - __any__
    type: linestring
`,
		[]osm.Tags{
			osm.Tags{"highway": "residential", "name": "Main St"},
			osm.Tags{"highway": "service", "name": "Depot Road"},
		},
		[]osm.Tags{
			osm.Tags{"highway": "residential"},
			osm.Tags{"name": "Main St"},
			osm.Tags{"highway": "__any__", "name": ""},
		},
	)
}

func TestFilters_expression_closed(t *testing.T) {
	const mapping = `
tables:
  open_roads:
    fields:
    - name: id
      type: id
    - key: highway
      name: highway
      type: string
    filters:
      filter: '!closed'
    mapping:
      highway:
      - __any__
    type: linestring
`

	configTestMapping, err := New([]byte(mapping))
	if err != nil {
		t.Fatal(err)
	}

	openWay := osm.Way{
		Element: osm.Element{
			Tags: osm.Tags{"highway": "residential"},
		},
		Refs: []int64{1, 2, 3},
	}
	if matches := configTestMapping.LineStringMatcher.MatchWay(&openWay); len(matches) == 0 {
		t.Fatalf("open way should match expression filter")
	}

	closedWay := osm.Way{
		Element: osm.Element{
			Tags: osm.Tags{"highway": "residential"},
		},
		Refs: []int64{1, 2, 3, 1},
	}
	if matches := configTestMapping.LineStringMatcher.MatchWay(&closedWay); len(matches) != 0 {
		t.Fatalf("closed way should be rejected by expression filter")
	}
}

func filterTest(t *testing.T, mapping string, accept []osm.Tags, reject []osm.Tags) {
	var configTestMapping *Mapping
	var err error

	tmpfile, err := os.CreateTemp("", "filter_test_mapping.yml")
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(mapping)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	configTestMapping, err = FromFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	var actualMatch []Match

	elem := osm.Way{}
	ls := configTestMapping.LineStringMatcher

	for _, et := range accept {
		elem.Tags = et
		actualMatch = ls.MatchWay(&elem)
		if len(actualMatch) == 0 {
			t.Errorf("TestFilter - Not Accepted : (%+v)  ", et)
		}
	}

	for _, et := range reject {
		elem.Tags = et
		actualMatch = ls.MatchWay(&elem)

		if len(actualMatch) != 0 {
			t.Errorf("TestFilter - Not Rejected : (%+v)  ", et)
		}
	}

}
