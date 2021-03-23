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

All source directives are overlayed in the z-order specified on the command line. The first path specification is the base layer (and the output). See the 

```prioritile [-q] [-best-effort] [-parallel=4] /tiles/target/ /tiles/source1/ [s3://foo/tiles/source2/ [...]]```
