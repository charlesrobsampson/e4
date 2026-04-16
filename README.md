# e4

Command-line interface for [etak](https://github.com/charlesrobsampson/etak) —
a 4-word geographic address encoder.

**Why `e4`?** The binary is named `e4` as a short form of `etak`, with the
`4` referencing the four-word encoding. It is short enough to type comfortably
as a terminal command while remaining distinct from other common CLI tools.

## Install

```bash
go install github.com/charlesrobsampson/e4@latest
```

## Commands

```
e4 encode      <lat> <lon>
e4 decode      <location>
e4 fuzzy       <address> [--hint <lat,lon>] [--results <n>]
e4 step        <address> <bearing> <distance> [unit]
e4 dist        <address1> <address2> [unit]
e4 bearing     <address1> <address2>
e4 neighbors   <address>
e4 interpolate <address1> <address2> <n>
e4 cellsize    <address>
```

## Examples

```bash
# Encode coordinates
e4 encode 40.7128 -74.0060

# Decode — auto-detects format
e4 decode slam.annoy.brim.polar
e4 decode "87G7PX7V+4J"
e4 decode "18N 583959 4507351"
e4 decode 40.7128 -74.006
e4 decode "40.7128,-74.006"

# Fuzzy search — tolerates misheard or misordered words
e4 fuzzy slm.anoy.brim.polr
e4 fuzzy polar.brim.annoy.slam --hint 40.7,-74.0 --results 3

# Navigate
e4 step  slam.annoy.brim.polar 90 500         # 500 m east
e4 step  slam.annoy.brim.polar 45 1.5 km      # 1.5 km northeast
e4 dist  slam.annoy.brim.polar other.four.word.addr km
e4 bearing slam.annoy.brim.polar other.four.word.addr

# Grid
e4 neighbors   slam.annoy.brim.polar
e4 interpolate slam.annoy.brim.polar other.four.word.addr 5
e4 cellsize    slam.annoy.brim.polar
```

## decode auto-detects input format

```
etak address   slam.annoy.brim.polar
Plus Code       87G7PX7V+4J
UTM             18N 583959E 4507351N
lat/lon         40.7128 -74.006  or  40.7128,-74.006
```

Output always includes the etak address, lat/lon, Google Maps link, Plus Code,
and UTM.

## Accepted distance units

`m` (default), `km`, `ft`, `yd`, `mi`, `nm`

## Direction reference

```
N=0  NE=45  E=90  SE=135  S=180  SW=225  W=270  NW=315
```
