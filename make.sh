#!/bin/bash

modules=( \
"http/endpoint" \
"http/trigger" \
"serial/endpoint" \
"serial/trigger" \
"socket/endpoint" \
"socket/trigger" \
)

makeModule() {
    for dir in ${modules[@]} ; do
        cd ${dir}
        ./make.sh $*
        cd -
    done
}

makeModule $*

