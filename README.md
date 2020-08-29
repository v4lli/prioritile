# prioritile - an efficient slippy map XYZ/WMS tile priority fill implementation

<img align="left" width="215" height="159" src="https://user-images.githubusercontent.com/1577223/91644898-bdb84c80-ea40-11ea-904e-8bbf8156ab6d.png">

prioritile applies a painter-type algorithm in an efficient way by leveraging the XYZ (and WMTS) directory 
structure. All trailing tile source directives will be overlayed in the z-order specified. At least two (one base tileset + one overlay) source directives
are required. The zoom levels of all files must be the same.

Some assumptions about the directories:

- Tiles are RGBA PNGs
- NODATA is represented by 100% transparency
- All zoom levels are the same

## Usage

```prioritile [-parallel=4] /tiles/target/ /tiles/source1/ [/tiles/source2/ [...]]```
