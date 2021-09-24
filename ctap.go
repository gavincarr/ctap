/*
ctap is a portable lightweight colouriser for TAP
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
)

type Options struct {
	//Verbose bool `short:"v" long:"verbose" description:"display verbose debug output"`
	Summary bool `short:"s" long:"summary" description:"append a summary of the test results after TAP output"`
	Args    struct {
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
)

type LineType int

const (
	Unknown LineType = iota
	Version
	Plan
	Test
	Diagnostic
	Bail
)

func (t LineType) String() string {
	return [...]string{"Unkn", "Vers", "Plan", "Test", "Diag", "Bail"}[t]
}

type StatusType int

const (
	Unset StatusType = iota
	OK
	NOK
)

type Line struct {
	Type        LineType
	PlanFirst   int        // Plan
	PlanLast    int        // Plan
	Status      StatusType // Test
	TestNum     int        // Test
	Description string
	Directive   string
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
		line := Line{Type: Test}
		res := matches[1]
		if testno := matches[2]; testno != "" {
			i, err := strconv.Atoi(testno)
			if err == nil {
				line.TestNum = i
			}
		}
		switch res {
		case "ok":
			line.Status = OK
		case "not ok":
			line.Status = NOK
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
	//red := color.New(color.FgRed)
	redb := color.New(color.FgRed, color.OpBold)
	green := color.New(color.FgGreen)
	greenb := color.New(color.FgGreen, color.OpBold)
	yellow := color.New(color.FgYellow, color.OpBold)
	//cyan := color.New(color.FgCyan, color.OpBold)
	//def := color.New(color.FgDefault)
	gray1 := color.HEX("#999999")
	gray2 := color.HEX("#666666")
	magentab := color.New(color.FgMagenta, color.Bold)

	var planLast int
	testnum := 0
	failures := []int{}
	exitCode := 0
	for scanner.Scan() {
		text := scanner.Text()

		line := parseLine(text)

		switch line.Type {
		case Plan:
			planLast = line.PlanLast
			gray1.Println(text)
		case Test:
			if line.TestNum > 0 {
				testnum = line.TestNum
			} else {
				testnum++
			}
			switch line.Status {
			case OK:
				green.Println(text)
			case NOK:
				failures = append(failures, testnum)
				redb.Println(text)
			default:
				log.Fatal("unhandled status: " + text)
			}
		case Diagnostic:
			gray2.Println(text)
		case Bail:
			yellow.Println(text)
			exitCode = BailExitCode
		default:
			fmt.Printf("[%s] %s\n", line.Type.String(), text)
		}
	}

	badPlan := planLast > 0 && testnum != planLast
	if badPlan && exitCode == 0 {
		exitCode = PlanFailExitCode
	}

	if opts.Summary {
		if len(failures) > 0 {
			plural := ""
			if len(failures) > 1 {
				plural = "s"
			}
			redb.Printf("FAILED test%s: %s\n",
				plural,
				failureString(failures))
			redb.Printf("Failed %d/%d tests, %0.02f%% ok \u2717\n",
				len(failures), testnum,
				float64(testnum-len(failures))*100/float64(testnum))
			if exitCode == 0 {
				exitCode = TestFailExitCode
			}
		} else if !badPlan {
			greenb.Printf("Passed %d/%d tests, 100%% ok \u2713\n", testnum, testnum)
		}
	}

	// Fail if we haven't seen all planned tests
	if badPlan {
		magentab.Printf("Failed plan: only %d/%d planned tests seen\n",
			testnum, planLast)
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
