#!/usr/bin/env bash

mkdir -p test/diff test/expected

dataset=test/expected/$(stat -c'%s' client/data/test.csv)

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
        mv client/results/* "$dataset"
    fi
    exit
fi

test_query() {
    if diff <(sort $dataset/$1.csv) <(sort client/results/$1.csv) >test/diff/$1.diff
        then echo $1 query succeeded
    else
        echo $1 query failed
    fi
}

test_query first
test_query second
test_query third
test_query fourth
