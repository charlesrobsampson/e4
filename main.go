// e4 — command-line interface for the etak geographic address package.
package main

import (
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/charlesrobsampson/etak"
)

const usage = `e4 — 4-word geographic address encoder/decoder
(e4 is the CLI for the etak package — named after the Micronesian navigational concept)

USAGE
  e4 encode      <lat> <lon>
  e4 decode      <location>
  e4 fuzzy       <address> [--hint <lat,lon>] [--results <n>]
  e4 step        <address> <bearing> <distance> [unit]
  e4 dist        <address1> <address2> [unit]
  e4 bearing     <address1> <address2>
  e4 nav         <address1> <address2> [--unit <unit>] [--dir cardinal|compass|signed]
  e4 neighbors   <address>
  e4 interpolate <address1> <address2> <n>
  e4 cellsize    <address>

COMMANDS
  encode      Convert latitude/longitude to a 4-word address (~3 m resolution).

  decode      Convert any supported location format to a 4-word address.
              Auto-detects the input format — no flags needed.

              FORMATS ACCEPTED:
                etak address  slam.annoy.brim.polar
                              slam annoy brim polar  (spaces OK)
                lat/lon       40.7128 -74.006        (two args)
                              40.7128,-74.006        (comma-separated, one arg)
                Plus Code     87G7PX7V+4J
                UTM           18N 583959E 4507351N
                              18N 583959 4507351     (E/N suffix optional)

  fuzzy       Find candidate locations for a possibly garbled address.
              Tolerates wrong word order and 1-2 misheard/misspelled words.

  step        Travel from an address in a compass direction and get the new address.
              bearing: 0–360° clockwise from north (0=N, 90=E, 180=S, 270=W).
              unit: m (default), km, ft, yd, mi, nm.

  dist        Great-circle distance between two addresses.
              unit: m (default), km, ft, yd, mi, nm.

  bearing     Initial compass bearing (0–360°) from address1 to address2.

  nav         Decompose the route into north/south and east/west components.
              Useful for relating etak addresses to cardinal map directions.
              --unit: m (default), km, ft, yd, mi, nm
              --dir:  cardinal (default) — "north"/"south"/"east"/"west"
                      compass            — N/S/E/W
                      signed             — signed numeric values (+ or -)

  neighbors   The 8 immediately adjacent grid cells (~3 m apart).

  interpolate N evenly-spaced addresses along the great-circle route
              between two addresses, including both endpoints (n ≥ 2).

  cellsize    Geographic dimensions of the cell at an address.
              Cell height is constant (~3.14 m); width shrinks toward the poles.

FUZZY FLAGS
  --hint <lat,lon>   Coarse location hint (1-2 decimal places is enough).
                     Filters to within ~200 km and ranks closer matches higher.
  --results <n>      Maximum results to return (default: 5).

DIRECTION CONSTANTS
  N=0  NE=45  E=90  SE=135  S=180  SW=225  W=270  NW=315

EXAMPLES
  e4 encode 40.7128 -74.0060
  e4 decode slam.annoy.brim.polar
  e4 decode "87G7PX7V+4J"
  e4 decode "18N 583959 4507351"
  e4 decode 40.7128 -74.006
  e4 decode "40.7128,-74.006"
  e4 fuzzy  slm.anoy.brim.polr --hint 40.7,-74.0
  e4 step   slam.annoy.brim.polar 90 500
  e4 step   slam.annoy.brim.polar 45 1.5 km
  e4 dist   slam.annoy.brim.polar some.other.four.words km
  e4 bearing slam.annoy.brim.polar some.other.four.words
  e4 nav     slam.annoy.brim.polar some.other.four.words
  e4 nav     slam.annoy.brim.polar some.other.four.words --unit km
  e4 nav     slam.annoy.brim.polar some.other.four.words --unit mi --dir compass
  e4 neighbors slam.annoy.brim.polar
  e4 interpolate slam.annoy.brim.polar some.other.four.words 5
  e4 cellsize slam.annoy.brim.polar
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n\nRun 'e4' with no arguments for usage.\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(usage)
		return nil
	}
	switch strings.ToLower(args[0]) {
	case "encode":
		return cmdEncode(args[1:])
	case "decode":
		return cmdDecode(args[1:])
	case "fuzzy":
		return cmdFuzzy(args[1:])
	case "step":
		return cmdStep(args[1:])
	case "dist", "distance":
		return cmdDist(args[1:])
	case "bearing":
		return cmdBearing(args[1:])
	case "nav":
		return cmdNav(args[1:])
	case "neighbors", "neighbours":
		return cmdNeighbors(args[1:])
	case "interpolate", "interp":
		return cmdInterpolate(args[1:])
	case "cellsize", "cell":
		return cmdCellSize(args[1:])
	case "-h", "--help", "help":
		fmt.Print(usage)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

// ── encode ────────────────────────────────────────────────────────────────────

func cmdEncode(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("encode requires exactly 2 arguments: <lat> <lon>")
	}
	lat, err := parseFloat(args[0], "lat")
	if err != nil {
		return err
	}
	lon, err := parseFloat(args[1], "lon")
	if err != nil {
		return err
	}
	addr, err := etak.Encode(lat, lon)
	if err != nil {
		return err
	}
	fmt.Println(addr)
	return nil
}

// ── decode ────────────────────────────────────────────────────────────────────

func cmdDecode(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("decode requires a location argument")
	}

	lat, lon, format, err := resolveLocation(args)
	if err != nil {
		return err
	}

	addr, err := etak.Encode(lat, lon)
	if err != nil {
		return err
	}

	// If the input was already an etak address, decode shows it as-is.
	// Otherwise show the detected format so the user knows what was parsed.
	if format != "etak" {
		fmt.Printf("format:  %s\n", format)
	}
	fmt.Printf("address: %s\n", addr)
	fmt.Printf("lat:     %+.6f\n", lat)
	fmt.Printf("lon:     %+.6f\n", lon)
	fmt.Printf("maps:    %s\n", mapsURL(lat, lon))

	// Show cross-format output for non-etak inputs.
	if format != "pluscode" {
		pc, _ := etak.LatLonToPlusCode(lat, lon)
		fmt.Printf("pluscode: %s\n", pc)
	}
	if format != "utm" {
		utm, err := etak.LatLonToUTM(lat, lon)
		if err == nil {
			fmt.Printf("utm:     %s\n", utm)
		}
	}
	return nil
}

// ── fuzzy ─────────────────────────────────────────────────────────────────────

func cmdFuzzy(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("fuzzy requires an address argument")
	}
	var addrParts []string
	hintLat, hintLon := math.NaN(), math.NaN()
	maxResults := 5

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--hint":
			if i+1 >= len(args) {
				return fmt.Errorf("--hint requires a value, e.g. --hint 40.7,-74.0")
			}
			i++
			var err error
			hintLat, hintLon, err = parseHint(args[i])
			if err != nil {
				return err
			}
		case "--results":
			if i+1 >= len(args) {
				return fmt.Errorf("--results requires a value, e.g. --results 10")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 1 {
				return fmt.Errorf("--results must be a positive integer, got %q", args[i])
			}
			maxResults = n
		default:
			if strings.HasPrefix(args[i], "--") {
				return fmt.Errorf("unknown flag %q", args[i])
			}
			addrParts = append(addrParts, args[i])
		}
	}
	if len(addrParts) == 0 {
		return fmt.Errorf("fuzzy requires an address argument")
	}

	addr := strings.Join(addrParts, ".")
	if !math.IsNaN(hintLat) {
		fmt.Printf("searching near %.2f, %.2f …\n\n", hintLat, hintLon)
	}

	results, err := etak.FuzzySearch(addr, hintLat, hintLon, maxResults)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Println("no matches found")
		if !math.IsNaN(hintLat) {
			fmt.Println("try removing --hint to search without a location filter")
		}
		return nil
	}

	addrW := maxLen("address", mapStrings(results, func(r etak.FuzzyResult) string { return r.Address }))
	fmt.Printf("%-*s   %+10s  %+11s   %s\n", addrW, "address", "lat", "lon", "maps")
	fmt.Printf("%s   %s  %s   %s\n", rep("─", addrW), rep("─", 10), rep("─", 11), rep("─", 40))
	for _, r := range results {
		fmt.Printf("%-*s   %+10.6f  %+11.6f   %s\n", addrW, r.Address, r.Lat, r.Lon, mapsURL(r.Lat, r.Lon))
	}
	return nil
}

// ── step ──────────────────────────────────────────────────────────────────────

func cmdStep(args []string) error {
	if len(args) < 3 || len(args) > 4 {
		return fmt.Errorf("step requires 3-4 arguments: <address> <bearing> <distance> [unit]\n  e.g.: e4 step slam.annoy.brim.polar 90 500\n        e4 step slam.annoy.brim.polar 45 1.5 km")
	}
	addr := args[0]
	bearing, err := parseFloat(args[1], "bearing")
	if err != nil {
		return err
	}
	dist, err := parseFloat(args[2], "distance")
	if err != nil {
		return err
	}
	unit := "m"
	if len(args) == 4 {
		unit = args[3]
	}

	next, err := etak.Step(addr, bearing, dist, unit)
	if err != nil {
		return err
	}
	lat, lon, _ := etak.Decode(next)
	fmt.Printf("address: %s\n", next)
	fmt.Printf("lat:     %+.6f\n", lat)
	fmt.Printf("lon:     %+.6f\n", lon)
	fmt.Printf("maps:    %s\n", mapsURL(lat, lon))
	return nil
}

// ── dist ──────────────────────────────────────────────────────────────────────

func cmdDist(args []string) error {
	if len(args) < 2 || len(args) > 3 {
		return fmt.Errorf("dist requires 2-3 arguments: <address1> <address2> [unit]")
	}
	unit := "m"
	if len(args) == 3 {
		unit = args[2]
	}
	d, err := etak.Distance(args[0], args[1], unit)
	if err != nil {
		return err
	}
	fmt.Printf("%.4f %s\n", d, unit)
	return nil
}

// ── bearing ───────────────────────────────────────────────────────────────────

func cmdBearing(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("bearing requires exactly 2 arguments: <address1> <address2>")
	}
	b, err := etak.Bearing(args[0], args[1])
	if err != nil {
		return err
	}
	fmt.Printf("%.2f° %s\n", b, compassPoint(b))
	return nil
}

// ── neighbors ─────────────────────────────────────────────────────────────────

func cmdNeighbors(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("neighbors requires exactly 1 argument: <address>")
	}
	nb, err := etak.Neighbors(args[0])
	if err != nil {
		return err
	}

	addrW := len(args[0])
	for _, a := range nb {
		if len(a) > addrW {
			addrW = len(a)
		}
	}

	fmt.Printf("  %-*s  %-*s  %-*s\n", addrW, nb[etak.DirNW], addrW, nb[etak.DirN], addrW, nb[etak.DirNE])
	fmt.Printf("  %-*s  %-*s  %-*s\n", addrW, nb[etak.DirW], addrW, args[0], addrW, nb[etak.DirE])
	fmt.Printf("  %-*s  %-*s  %-*s\n", addrW, nb[etak.DirSW], addrW, nb[etak.DirS], addrW, nb[etak.DirSE])
	fmt.Println()

	dirNames := [8]string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	fmt.Printf("%-2s  %-*s   %s\n", "  ", addrW, "address", "maps")
	fmt.Printf("%-2s  %s   %s\n", "  ", rep("─", addrW), rep("─", 40))
	fmt.Printf("%-2s  %-*s   (centre)\n", "·", addrW, args[0])
	for i, a := range nb {
		lat, lon, _ := etak.Decode(a)
		fmt.Printf("%-2s  %-*s   %s\n", dirNames[i], addrW, a, mapsURL(lat, lon))
	}
	return nil
}

// ── interpolate ───────────────────────────────────────────────────────────────

func cmdInterpolate(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("interpolate requires exactly 3 arguments: <address1> <address2> <n>")
	}
	n, err := strconv.Atoi(args[2])
	if err != nil || n < 2 {
		return fmt.Errorf("<n> must be an integer ≥ 2, got %q", args[2])
	}

	points, err := etak.Interpolate(args[0], args[1], n)
	if err != nil {
		return err
	}

	addrW := maxLen("address", points)
	totalDist, _ := etak.Distance(args[0], args[1], "m")

	fmt.Printf("%-3s  %-*s   %+10s  %+11s   %s\n", "#", addrW, "address", "lat", "lon", "maps")
	fmt.Printf("%-3s  %s   %s  %s   %s\n", rep("─", 3), rep("─", addrW), rep("─", 10), rep("─", 11), rep("─", 40))
	for i, p := range points {
		lat, lon, _ := etak.Decode(p)
		label := fmt.Sprintf("%d", i+1)
		if i == 0 || i == len(points)-1 {
			label += " ←"
		}
		fmt.Printf("%-4s %-*s   %+10.6f  %+11.6f   %s\n", label, addrW, p, lat, lon, mapsURL(lat, lon))
	}
	fmt.Printf("\ntotal distance: %.2f m\n", totalDist)
	return nil
}

// ── cellsize ──────────────────────────────────────────────────────────────────

func cmdCellSize(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("cellsize requires exactly 1 argument: <address>")
	}
	info, err := etak.CellSize(args[0])
	if err != nil {
		return err
	}
	fmt.Printf("address:  %s\n", info.Address)
	fmt.Printf("lat:      %+.6f\n", info.Lat)
	fmt.Printf("lon:      %+.6f\n", info.Lon)
	fmt.Printf("height:   %.4f m  (north-south, constant)\n", info.HeightM)
	fmt.Printf("width:    %.4f m  (east-west at this latitude)\n", info.WidthM)
	fmt.Printf("area:     %.4f m²\n", info.AreaM2)
	return nil
}

// ── format detection & resolution ────────────────────────────────────────────

// resolveLocation parses one or more args as any supported location format and
// returns lat, lon, and the detected format name.
//
// Accepted formats (tried in order):
//  1. Plus Code  — single arg containing '+'
//  2. UTM        — single arg or two-arg "zone+hemi easting northing"
//  3. lat/lon    — two float args, or one "float,float" arg
//  4. etak address — one or more words joined with dots
func resolveLocation(args []string) (lat, lon float64, format string, err error) {
	joined := strings.Join(args, " ")

	// 1. Plus Code
	if etak.IsPlusCode(joined) {
		lat, lon, err = etak.PlusCodeToLatLon(joined)
		if err != nil {
			return 0, 0, "", fmt.Errorf("invalid Plus Code: %w", err)
		}
		return lat, lon, "Plus Code", nil
	}
	if len(args) == 1 && etak.IsPlusCode(args[0]) {
		lat, lon, err = etak.PlusCodeToLatLon(args[0])
		if err != nil {
			return 0, 0, "", fmt.Errorf("invalid Plus Code: %w", err)
		}
		return lat, lon, "Plus Code", nil
	}

	// 2. UTM — single arg "18N 583959 4507351" or split across args
	utmRe := regexp.MustCompile(`(?i)^\d{1,2}\s*[NS]`)
	if utmRe.MatchString(joined) && etak.IsUTM(joined) {
		u, e := etak.ParseUTM(joined)
		if e == nil {
			lat, lon, err = etak.UTMToLatLon(u)
			if err != nil {
				return 0, 0, "", err
			}
			return lat, lon, "UTM", nil
		}
	}

	// 3. lat/lon — two separate float args OR "float,float"
	if len(args) == 2 {
		la, e1 := strconv.ParseFloat(args[0], 64)
		lo, e2 := strconv.ParseFloat(args[1], 64)
		if e1 == nil && e2 == nil {
			return la, lo, "lat/lon", nil
		}
	}
	if len(args) == 1 && strings.Contains(args[0], ",") {
		parts := strings.SplitN(args[0], ",", 2)
		la, e1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		lo, e2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if e1 == nil && e2 == nil {
			return la, lo, "lat/lon", nil
		}
	}

	// 4. etak address — join with dots and try to decode
	addr := strings.Join(args, ".")
	lat, lon, err = etak.Decode(addr)
	if err != nil {
		var unknown *etak.ErrUnknownWord
		if errors.As(err, &unknown) {
			return 0, 0, "", fmt.Errorf(
				"%w\n\nCould not detect format. Accepted inputs:\n"+
					"  etak address: slam.annoy.brim.polar\n"+
					"  lat/lon:      40.7128 -74.006\n"+
					"  Plus Code:    87G7PX7V+4J\n"+
					"  UTM:          18N 583959 4507351", err)
		}
		return 0, 0, "", err
	}
	return lat, lon, "etak", nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func parseFloat(s, name string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: must be a decimal number", name, s)
	}
	return v, nil
}

func parseHint(s string) (lat, lon float64, err error) {
	parts := strings.SplitN(s, ",", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("--hint must be <lat,lon>, e.g. 40.7,-74.0 (got %q)", s)
	}
	lat, err = parseFloat(strings.TrimSpace(parts[0]), "hint lat")
	if err != nil {
		return
	}
	lon, err = parseFloat(strings.TrimSpace(parts[1]), "hint lon")
	return
}

func mapsURL(lat, lon float64) string {
	return fmt.Sprintf("https://maps.google.com/?q=%+.6f,%+.6f", lat, lon)
}

func compassPoint(b float64) string {
	points := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	return points[int((b+22.5)/45)%8]
}

func rep(s string, n int) string { return strings.Repeat(s, n) }

func maxLen(base string, ss []string) int {
	n := len(base)
	for _, s := range ss {
		if len(s) > n {
			n = len(s)
		}
	}
	return n
}

func mapStrings[T any](slice []T, f func(T) string) []string {
	out := make([]string, len(slice))
	for i, v := range slice {
		out[i] = f(v)
	}
	return out
}

// ── nav ───────────────────────────────────────────────────────────────────────

func cmdNav(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("nav requires two address arguments")
	}

	// Parse flags after the two address args.
	unit := "m"
	dir := "cardinal"
	addr1 := args[0]
	addr2 := args[1]

	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--unit":
			if i+1 >= len(args) {
				return fmt.Errorf("--unit requires a value (m, km, ft, yd, mi, nm)")
			}
			i++
			unit = args[i]
		case "--dir":
			if i+1 >= len(args) {
				return fmt.Errorf("--dir requires a value (cardinal, compass, signed)")
			}
			i++
			dir = args[i]
			if dir != "cardinal" && dir != "compass" && dir != "signed" {
				return fmt.Errorf("--dir must be cardinal, compass, or signed (got %q)", dir)
			}
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}

	r, err := etak.NavInUnit(addr1, addr2, unit)
	if err != nil {
		return err
	}

	nsAbs := math.Abs(r.NSMetres)
	ewAbs := math.Abs(r.EWMetres)

	switch dir {
	case "cardinal":
		fmt.Printf("%-6s  %s %s\n", r.NSDirection(), formatDist(nsAbs, unit), unit)
		fmt.Printf("%-6s  %s %s\n", r.EWDirection(), formatDist(ewAbs, unit), unit)
	case "compass":
		fmt.Printf("%s  %s %s\n", r.NSCompass(), formatDist(nsAbs, unit), unit)
		fmt.Printf("%s  %s %s\n", r.EWCompass(), formatDist(ewAbs, unit), unit)
	case "signed":
		nsSign := "+"
		if r.NSMetres < 0 {
			nsSign = "-"
		}
		ewSign := "+"
		if r.EWMetres < 0 {
			ewSign = "-"
		}
		fmt.Printf("%s%s %s\n", nsSign, formatDist(nsAbs, unit), unit)
		fmt.Printf("%s%s %s\n", ewSign, formatDist(ewAbs, unit), unit)
	}
	return nil
}

// formatDist formats a distance value for display.
// Uses decimal notation with enough precision to be useful but not noisy.
func formatDist(v float64, unit string) string {
	// For metres show 3 decimal places; for larger units show 4.
	if unit == "m" {
		return fmt.Sprintf("%.3f", v)
	}
	return fmt.Sprintf("%.4f", v)
}
