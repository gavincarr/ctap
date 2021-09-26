package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"path/filepath"
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
	}

	for _, tc := range tests {
		opts := Options{}
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

		ec := run(opts, buf)
		got := buf.Bytes()
		assert.Equal(t, tc.exitCode, ec, tc.name+" exitCode")

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
		if !bytes.Equal(got, exp) {
			t.Errorf("test %q failed:\n%s\n", tc.name,
				diff.Diff(string(got), string(exp)))
		}

	}
}
