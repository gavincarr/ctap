
ctap
====

ctap is a lightweight, portable [TAP-output](http://testanything.org/)
colouriser, written in Go.

It turns boring old TAP output:

![Boring TAP output](/screenshots/test2.png?raw=true)

into snazzy, easily-scannable output:

![Snazzy, scannable TAP output](/screenshots/test2c.png?raw=true)

and returns non-zero exit codes on failures.


Installation
------------

Binary packages are available from the
(Releases page)[/gavincarr/ctap/releases/latest/].

Or if you have go installed, you can do:

    go install github.com/gavincarr/ctap@latest

which installs the latest version of ctap in your `$GOPATH/bin`
or `$HOME/go/bin` directory (which you might need to add to your
`$PATH`).


Author and Licence
------------------

Copyright 2021 Gavin Carr <gavin@openfusion.com.au>.

ctap is available under the terms of the MIT Licence.

