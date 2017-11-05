# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: geta android ios geta-cross swarm evm all test clean
.PHONY: geta-linux geta-linux-386 geta-linux-amd64 geta-linux-mips64 geta-linux-mips64le
.PHONY: geta-linux-arm geta-linux-arm-5 geta-linux-arm-6 geta-linux-arm-7 geta-linux-arm64
.PHONY: geta-darwin geta-darwin-386 geta-darwin-amd64
.PHONY: geta-windows geta-windows-386 geta-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

geta:
	build/env.sh go run build/ci.go install ./cmd/geta
	@echo "Done building."
	@echo "Run \"$(GOBIN)/geta\" to launch geta."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/geta.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/geta.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/jteeuwen/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go install ./cmd/abigen

# Cross Compilation Targets (xgo)

geta-cross: geta-linux geta-darwin geta-windows geta-android geta-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/geta-*

geta-linux: geta-linux-386 geta-linux-amd64 geta-linux-arm geta-linux-mips64 geta-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/geta-linux-*

geta-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/geta
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/geta-linux-* | grep 386

geta-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/geta
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/geta-linux-* | grep amd64

geta-linux-arm: geta-linux-arm-5 geta-linux-arm-6 geta-linux-arm-7 geta-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/geta-linux-* | grep arm

geta-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/geta
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/geta-linux-* | grep arm-5

geta-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/geta
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/geta-linux-* | grep arm-6

geta-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/geta
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/geta-linux-* | grep arm-7

geta-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/geta
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/geta-linux-* | grep arm64

geta-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/geta
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/geta-linux-* | grep mips

geta-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/geta
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/geta-linux-* | grep mipsle

geta-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/geta
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/geta-linux-* | grep mips64

geta-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/geta
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/geta-linux-* | grep mips64le

geta-darwin: geta-darwin-386 geta-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/geta-darwin-*

geta-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/geta
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/geta-darwin-* | grep 386

geta-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/geta
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/geta-darwin-* | grep amd64

geta-windows: geta-windows-386 geta-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/geta-windows-*

geta-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/geta
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/geta-windows-* | grep 386

geta-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/geta
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/geta-windows-* | grep amd64
