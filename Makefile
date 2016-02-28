#!/usr/bin/make -f

SHELL=/bin/bash
SRC=src/roo/roo.go
APP=roo

build: deps
	go build ${SRC}

compile: 
	GOOS=linux GOARCH=amd64 gb build; \
	GOOS=darwin GOARCH=amd64 gb build;
release: compile
	tar czvf release/${APP}-darwin-x86_64.tar.gz bin/${APP}-darwin-amd64; \
	tar czvf release/${APP}-linux-x86_64.tar.gz bin/${APP}-linux-amd64;

#compile: deps golang-crosscompile
#	source golang-crosscompile/crosscompile.bash; \
#	go-darwin-amd64 build -o release/${APP}-Darwin-x86_64 ${SRC}; \
#	go-linux-amd64 build -o release/${APP}-Linux-x86_64 ${SRC};

golang-crosscompile:
	git clone https://github.com/davecheney/golang-crosscompile.git

deps:
