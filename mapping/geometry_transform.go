package mapping

import (
	"strings"
	"sync"

	osm "github.com/omniscale/go-osm"
	"github.com/omniscale/imposm3/geom"
	"github.com/omniscale/imposm3/geom/geos"
	"github.com/omniscale/imposm3/log"
	"github.com/pkg/errors"
)

const (
	geometryTransformCentroid              = "centroid"
	geometryTransformCenter                = "center"
	geometryTransformPointOnSurface        = "point_on_surface"
	geometryTransformPoleOfInaccessibility = "pole_of_inaccessibility"
)

var geometryTransformAliases = map[string]string{
	"centroid":                 geometryTransformCentroid,
	"center":                   geometryTransformCenter,
	"point_on_surface":         geometryTransformPointOnSurface,
	"pointonsurface":           geometryTransformPointOnSurface,
	"pole_of_inaccessibility":  geometryTransformPoleOfInaccessibility,
	"maximum_inscribed_circle": geometryTransformPoleOfInaccessibility,
}

var geometryTransformPool = sync.Pool{
	New: func() interface{} {
		return geos.NewGeos()
	},
}

func normalizeGeometryTransform(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.EqualFold(trimmed, "none") {
		return "", nil
	}
	normalized := strings.ToLower(trimmed)
	transform, ok := geometryTransformAliases[normalized]
	if !ok {
		return "", errors.Errorf("unknown geometry_transform %q", value)
	}
	return transform, nil
}

func makeGeometryTransformFunc(transform string) MakeValue {
	return func(val string, elem *osm.Element, geomValue *geom.Geometry, match Match) interface{} {
		if geomValue == nil || geomValue.Geom == nil {
			return nil
		}
		wkb, err := transformGeometry(geomValue, transform)
		if err != nil {
			log.Printf("[warn] geometry_transform %s failed for table %s: %v", transform, match.Table.Name, err)
			return string(geomValue.Wkb)
		}
		if wkb == nil {
			return nil
		}
		return string(wkb)
	}
}

func transformGeometry(geomValue *geom.Geometry, transform string) ([]byte, error) {
	geosHandle := geometryTransformPool.Get().(*geos.Geos)
	defer geometryTransformPool.Put(geosHandle)

	srid := geosHandle.SRID(geomValue.Geom)
	geosHandle.SetHandleSrid(srid)

	var result *geos.Geom
	switch transform {
	case geometryTransformCentroid:
		result = geosHandle.Centroid(geomValue.Geom)
	case geometryTransformCenter:
		bounds := geomValue.Geom.Bounds()
		if bounds.MinX > bounds.MaxX || bounds.MinY > bounds.MaxY {
			return nil, errors.New("invalid geometry bounds")
		}
		centerX := bounds.MinX + (bounds.MaxX-bounds.MinX)/2
		centerY := bounds.MinY + (bounds.MaxY-bounds.MinY)/2
		result = geosHandle.Point(centerX, centerY)
	case geometryTransformPointOnSurface:
		result = geosHandle.PointOnSurface(geomValue.Geom)
	case geometryTransformPoleOfInaccessibility:
		circle := geosHandle.MaximumInscribedCircle(geomValue.Geom, 0)
		if circle == nil {
			return nil, errors.New("maximum inscribed circle failed")
		}
		defer geosHandle.Destroy(circle)
		result = geosHandle.Centroid(circle)
	default:
		return nil, errors.Errorf("unknown geometry transform %q", transform)
	}

	if result == nil {
		return nil, errors.Errorf("geometry transform %q failed", transform)
	}
	defer geosHandle.Destroy(result)

	wkb := geosHandle.AsEwkbHex(result)
	if wkb == nil {
		return nil, errors.New("failed to encode transformed geometry")
	}
	return wkb, nil
}
