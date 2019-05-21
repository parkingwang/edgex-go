#!/bin/bash

modules=( \
"http/endpoint" \
"http/trigger" \
"serial/endpoint" \
"serial/trigger" \
"socket/endpoint" \
"socket/trigger" \
"dongkong/endpoint" \
"dongkong/trigger" \
"lua" \
"echo" \
)

makeModule() {
    for dir in ${modules[@]} ; do
        cd ${dir}
        OSARCH=arm ./make.sh $*
        OSARCH=amd64 ./make.sh $*
        cd -
    done
}

makeModule $*

