# Imposm with improvements

This is a fork of https://github.com/omniscale/imposm3 with various improvements, implemented with significant help from Codex.

## Mapping extensions in this fork

- **`geometry_transform` (column-level, geometry columns only)**
  Apply a transform to geometry values before insert.
  Supported values: `centroid`, `center`, `point_on_surface`, `pole_of_inaccessibility`.

  Example:

  ```yaml
  columns:
    - name: geometry
      type: geometry
      geometry_transform: point_on_surface
    - name: geometry_raw
      type: geometry
  ```

- **New table type: `point_or_polygon`**
  Matches **nodes** and **polygons/multipolygons** only (no linestrings).

  Example:

  ```yaml
  type: point_or_polygon
  ```

- **`type_mappings.any`**
  Allows tag mappings to match across all geometry types for that table.
  Works with `type: geometry` and `type: point_or_polygon`.

  Example:

  ```yaml
  type_mappings:
    any:
      mapping:
        amenity: [school, hospital]
  ```

- **`type_mappings.*` supports `mapping` and `mappings` (sub-mappings)**
  Same structure as top-level `mapping`/`mappings`.

  Example:

  ```yaml
  type_mappings:
    any:
      mapping:
        highway: [bus_stop]
      mappings:
        stops:
          mapping:
            highway: [bus_stop]
  ```

- **`mapping_value` aliases (per column)**
  Remap mapping values for a specific key; supports `__any__` as catch-all per key.

  Example:

  ```yaml
  columns:
    - name: type
      type: mapping_value
      aliases:
        power:
          tower: power_tower
          __any__: power_other
  ```

- **New column type: `expression` (string result)**
  Compute a column value with `expr-lang/expr`.
  The expression must return a string. If it returns another type or evaluation fails, the value is `NULL`.
  Language reference: https://expr-lang.org/docs/language-definition
  Available variables: `tags`, `id`, `key`, `value`, `tag`.

  Example:

  ```yaml
  columns:
    - name: label
      type: expression
      args:
        expression: 'tags["name"] + " (" + value + ")"'
  ```

- **`multi_values` (per table)**
  Enables splitting `;`-separated tag values for selected keys and keeps multiple matches for those keys.
  Use `__any__` to apply to all keys in the table.
  This also affects filters (`require`/`reject`/`*_regexp`) for the same keys.

  Example:

  ```yaml
  tables:
    places:
      type: point
      multi_values: [place]
      mapping:
        place: [city, town, village]
  ```

- **`filters.filter` (expression-based table filter)**
  Add a boolean expression for advanced filter logic using (Expr)[https://expr-lang.org/docs/language-definition].
  Expression variables: `tags` (map), `type` (`point|way|relation`), `closed` (bool).
  Return `true` to accept, `false` to reject.

  Example:

  ```yaml
  tables:
    sports:
      type: point_or_polygon
      filters:
        filter: 'type == "way" and tags["disused:leisure"] == ""'
      mapping:
        sport: [tennis]
  ```

- **`tags.include_regex` (global tag-key include by regex)**
  Keep extra tags based on regular expressions, without enabling `tags.load_all`.
  Useful for namespaces like `addr:*` or `contact:*`.

  Example:

  ```yaml
  tags:
    include_regex: ["^addr:.*$", "^contact:.*$"]
  ```

- **Strict YAML parsing**
  Mapping files are parsed with `yaml.UnmarshalStrict`, so unknown fields or wrong shapes fail fast.

# Imposm

Imposm is an importer for OpenStreetMap data. It reads PBF files and imports the data into PostgreSQL/PostGIS. It can also automatically update the database with the latest changes from OSM.

It is designed to create databases that are optimized for rendering (i.e. generating tiles or for WMS services).

The development of Imposm is sponsored by [Omniscale](https://omniscale.com/).

*Imposm is in production use by the authors. It is actively maintained, with a focus on resolving future incompatibilities with dependencies such as PostGIS. However, there is no capacity for end-user support, and no new features will be developed beyond its existing scope.*


Features
--------

* High-performance
* Diff support
* Custom database schemas
* Generalized geometries


### In detail


- High performance:
  Parallel from the ground up. It distributes parsing and processing to all available CPU cores.

- Custom database schemas:
  Creates tables for different data types. This allows easier styling and better performance for rendering in WMS or tile services.

- Unify values:
  For example, the boolean values `1`, `on`, `true` and `yes` all become ``TRUE``.

- Filter by tags and values:
  Only import data you are going to render/use.

- Efficient nodes cache:
  It is necessary to store all nodes to build ways and relations. Imposm uses a file-based key-value database to cache this data.

- Generalized tables:
  Automatically creates tables with lower spatial resolutions, perfect for rendering large road networks in low resolutions.

- Limit to polygons:
  Limit imported geometries to polygons from GeoJSON, for city/state/country imports.

- Easy deployment:
  Single binary with only runtime dependencies to common libs (GEOS and LevelDB).

- Automatic OSM updates:
  Includes background service (`imposm run`) that automatically downloads and imports the latest OSM changes.

- Route relations:
  Import all relation types including routes.

- Support for table namespace (PostgreSQL schema)


Performance
-----------

* Imposm makes full use of all available CPU cores
* Imposm uses bulk inserts into PostgreSQL with `COPY FROM`
* Imposm uses efficient intermediate caches for reduced IO load during ways and relations building


An import in diff-mode on a Hetzner AX102 server (AMD Ryzen 9 7950X3D, 256GB RAM and NVMe storage) of a 78GB planet PBF (2024-01-29) with generalized tables and spatial indices, etc. takes around 7:30h. This is for an import that is ready for minutely updates. The non-diff mode is even faster.

It's recommended that the memory size of the server is roughly twice the size of the PBF extract you are importing. For example: You should have 192GB RAM or more for a current (2024) 78GB planet file, 8GB for a 4GB regional extract, etc.
Imports with spinning disks will take significantly longer and are not recommended.


Installation
------------

### Binary

[Binary releases are available at GitHub.](https://github.com/omniscale/imposm3/releases)

These builds are for x86 64bit Linux and require *no* further dependencies. Download, untar and start `imposm`.
Binaries are compatible with Debian 10 and other distributions from 2022 or
newer. You can build from source if you need to support older distributions.

### Source

There are some dependencies:

#### Compiler

You need [Go](http://golang.org). 1.21 or higher is recommended.

#### C/C++ libraries

Other dependencies are [libleveldb][] and [libgeos][].
Imposm was tested with recent versions of these libraries, but you might succeed with older versions.
GEOS >=3.2 is recommended, since it became much more robust when handling invalid geometries.


[libleveldb]: https://github.com/google/leveldb/
[libgeos]: https://libgeos.org/

#### Compile

The quickest way to install Imposm is to call:

    go install github.com/omniscale/imposm3/cmd/imposm@latest

This will download, compile and install Imposm to `~/go/bin/imposm`. You can change the location by setting the `GOBIN` environment.

The recommended way for installation is:

    git clone https://github.com/omniscale/imposm3.git
    cd imposm3
    make build

`make build` will build Imposm into your local path and it will add version information to your binary.

You can also directly use go to build or install imposm with `go build ./cmd/imposm`. However, this will not set the version information.

Go compiles to static binaries and so Imposm has no runtime dependencies to Go.
Just copy the `imposm` binary to your server for deployment. The C/C++ libraries listed above are still required though.

See also `Dockerfile` for instructions on how to build standalone binary packages for Linux.

#### LevelDB

For better performance you should use LevelDB >1.21. You can still build with support for 1.21 with ``go build -tags="ldbpre121"`` or ``LEVELDB_PRE_121=1 make build``.


Usage
-----

`imposm` has multiple subcommands. Use `imposm import` for basic imports.

For a simple import:

    imposm import -connection postgis://user:password@host/database \
        -mapping mapping.json -read /path/to/osm.pbf -write

You need a JSON file with the target database mapping. See `example-mapping.json` to get an idea what is possible with the mapping.

Imposm creates all new tables inside the `import` table schema. So you'll have `import.osm_roads` etc. You can change the tables to the `public` schema:

    imposm import -connection postgis://user:passwd@host/database \
        -mapping mapping.json -deployproduction


You can write some options into a JSON configuration file:

    {
        "cachedir": "/var/local/imposm",
        "mapping": "mapping.json",
        "connection": "postgis://user:password@localhost:port/database"
    }

To use that config:

    imposm import -config config.json [args...]

For more options see:

    imposm import -help


Note: TLS/SSL support is disabled by default. You can re-enable encryption by setting the `PGSSLMODE` environment variable or the `sslmode` connection option to `require` or `verify-full`, eg: `-connect postgis://host/dbname?sslmode=require`.


Documentation
-------------

The latest documentation can be found here: <http://imposm.org/docs/imposm3/latest/>

Support
-------

There is a [mailing list at Google Groups](http://groups.google.com/group/imposm) for all questions. You can subscribe by sending an email to: `imposm+subscribe@googlegroups.com`

Development
-----------

The source code is available at: <https://github.com/omniscale/imposm3/>

You can report any issues at: <https://github.com/omniscale/imposm3/issues>

License
-------

Imposm is released as open source under the Apache License 2.0. See LICENSE.

All dependencies included as source code are released under a BSD-ish license. See LICENSE.dep.

All dependencies included in binary releases are released under a BSD-ish license except the GEOS package.
The GEOS package is released as LGPL3 and is linked dynamically. See LICENSE.bin.


### Test ###

#### Unit tests ####

To run all unit tests:

    make test-unit


#### System tests ####

There are system test that import and update OSM data and verify the database content.
You need `osmosis` to create the test PBF files.
There is a Makefile that creates all test files if necessary and then runs the test itself.

    make test

Call `make test-system` to skip the unit tests.

WARNING: It uses your local PostgreSQL database (`imposm_test_import`, `imposm_test_production` and `imposm_test_backup` schema). Change the database with the standard `PGDATABASE`, `PGHOST`, etc. environment variables.
