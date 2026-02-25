package expire

import (
	osm "github.com/omniscale/go-osm"
	"github.com/omniscale/imposm3/proj"
)

type Expireor interface {
	Expire(long, lat float64)
	ExpireNodes(nodes []osm.Node, closed bool)
}

func ExpireProjectedNodes(expireor Expireor, nodes []osm.Node, srid int, closed bool) {
	switch srid {
	case 4326:
		expireor.ExpireNodes(nodes, closed)
	case 3857:
		nds := make([]osm.Node, len(nodes))
		for i, nd := range nodes {
			nds[i].Long, nds[i].Lat = proj.MercToWgs(nd.Long, nd.Lat)
		}
		expireor.ExpireNodes(nds, closed)
	default:
		panic("unsupported srid")
	}
}

func ExpireProjectedNode(expireor Expireor, node osm.Node, srid int) {
	switch srid {
	case 4326:
		expireor.Expire(node.Long, node.Lat)
	case 3857:
		long, lat := proj.MercToWgs(node.Long, node.Lat)
		expireor.Expire(long, lat)
	default:
		panic("unsupported srid")
	}
}
