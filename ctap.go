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
	TestFailExitCode = 3
	PlanFailExitCode = 4
	BailExitCode     = 5

	GlyphOK  = "\u2713"
	GlyphNOK = "\u2717"
)

type Options struct {
	Failures bool `short:"f" long:"failures" description:"show test failures (suppress TAP successes)"`
	Glyphs   bool `short:"g" long:"glyphs" description:"show \u2713\u2717 glyphs instead of 'ok/not ok' in TAP output"`
	Summary  bool `short:"s" long:"summary" description:"append a Test::Harness-like summary of the test results"`
	Args     struct {
		TapFile string
	} `positional-args:"yes"`
}

var opts Options

var (
	reVersion    = regexp.MustCompile(`^TAP version (\d+)`)
	rePlan       = regexp.MustCompile(`^(\d+)..(\d+)\s*(?:#\s*(.*?)\s*)?$`)
	reTest       = regexp.MustCompile(`^(ok|not ok)(?:\pZ+(\d+))?(?:\pZ+([^#]+))?(?:\pZ+(#\pZ*(.*?)))?\pZ*?$`)
	reDiagnostic = regexp.MustCompile(`^#`)
	reBail       = regexp.MustCompile(`^Bail out!(?:\pZ*(.*?))?\pZ*$`)

	reTestPrefix = regexp.MustCompile(`^(ok|not ok)\pZ*`)
)

type LineType int

const (
	Unknown LineType = iota
	Version
	Plan
	TestOK
	TestNOK
	Diagnostic
	Bail
	SummaryOK
	SummaryNOK
	PlanNOK
)

func (t LineType) String() string {
	return [...]string{
		"Unknown", "Version", "Plan", "TestOK", "TestNOK",
		"Diag", "Bail", "SummaryOK", "SummaryNOK", "PlanNOK"}[t]
}

type Line struct {
	Type        LineType
	PlanFirst   int // Plan
	PlanLast    int // Plan
	TestNum     int // Test
	Description string
	Directive   string
}

type ColourMap map[LineType]color.PrinterFace

func colourMap(opt Options) ColourMap {
	cmap := make(ColourMap)
	cmap[Version] = color.New(color.FgCyan)
	cmap[Plan] = color.HEX("#999999")
	cmap[TestOK] = color.New(color.FgGreen)
	cmap[TestNOK] = color.New(color.FgRed, color.OpBold)
	cmap[Diagnostic] = color.HEX("#666666")
	cmap[Bail] = color.New(color.FgYellow, color.OpBold)
	cmap[SummaryOK] = color.New(color.FgGreen, color.OpBold)
	cmap[SummaryNOK] = color.New(color.FgRed, color.OpBold)
	cmap[PlanNOK] = color.New(color.FgMagenta, color.Bold)
	return cmap
}

func parseLine(line string) Line {
	if matches := reVersion.FindStringSubmatch(line); matches != nil {
		return Line{Type: Version}
	}
	if matches := rePlan.FindStringSubmatch(line); matches != nil {
		line := Line{Type: Plan}
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
	if matches := reTest.FindStringSubmatch(line); matches != nil {
		line := Line{}
		res := matches[1]
		if testno := matches[2]; testno != "" {
			i, err := strconv.Atoi(testno)
			if err == nil {
				line.TestNum = i
			}
		}
		switch res {
		case "ok":
			line.Type = TestOK
		case "not ok":
			line.Type = TestNOK
		}
		return line
	}
	if matches := reDiagnostic.FindStringSubmatch(line); matches != nil {
		return Line{Type: Diagnostic}
	}
	if matches := reBail.FindStringSubmatch(line); matches != nil {
		return Line{Type: Bail}
	}
	return Line{Type: Unknown}
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

func cprintln(text string, linetype LineType, cmap ColourMap, opts Options) {
	if opts.Failures && linetype == TestOK {
		return
	}
	if opts.Glyphs {
		// Replace `ok/not ok` (or prepend) glyphs
		switch linetype {
		case TestOK:
			text = reTestPrefix.ReplaceAllString(text, GlyphOK+" ")
		case TestNOK:
			text = reTestPrefix.ReplaceAllString(text, GlyphNOK+" ")
		case Bail:
			text = GlyphNOK + " " + text
		}
	}
	cmap[linetype].Println(text)
}

func run(opts Options, ofh io.Writer) int {
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
	cmap := colourMap(opts)

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
		case Plan:
			planLast = line.PlanLast
		case TestOK, TestNOK:
			if line.TestNum > 0 {
				testnum = line.TestNum
			} else {
				testnum++
			}
			if line.Type == TestNOK {
				failures = append(failures, testnum)
				if exitCode < TestFailExitCode {
					exitCode = TestFailExitCode
				}
			}
		case Bail:
			if exitCode < BailExitCode {
				exitCode = BailExitCode
			}
		}
	}

	planNOK := planLast > 0 && testnum != planLast
	if planNOK && exitCode < PlanFailExitCode {
		exitCode = PlanFailExitCode
	}

	glyph := ""
	if opts.Glyphs {
		if planNOK || len(failures) > 0 {
			glyph = GlyphNOK + " "
		} else {
			glyph = GlyphOK + " "
		}
	}

	if opts.Summary {
		if len(failures) > 0 {
			plural := ""
			if len(failures) > 1 {
				plural = "s"
			}
			cmap[SummaryNOK].Printf("%sFAILED test%s: %s\n",
				glyph, plural,
				failureString(failures))
			cmap[SummaryNOK].Printf("%sFailed %d/%d tests, %0.02f%% ok\n",
				glyph, len(failures), testnum,
				float64(testnum-len(failures))*100/float64(testnum))
		} else if !planNOK {
			cmap[SummaryOK].Printf("%sPassed %d/%d tests, 100%% ok\n",
				glyph, testnum, testnum)
		}
	}

	// Fail if we haven't seen all planned tests
	if planNOK {
		cmap[PlanNOK].Printf("%sFailed plan: only %d/%d planned tests seen\n",
			glyph, testnum, planLast)
	}

	return exitCode
}

func main() {
	// Parse default options are HelpFlag | PrintErrors | PassDoubleDash
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		ferr := err.(*flags.Error)
		if ferr.Type == flags.ErrHelp {
			os.Exit(0)
		}

		// Does PrintErrors work? Is it not set?
		fmt.Fprintln(os.Stderr, "")
		parser.WriteHelp(os.Stderr)
		os.Exit(2)
	}

	exitCode := run(opts, os.Stdout)
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
