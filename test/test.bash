#!/usr/bin/env bash

client_dir=${1-clients/c1}
mkdir -p test/diff test/expected

dataset=test/expected/$(stat -c'%s' "$client_dir"/data/test.csv)

yesno () {
  printf '%s ' "$*" '[Y/n]'
  read -r ans
  case "$ans" in
    y|Y|'')
      ;;
    *)
      return 1
      ;;
  esac
}

if ! [ -d "$dataset" ]
then
    if yesno 'New dataset. Store test data to compare against later runs?'
    then
        mkdir "$dataset"
        for f in "$client_dir"/results/*
        do
            sort "$f" > "$dataset/${f##*/}"
        done
    fi
    exit
fi

test_query() {
    if diff < "$dataset/$1.csv" <(sort "$client_dir"/results/$1.csv) >test/diff/$1.diff
        then printf '\x1b[32;1m%s:\x1b[m %s\n' PASSED "$1 query"
    else
        printf '\x1b[31;1m%s:\x1b[m %s\n' FAILED "$1 query"
    fi
}

test_query first
test_query second
test_query third
test_query fourth
