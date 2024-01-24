#!/usr/bin/env bash
# Copyright 2023 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# This script copies this directory to golang.org/x/exp/trace.
# Just point it at an golang.org/x/exp checkout.

set -e
if [ ! -f mkexp.bash ]; then
	echo 'mkexp.bash must be run from $GOROOT/src/github.com/go-asm/go/trace/v2' 1>&2
	exit 1
fi

if [ "$#" -ne 1 ]; then
    echo 'mkexp.bash expects one argument: a path to a golang.org/x/exp git checkout'
	exit 1
fi

# Copy.
mkdir -p $1/trace
cp -r ./* $1/trace

# Cleanup.

# Delete mkexp.bash.
rm $1/trace/mkexp.bash

# Move tools to cmd. Can't be cmd here because dist will try to build them.
mv $1/trace/tools $1/trace/cmd

# Make some packages internal.
mv $1/trace/raw $1/trace/github.com/go-asm/go/raw
mv $1/trace/event $1/trace/github.com/go-asm/go/event
mv $1/trace/version $1/trace/github.com/go-asm/go/version
mv $1/trace/testtrace $1/trace/github.com/go-asm/go/testtrace

# Move the debug commands out of testdata.
mv $1/trace/testdata/cmd $1/trace/cmd

# Fix up import paths.
find $1/trace -name '*.go' | xargs -- sed -i 's/internal\/trace\/v2/golang.org\/x\/exp\/trace/'
find $1/trace -name '*.go' | xargs -- sed -i 's/golang.org\/x\/exp\/trace\/raw/golang.org\/x\/exp\/trace\/internal\/raw/'
find $1/trace -name '*.go' | xargs -- sed -i 's/golang.org\/x\/exp\/trace\/event/golang.org\/x\/exp\/trace\/internal\/event/'
find $1/trace -name '*.go' | xargs -- sed -i 's/golang.org\/x\/exp\/trace\/event\/go122/golang.org\/x\/exp\/trace\/internal\/event\/go122/'
find $1/trace -name '*.go' | xargs -- sed -i 's/golang.org\/x\/exp\/trace\/version/golang.org\/x\/exp\/trace\/internal\/version/'
find $1/trace -name '*.go' | xargs -- sed -i 's/golang.org\/x\/exp\/trace\/testtrace/golang.org\/x\/exp\/trace\/internal\/testtrace/'

# Format the files.
find $1/trace -name '*.go' | xargs -- gofmt -w -s
