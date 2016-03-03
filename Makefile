#!/usr/bin/make -f

SHELL=/bin/bash
SRC=src/roo/roo.go
APP=roo

build: deps compile

compile: 
	GOOS=linux GOARCH=amd64 gb build && \
	GOOS=darwin GOARCH=amd64 gb build
release: compile
	tar czvf release/${APP}-darwin-x86_64.tar.gz bin/${APP}-darwin-amd64 && \
	tar czvf release/${APP}-linux-x86_64.tar.gz bin/${APP}-linux-amd64

publish:
	aws s3 cp release/roo-linux-x86_64.tar.gz s3://hooroo-builds/roo/latest/roo-linux-x86_64.tar.gz && \
	aws s3 cp release/roo-darwin-x86_64.tar.gz s3://hooroo-builds/roo/latest/roo-darwin-x86_64.tar.gz

#compile: deps golang-crosscompile
#	source golang-crosscompile/crosscompile.bash; \
#	go-darwin-amd64 build -o release/${APP}-Darwin-x86_64 ${SRC}; \
#	go-linux-amd64 build -o release/${APP}-Linux-x86_64 ${SRC};

golang-crosscompile:
	git clone https://github.com/davecheney/golang-crosscompile.git

deps:
