# prioritile - efficient slippy map XYZ/WMS tile priority fill implementation

<img align="left" width="215" height="159" src="https://user-images.githubusercontent.com/1577223/91644898-bdb84c80-ea40-11ea-904e-8bbf8156ab6d.png">

prioritile applies a painter-type algorithm in an efficient way by leveraging the XYZ (and WM(T)S) directory structure on local and remote file systems. It is intended to replace complicated and inefficient GDAL VRT chains which are sometimes used to create a "world grids" of e.g. satellite imagery. XYZ/WMS tile directories can be created with e.g. [gdal2tiles](https://gdal.org/programs/gdal2tiles.html) and viewed with any slippy map software (OpenLayers or Leaflet) or GIS Software (e.g. QGIS).

prioritile supports S3 storage backends for both input and output tilesets, as well as mixed configurations.

## Limitations

At least two (one base tileset + one overlay) source directives are required (obviously). Some assumptions about the tiles and structure:

- All files are RGBA PNGs
- "No data" is represented by 100% transparency
- All zoom levels are the same (no up or downsampling supported)

## Installation

`go build` 😜

(You can try `make` to run prioritile on the included demo dataset).

## Usage

All source directives are overlayed in the z-order specified on the command line. The first path specification is both the base layer and the output:

```prioritile [-q] [-parallel=4] /tiles/target/ /tiles/source1/ [s3://foo/tiles/source2/ [...]]```
