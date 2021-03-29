# prioritile - efficient slippy map XYZ/WMS tile priority fill implementation

<img align="left" width="215" height="159" src="https://user-images.githubusercontent.com/1577223/91644898-bdb84c80-ea40-11ea-904e-8bbf8156ab6d.png">

prioritile applies a painter-type algorithm in an efficient way by
leveraging the XYZ (and WM(T)S) directory structure on local and
remote file systems. It is intended to replace complicated and
inefficient GDAL VRT chains which are sometimes used to create a
"world grids" of e.g. satellite imagery. XYZ/WMS tile directories
can be created with e.g.
[gdal2tiles](https://gdal.org/programs/gdal2tiles.html) and viewed
with any slippy map software (OpenLayers or Leaflet) or GIS Software
(e.g. QGIS).

prioritile supports S3 storage backends for both input and output
tilesets, as well as mixed configurations. prioritile was developed
for and is used by [meteocool](https://meteocool.com/).

![Go](https://github.com/v4lli/prioritile/workflows/Go/badge.svg)

## Limitations

At least two (one base tileset + one overlay) source directives are
required (obviously). Some assumptions about the tiles and structure:

- All files are RGBA PNGs
- "No data" is represented by 100% transparency
- All zoom levels are the same (no up or downsampling supported)
- Tile resolution is equal in target and source tilesets

## Installation

`go get -u github.com/v4lli/prioritile`

- Alternatively, clone the repo and run `go build`.
- Run `make` to execute prioritile on the included demo dataset.
- Note that [prioritile is provided as a Docker base
layer](https://github.com/users/v4lli/packages/container/package/prioritle)
which is updated automatically through Github Actions. Use this in
your `Dockerfile`:

```
COPY --from=ghcr.io/v4lli/prioritile:latest /app/prioritile /bin/
```

## Usage

All source directives are overlayed in the z-order specified on the command line. The first path specification is the base layer (and the output).

```
Usage: prioritile [-zoom '1-8'] [-debug] [-report] [-best-effort] [-parallel=2] /tiles/target/ /tiles/source1/ [https://foo.com/tiles/source2/ [...]]

prioritile applies a painter-type algorithm to the first tiles location specified
on the commandline in an efficient way by leveraging the XYZ (and WMTS) directory
structure. All trailing tile source directives will be used by the algorithm, in the
z-order specified. At least two (one base tileset + one overlay) source directives
are required. The zoom levels of all files must be the same.
Some assumptions about the source directories:
- Tiles are RGBA PNGs
- NODATA is represented by 100% alpha
- Resolution of corresponding tiles matches

S3 disk backends are supported as source and target, e.g. 'https://example.com[:port]/foobucket/'.
S3 authentication information is read from environment variables prefixed with the target hostname:
example.com[:port]_ACCESS_KEY_ID, example.com[:port]_SECRET_ACCESS_KEY

  -best-effort
    	Best-effort merging: ignore erroneous tilesets completely and silently skip single failed tiles.
  -debug
    	Enable debugging (tracing and some perf counters)
  -parallel int
    	Number of parallel threads to use for processing (default 2)
  -quiet
    	Don't output progress information
  -report
    	Enable periodic reports (every min); intended for non-interactive environments
  -zoom string
    	Restrict/manually set zoom levels to work on, in the form of 'minZ-maxZ' (e.g. '1-8'). If this option is specified, prioritile does not try to automatically detect the zoom levels of the target but rather uses these hardcoded ones.
```

## Further Reading

- https://wiki.openstreetmap.org/wiki/Slippy_map_tilenames
- https://github.com/chrislusf/seaweedfs
- https://gdal.org/programs/gdal2tiles.html

## Credits etc

- Example datasets, including the screenshot above, are courtesy of the ESA/EC Copernicus Programme (Contains Modified Copernicus Sentinel data, 2020/2021)
- [Code is MIT](https://github.com/v4lli/prioritile/blob/master/LICENSE) - contributions welcome 😊
