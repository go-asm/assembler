package buildcfg

import (
	"runtime"
)

const defaultGO386 = `sse2`
const defaultGOARM = `5`
const defaultGOMIPS = `hardfloat`
const defaultGOMIPS64 = `hardfloat`
const defaultGOPPC64 = `power8`
const defaultGOEXPERIMENT = ``
const defaultGO_EXTLINK_ENABLED = ``
const defaultGO_LDSO = ``
const defaultGOOS = runtime.GOOS
const defaultGOARCH = runtime.GOARCH

var version = runtime.Version()
