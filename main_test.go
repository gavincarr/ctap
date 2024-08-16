package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/diff"
	"github.com/stretchr/testify/assert"
)

var update *bool

func init() {
	testing.Init()
	update = flag.Bool("u", false, "update .golden files")
	flag.Parse()
}

func TestBasic(t *testing.T) {
	var tests = []struct {
		name     string
		infile   string
		outfile  string
		exitCode int
		flags    string
	}{
		{"test1", "test1.txt", "test1.txt", 0, ""},
		{"test2", "test2.txt", "test2.txt", 3, ""},
		{"test3", "test3.txt", "test3.txt", 4, ""},
		{"test4", "test4.txt", "test4.txt", 4, ""},
		{"test5", "test5.txt", "test5.txt", 5, ""},
		// -s
		{"test1 -s", "test1.txt", "test1s.txt", 0, "s"},
		{"test2 -s", "test2.txt", "test2s.txt", 3, "s"},
		{"test3 -s", "test3.txt", "test3s.txt", 4, "s"},
		{"test4 -s", "test4.txt", "test4s.txt", 4, "s"},
		{"test5 -s", "test5.txt", "test5s.txt", 5, "s"},
		// -f
		{"test1 -f", "test1.txt", "test1f.txt", 0, "f"},
		{"test2 -f", "test2.txt", "test2f.txt", 3, "f"},
		// -g
		{"test1 -g", "test1.txt", "test1g.txt", 0, "g"},
		{"test2 -g", "test2.txt", "test2g.txt", 3, "g"},
		{"test3 -g", "test3.txt", "test3g.txt", 4, "g"},
		{"test4 -g", "test4.txt", "test4g.txt", 4, "g"},
		{"test5 -g", "test5.txt", "test5g.txt", 5, "g"},
		// Combos
		{"test1 -fs", "test1.txt", "test1fs.txt", 0, "fs"},
		{"test2 -fs", "test2.txt", "test2fs.txt", 3, "fs"},
		{"test1 -gs", "test1.txt", "test1gs.txt", 0, "gs"},
		{"test2 -gs", "test2.txt", "test2gs.txt", 3, "gs"},
		{"test3 -gs", "test3.txt", "test3gs.txt", 4, "gs"},
		{"test4 -gs", "test4.txt", "test4gs.txt", 4, "gs"},
		{"test5 -gs", "test5.txt", "test5gs.txt", 5, "gs"},
		{"test1 -fgs", "test1.txt", "test1fgs.txt", 0, "fgs"},
		{"test2 -fgs", "test2.txt", "test2fgs.txt", 3, "fgs"},
		{"test3 -fgs", "test3.txt", "test3fgs.txt", 4, "fgs"},
		{"test4 -fgs", "test4.txt", "test4fgs.txt", 4, "fgs"},
		{"test5 -fgs", "test5.txt", "test5fgs.txt", 5, "fgs"},
		// Version 13
		{"test13", "test13.txt", "test13.txt", 0, ""},
		{"test14", "test14.txt", "test14.txt", 3, ""},
		// Empty
		{"test0", "test0.txt", "test0.txt", 4, ""},
		{"empty", "empty.txt", "empty.txt", 4, ""},
	}

	reNL := regexp.MustCompile("\r?\n")

	for _, tc := range tests {
		opts := options{}
		if strings.Contains(tc.flags, "f") {
			opts.Failures = true
		}
		if strings.Contains(tc.flags, "g") {
			opts.Glyphs = true
		}
		if strings.Contains(tc.flags, "s") {
			opts.Summary = true
		}
		opts.Args.TapFile = filepath.Join("testdata", tc.infile)
		buf := new(bytes.Buffer)

		code, err := runCLI(opts, buf)
		if err != nil {
			t.Error(err)
		}
		got := buf.Bytes()
		assert.Equal(t, tc.exitCode, code, tc.name+" exitCode")

		golden := filepath.Join("testdata", "golden", tc.outfile)
		if *update {
			if err := ioutil.WriteFile(golden, got, 0644); err != nil {
				t.Fatalf("failed to update golden file %q: %s\n", golden, err)
			}
			continue
		}

		exp, err := ioutil.ReadFile(golden)
		if err != nil {
			t.Fatalf("%s: %s", err.Error(), string(exp))
		}
		if runtime.GOOS == "windows" {
			// For Windows tests, normalise line endings
			got = reNL.ReplaceAll(exp, []byte("\n"))
			exp = reNL.ReplaceAll(exp, []byte("\n"))
		}
		if !bytes.Equal(got, exp) {
			t.Errorf("test %q failed:\n%s\n", tc.name,
				diff.Diff(string(exp), string(got)))
		}
	}
}

func TestCustomColours(t *testing.T) {
	var tests = []struct {
		name    string
		infile  string
		outfile string
		opts    options
	}{
		{"test1", "test1.txt", "test1cc1.txt",
			options{COk: "#339933", CFail: "bold c60",
				CPlan: "yellow bold", CDiag: "#939"},
		},
		{"test2", "test2.txt", "test2cc1.txt",
			options{COk: "#339933", CFail: "bold c60",
				CPlan: "yellow bold", CDiag: "#939"},
		},
		{"test5", "test5.txt", "test5cc1.txt",
			options{COk: "#339933", CFail: "bold c60",
				CPlan: "yellow bold", CDiag: "#939",
				CBail: "yellow bold reverse blink"},
		},
	}

	reNL := regexp.MustCompile("\r?\n")

	for _, tc := range tests {
		opts := tc.opts
		opts.Args.TapFile = filepath.Join("testdata", tc.infile)
		buf := new(bytes.Buffer)

		_, err := runCLI(opts, buf)
		if err != nil {
			t.Error(err)
		}
		got := buf.Bytes()

		golden := filepath.Join("testdata", "golden", tc.outfile)
		if *update {
			if err = ioutil.WriteFile(golden, got, 0644); err != nil {
				t.Fatalf("failed to update golden file %q: %s\n", golden, err)
			}
			continue
		}

		exp, err := ioutil.ReadFile(golden)
		if err != nil {
			t.Fatalf("%s: %s", err.Error(), string(exp))
		}
		if runtime.GOOS == "windows" {
			// For Windows tests, normalise line endings
			got = reNL.ReplaceAll(exp, []byte("\n"))
			exp = reNL.ReplaceAll(exp, []byte("\n"))
		}
		if !bytes.Equal(got, exp) {
			t.Errorf("test %q failed:\n%s\n", tc.name,
				diff.Diff(string(exp), string(got)))
		}
	}
}
