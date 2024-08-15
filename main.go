/*
ctap is a lightweight, portable colouriser for TAP
(Test-Anything-Protocol) output
*/
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gookit/color"
	flags "github.com/jessevdk/go-flags"
)

const (
	testFailExitCode = 3
	planFailExitCode = 4
	bailExitCode     = 5

	glyphOK  = "\u2713"
	glyphNOK = "\u2717"

	// Default colours
	defaultCUnknown  = "default"
	defaultCVersion  = "cyan"
	defaultCPlan     = "white"
	defaultCOk       = "green"
	defaultCFail     = "red bold"
	defaultCDiag     = "gray"
	defaultCBail     = "yellow bold"
	defaultCSummOk   = "green bold"
	defaultCSummFail = "red bold"
	defaultCPlanFail = "magenta bold"

	// Usage addendum
	usageAddendum = `
Colour strings may be any of the following colour names:

  red, green, blue, yellow, cyan, magenta, white, black, gray, default

They may also be hex colour strings like "#cc9900" or "#c90" (with the
leading "#" optional).

Colour names or hex strings can also have any of the following modifiers
appended to them (space-separated):

  bold, italic, underscore, reverse, blink, concealed, fuzzy

(though how they work will depend on your terminal support)
`
)

type options struct {
	Failures bool   `short:"f" long:"failures" description:"show test failures (suppress TAP successes)"`
	Glyphs   bool   `short:"g" long:"glyphs" description:"show \u2713\u2717 glyphs instead of 'ok/not ok' in TAP output"`
	Summary  bool   `short:"s" long:"summary" description:"append a Test::Harness-like summary of the test results"`
	CVersion string `short:"V" long:"cversion" description:"colour to use for version lines"`
	CPlan    string `short:"P" long:"cplan" description:"colour to use for plan lines"`
	COk      string `short:"O" long:"cok" description:"colour to use for test ok lines"`
	CFail    string `short:"F" long:"cfail" description:"colour to use for test fail/not ok lines"`
	CDiag    string `short:"D" long:"cdiag" description:"colour to use for diagnostic lines"`
	CBail    string `short:"B" long:"cbail" description:"colour to use for bail out lines"`
	Args     struct {
		TapFile string
	} `positional-args:"yes"`
}

var opts options

var (
	reVersion    = regexp.MustCompile(`^TAP version (\d+)`)
	rePlan       = regexp.MustCompile(`^(\d+)..(\d+)\s*(?:#\s*(.*?)\s*)?$`)
	reTest       = regexp.MustCompile(`^(ok|not ok)(?:\pZ+(\d+))?(?:\pZ+([^#]+))?(?:\pZ+(#\pZ*(.*?)))?\pZ*?$`)
	reDiagnostic = regexp.MustCompile(`^#`)
	reBail       = regexp.MustCompile(`^Bail out!(?:\pZ*(.*?))?\pZ*$`)

	reTestPrefix = regexp.MustCompile(`^(ok|not ok)\pZ*`)

	reHexColour = regexp.MustCompile(`(?i)^#?([0-9a-f]{6}|[0-9a-f]{3})$`)

	colourStringMap = map[string]color.Color{
		"red":     color.FgRed,
		"blue":    color.FgBlue,
		"green":   color.FgGreen,
		"yellow":  color.FgYellow,
		"cyan":    color.FgCyan,
		"magenta": color.FgMagenta,
		"white":   color.FgWhite,
		"black":   color.FgBlack,
		"gray":    color.FgGray,
		"default": color.FgDefault,
	}
	colourOptMap = map[string]color.Color{
		"bold":       color.OpBold,
		"italic":     color.OpItalic,
		"underscore": color.OpUnderscore,
		"blink":      color.OpBlink,
		"concealed":  color.OpConcealed,
		"fuzzy":      color.OpFuzzy,
		"reverse":    color.OpReverse,
	}
)

type lineType int

const (
	tapUnknown lineType = iota
	tapVersion
	tapPlan
	tapTestOK
	tapTestNOK
	tapDiagnostic
	tapBail
	tapSummaryOK
	tapSummaryNOK
	tapPlanNOK
)

func (t lineType) String() string {
	return [...]string{
		"Unknown", "Version", "Plan", "TestOK", "TestNOK",
		"Diag", "Bail", "SummaryOK", "SummaryNOK", "PlanNOK"}[t]
}

type lineRecord struct {
	Type        lineType
	PlanFirst   int // Plan
	PlanLast    int // Plan
	TestNum     int // Test
	Description string
	Directive   string
}

func parseColour(c string) (color.PrinterFace, error) {
	// Extract colour+options from c
	var colourStr string
	var options []color.Color
	for _, t := range strings.Split(c, " ") {
		o, ok := colourOptMap[t]
		if ok {
			options = append(options, o)
			continue
		}
		// Error if more than one colour found
		if colourStr != "" {
			return nil, fmt.Errorf("multiple colours in string %q?", c)
		}
		colourStr = t
	}

	// Convert colour+options to a style
	if reHexColour.MatchString(colourStr) {
		style := color.HEXStyle(colourStr)
		if len(options) > 0 {
			style.AddOpts(options...)
		}
		return style, nil
	}
	colour, ok := colourStringMap[colourStr]
	if !ok {
		return nil, fmt.Errorf("bad colour string %q", colourStr)
	}
	if len(options) > 0 {
		options = append([]color.Color{colour}, options...)
		return color.New(options...), nil
	}
	return color.New(colour), nil
}

func getColour(defaultColour, optColour string) color.PrinterFace {
	if optColour != "" {
		c, err := parseColour(optColour)
		if err == nil {
			return c
		}
	}
	c, err := parseColour(defaultColour)
	if err != nil {
		log.Fatal(err)
	}
	return c
}

type colourMap map[lineType]color.PrinterFace

func getColourMap(opt options) colourMap {
	cmap := make(colourMap)
	cmap[tapUnknown] = getColour(defaultCUnknown, "")
	cmap[tapVersion] = getColour(defaultCVersion, opt.CVersion)
	cmap[tapPlan] = getColour(defaultCPlan, opt.CPlan)
	cmap[tapTestOK] = getColour(defaultCOk, opt.COk)
	cmap[tapTestNOK] = getColour(defaultCFail, opt.CFail)
	cmap[tapDiagnostic] = getColour(defaultCDiag, opt.CDiag)
	cmap[tapBail] = getColour(defaultCBail, opt.CBail)
	cmap[tapSummaryOK] = getColour(defaultCSummOk, "")
	cmap[tapSummaryNOK] = getColour(defaultCSummFail, "")
	cmap[tapPlanNOK] = getColour(defaultCPlanFail, "")
	return cmap
}

func parseLine(text string) lineRecord {
	if matches := reVersion.FindStringSubmatch(text); matches != nil {
		return lineRecord{Type: tapVersion}
	}
	if matches := rePlan.FindStringSubmatch(text); matches != nil {
		line := lineRecord{Type: tapPlan}
		if planfirst := matches[1]; planfirst != "" {
			if i, err := strconv.Atoi(planfirst); err == nil {
				line.PlanFirst = i
			}
		}
		if planlast := matches[2]; planlast != "" {
			if i, err := strconv.Atoi(planlast); err == nil {
				line.PlanLast = i
			}
		}
		return line
	}
	if matches := reTest.FindStringSubmatch(text); matches != nil {
		line := lineRecord{}
		res := matches[1]
		if testno := matches[2]; testno != "" {
			i, err := strconv.Atoi(testno)
			if err == nil {
				line.TestNum = i
			}
		}
		switch res {
		case "ok":
			line.Type = tapTestOK
		case "not ok":
			line.Type = tapTestNOK
		}
		return line
	}
	if matches := reDiagnostic.FindStringSubmatch(text); matches != nil {
		return lineRecord{Type: tapDiagnostic}
	}
	if matches := reBail.FindStringSubmatch(text); matches != nil {
		return lineRecord{Type: tapBail}
	}
	return lineRecord{Type: tapUnknown}
}

func failureString(failures []int) string {
	var sb strings.Builder
	for i, n := range failures {
		if i == 0 {
			fmt.Fprintf(&sb, "%d", n)
		} else {
			fmt.Fprintf(&sb, ", %d", n)
		}
	}
	return sb.String()
}

func cprintln(text string, linetype lineType, cmap colourMap, opts options) {
	if opts.Failures && linetype == tapTestOK {
		return
	}
	if opts.Glyphs {
		// Replace `ok/not ok` (or prepend) glyphs
		switch linetype {
		case tapTestOK:
			text = reTestPrefix.ReplaceAllString(text, glyphOK+" ")
		case tapTestNOK:
			text = reTestPrefix.ReplaceAllString(text, glyphNOK+" ")
		case tapBail:
			text = glyphNOK + " " + text
		}
	}
	cfmt, ok := cmap[linetype]
	if !ok {
		log.Fatalf("no formatter defined for linetype %q: %s\n",
			linetype.String(), text)
	}
	cfmt.Println(text)
}

func printSummary(failures []int, testnum int, planNOK bool, cmap colourMap, opts options) {
	plural := ""
	glyph := ""

	if len(failures) > 0 {
		if len(failures) > 1 {
			plural = "s"
		}
		if opts.Glyphs {
			glyph = glyphNOK + " "
		}
		cmap[tapSummaryNOK].Printf("%sFAILED test%s: %s\n",
			glyph, plural,
			failureString(failures))
		cmap[tapSummaryNOK].Printf("%sFailed %d/%d tests, %0.02f%% ok\n",
			glyph, len(failures), testnum,
			float64(testnum-len(failures))*100/float64(testnum))
	} else if !planNOK {
		if opts.Glyphs {
			glyph = glyphOK + " "
		}
		cmap[tapSummaryOK].Printf("%sPassed %d/%d tests, 100%% ok\n",
			glyph, testnum, testnum)
	}
}

func printAppends(failures []int, testnum, planLast, exitCode int,
	cmap colourMap, opts options) int {
	planNOK := testnum == 0 || testnum != planLast
	if planNOK && exitCode < planFailExitCode {
		exitCode = planFailExitCode
	}

	if opts.Summary {
		printSummary(failures, testnum, planNOK, cmap, opts)
	}

	// Fail if we haven't seen all planned tests
	if planNOK {
		glyph := ""
		if opts.Glyphs {
			glyph = glyphNOK + " "
		}
		if testnum == 0 {
			cmap[tapPlanNOK].Printf("%sFailed plan: no tests seen\n", glyph)
		} else {
			cmap[tapPlanNOK].Printf("%sFailed plan: only %d/%d planned tests seen\n",
				glyph, testnum, planLast)
		}
	}

	return exitCode
}

func runCLI(opts options, ofh io.Writer) int {
	// Setup
	log.SetFlags(0)
	var fh *os.File
	var err error
	if opts.Args.TapFile != "" {
		fh, err = os.Open(opts.Args.TapFile)
		if err != nil {
			log.Fatal(err)
		}
		defer fh.Close()
	} else {
		fh = os.Stdin
	}
	scanner := bufio.NewScanner(fh)

	// Setup colours
	color.SetOutput(ofh)
	// Force colours in CI environments
	if _, ok := os.LookupEnv("CI"); ok {
		color.ForceOpenColor()
	}
	cmap := getColourMap(opts)

	// Process input
	var planLast int
	testnum := 0
	failures := []int{}
	exitCode := 0
	for scanner.Scan() {
		text := scanner.Text()

		line := parseLine(text)
		cprintln(text, line.Type, cmap, opts)

		switch line.Type {
		case tapPlan:
			planLast = line.PlanLast
		case tapTestOK, tapTestNOK:
			if line.TestNum > 0 {
				testnum = line.TestNum
			} else {
				testnum++
			}
			if line.Type == tapTestNOK {
				failures = append(failures, testnum)
				if exitCode < testFailExitCode {
					exitCode = testFailExitCode
				}
			}
		case tapBail:
			if exitCode < bailExitCode {
				exitCode = bailExitCode
			}
		}
	}

	exitCode = printAppends(failures, testnum, planLast, exitCode, cmap, opts)

	return exitCode
}

func main() {
	// Parse default options are HelpFlag | PrintErrors | PassDoubleDash
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		if flags.WroteHelp(err) {
			fmt.Print(usageAddendum)
			os.Exit(0)
		}

		// Does PrintErrors work? Is it not set?
		fmt.Fprintln(os.Stderr, "")
		parser.WriteHelp(os.Stderr)
		os.Exit(2)
	}

	exitCode := runCLI(opts, os.Stdout)
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
