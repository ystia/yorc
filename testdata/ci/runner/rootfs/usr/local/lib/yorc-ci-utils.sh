#!/usr/bin/env bash

function getURLFromPart () {
    file=$(curl -s "$1/" |  grep -o -P "$2" | tail -1 ) || return
    if [[ -z "$file" ]] ; then
        return
    fi
    echo "$1/$file"
}

