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
		// Summary versions
		{"test1 summary", "test1.txt", "test1s.txt", 0, "S"},
		{"test2 summary", "test2.txt", "test2s.txt", 3, "S"},
		{"test3 summary", "test3.txt", "test3s.txt", 4, "S"},
		{"test4 summary", "test4.txt", "test4s.txt", 4, "S"},
		{"test5 summary", "test5.txt", "test5s.txt", 5, "S"},
	}

	opts := Options{}
	for _, tc := range tests {
		if strings.Contains(tc.flags, "S") {
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
