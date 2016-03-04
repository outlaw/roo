#!/usr/bin/make -f

SHELL=/bin/bash
SRC=src/roo/roo.go
APP=roo

build: deps compile

compile: 
	GOOS=linux GOARCH=amd64 gb build && \
	GOOS=darwin GOARCH=amd64 gb build

release: compile
	mkdir -p ./tmp && \
	cp bin/* tmp && \
	mv ./tmp/${APP}-darwin-amd64 ./tmp/${APP} && \
	tar -czv -C tmp -f release/${APP}-darwin-x86_64.tar.gz ${APP} && \
	rm -rf tmp/${APP} && \
	mv ./tmp/${APP}-linux-amd64 ./tmp/${APP} && \
	tar -czv -C tmp -f release/${APP}-linux-x86_64.tar.gz ${APP} && \
	rm -rf ./tmp

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
