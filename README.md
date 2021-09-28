
ctap
====

[![GoReportCard](https://goreportcard.com/badge/github.com/gavincarr/ctap)](https://goreportcard.com/report/github.com/gavincarr/ctap)
[![Build Status](https://github.com/gavincarr/ctap/actions/workflows/test.yml/badge.svg)](https://github.com/gavincarr/ctap/actions?query=workflow%3Atest)
[![GitHub license](https://badgen.net/github/license/gavincarr/ctap)](https://github.com/gavincarr/ctap/blob/master/LICENSE)

ctap is a lightweight, portable [TAP-output](http://testanything.org/)
colouriser, written in Go.

It turns boring old TAP output:

![Boring TAP output](/screenshots/test2.png?raw=true)

into snazzy, easily-scannable output:

![Snazzy, scannable TAP output](/screenshots/test2gs.png?raw=true)

and returns non-zero exit codes on failures.


Installation
------------

Binary packages are available from the
[Releases page](https://github.com/gavincarr/ctap/releases/latest/).

Or if you have `go` installed, you can do:

    go install github.com/gavincarr/ctap@latest

which installs the latest version of ctap in your `$GOPATH/bin`
or `$HOME/go/bin` directory (which you might need to add to your
`$PATH`).


Usage
-----

    Usage:
      ctap [OPTIONS] [TapFile]

    Application Options:
      -f, --failures  show test failures (suppress TAP successes)
      -g, --glyphs    show ✓✗ glyphs instead of 'ok/not ok' in TAP output
      -s, --summary   append a Test::Harness-like summary of the test results

    Help Options:
      -h, --help      Show this help message


Todo
----

- [x] add `-f` option to show failures (suppress successes from TAP output)
- [x] add `-g` option to use glyphs instead of 'ok/not ok' in TAP output
- [ ] add options to specify custom colours (`-POFDB`?)
- [ ] add config file support for standard options/colours
- [ ] add a `--save` option for setting config file standard options/colours
- [ ] add other renderers to transmute TAP output (e.g. 'dots') (?)


See Also
--------

- [Tap Colorize](https://www.npmjs.com/package/tap-colorize) - nice
  javascript library and cli utility

- [Tap Prettify](https://github.com/toolness/tap-prettify) - javascript
  formatter with a focus on making TAP output useful (rather than colourful)

- [Tap::Formatter::Color](https://metacpan.org/pod/TAP::Formatter::Color) -
  the original perl TAP color formatter library

- [Awesome TAP](https://github.com/sindresorhus/awesome-tap) - great general
  list of TAP-related resources (esp. javascript-related)


Author
------

Copyright 2021 Gavin Carr <gavin@openfusion.com.au>.


Licence
--------

ctap is available under the terms of the MIT Licence.

