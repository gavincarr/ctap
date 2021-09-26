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
)

type options struct {
	Failures bool `short:"f" long:"failures" description:"show test failures (suppress TAP successes)"`
	Glyphs   bool `short:"g" long:"glyphs" description:"show \u2713\u2717 glyphs instead of 'ok/not ok' in TAP output"`
	Summary  bool `short:"s" long:"summary" description:"append a Test::Harness-like summary of the test results"`
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

type colourMap map[lineType]color.PrinterFace

func getColourMap(opt options) colourMap {
	cmap := make(colourMap)
	cmap[tapVersion] = color.New(color.FgCyan)
	cmap[tapPlan] = color.HEX("#999999")
	cmap[tapTestOK] = color.New(color.FgGreen)
	cmap[tapTestNOK] = color.New(color.FgRed, color.OpBold)
	cmap[tapDiagnostic] = color.HEX("#666666")
	cmap[tapBail] = color.New(color.FgYellow, color.OpBold)
	cmap[tapSummaryOK] = color.New(color.FgGreen, color.OpBold)
	cmap[tapSummaryNOK] = color.New(color.FgRed, color.OpBold)
	cmap[tapPlanNOK] = color.New(color.FgMagenta, color.Bold)
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
	cmap[linetype].Println(text)
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
	planNOK := planLast > 0 && testnum != planLast
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
		cmap[tapPlanNOK].Printf("%sFailed plan: only %d/%d planned tests seen\n",
			glyph, testnum, planLast)
	}

	return exitCode
}

func run(opts options, ofh io.Writer) int {
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
