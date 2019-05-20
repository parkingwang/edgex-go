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
"lua/endpoint" \
)

makeModule() {
    for dir in ${modules[@]} ; do
        cd ${dir}
        ./make.sh $*
        cd -
    done
}

makeModule $*

