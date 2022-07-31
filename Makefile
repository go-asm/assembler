.DEFAULT_GOAL = all

GO_VERSION ?= go1.19beta1

.PHONY: all
all: sync remove fiximport linkname fmt

.PHONY: sync
sync:
	rm -rf $(shell find . -mindepth 1 -maxdepth 1 -type d -not -iwholename '**.git**' -not -iwholename '**_**' -not -iwholename '**assembler**' | sort)
	ditto ${HOME}/sdk/${GO_VERSION}/src/internal .
	ditto ${HOME}/sdk/${GO_VERSION}/src/cmd/internal ./cmd
	ditto ${HOME}/sdk/${GO_VERSION}/src/cmd/asm/internal/arch ./asm/arch

.PHONY: remove
remove:
	rm -rf abi
	rm -f fuzz/trace.go
	rm -f syscall/unix/getentropy_darwin.go
	rm -f bytealg/compare_amd64.s bytealg/compare_arm64.s bytealg/equal_amd64.s bytealg/equal_arm64.s

define fix_linkname
sed -i -E ':a;N;$$!ba;s|${1}|${2}|' ${3}
endef

.PHONY: linkname
linkname:
	$(call fix_linkname,(//go:noescape)\n(func Compare),\1\n//go:linkname Compare internal/bytealg.Compare\n\2,bytealg/compare_native.go)
	$(call fix_linkname,import "sync/atomic",import (\n	"sync/atomic"\n	_ "unsafe" // for go:linkname\n),poll/fd_mutex.go)
	$(call fix_linkname,\n(func runtime_Semacquire),\n\n//go:linkname runtime_Semacquire runtime.semacquire\n\1,poll/fd_mutex.go)
	$(call fix_linkname,(func runtime_Semrelease),//go:linkname runtime_Semrelease runtime.semrelease\n\1,poll/fd_mutex.go)
	$(call fix_linkname,\n(func runtime_pollServerInit),//go:linkname runtime_pollServerInit runtime.pollServerInit\n\1,poll/fd_poll_runtime.go)
	$(call fix_linkname,\n(func runtime_pollOpen),\n//go:linkname runtime_pollOpen runtime.pollOpen\n\1,poll/fd_poll_runtime.go)
	$(call fix_linkname,\n(func runtime_pollClose),\n//go:linkname runtime_pollClose runtime.pollClose\n\1,poll/fd_poll_runtime.go)
	$(call fix_linkname,\n(func runtime_pollWait),\n//go:linkname runtime_pollWait runtime.pollWait\n\1,poll/fd_poll_runtime.go)
	$(call fix_linkname,\n(func runtime_pollWaitCanceled),\n//go:linkname runtime_pollWaitCanceled runtime.pollWaitCanceled\n\1,poll/fd_poll_runtime.go)
	$(call fix_linkname,\n(func runtime_pollReset),\n//go:linkname runtime_pollReset runtime.pollReset\n\1,poll/fd_poll_runtime.go)
	$(call fix_linkname,\n(func runtime_pollSetDeadline),\n//go:linkname runtime_pollSetDeadline runtime.pollSetDeadline\n\1,poll/fd_poll_runtime.go)
	$(call fix_linkname,\n(func runtime_pollUnblock),\n//go:linkname runtime_pollUnblock runtime.pollUnblock\n\1,poll/fd_poll_runtime.go)
	$(call fix_linkname,\n(func runtime_isPollServerDescriptor),\n//go:linkname runtime_isPollServerDescriptor runtime.isPollServerDescriptor\n\1,poll/fd_poll_runtime.go)

.PHONY: fiximport
fiximport:
	grep -rl 'cmd/internal/' ${CURDIR}/** | grep -v Makefile | xargs sed -i 's|cmd/internal/|github.com/go-asm/go/cmd/|g'
	grep -rl 'internal/' ${CURDIR}/** | grep -v Makefile | xargs sed -i 's|internal/|github.com/go-asm/go/|g'
	sed -i 's|../../github.com/go-asm/go/cmd/reflectdata/reflect.go|src/cmd/reflectdata/reflect.go|g' reflectlite/type.go

.PHONY: fmt
fmt:
	gofmt -w -s .
	goimports -w -local=github.com/go-asm/go .
