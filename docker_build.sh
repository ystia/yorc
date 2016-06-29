#!/usr/bin/env bash

script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

if [[ ! -e ${script_dir}/janus ]]; then
    cd ${script_dir}
    make
fi
cp ${script_dir}/janus ${script_dir}/pkg/
cd ${script_dir}/pkg
docker build -t starlings/janus .
