#!/usr/bin/env bash

mkdir -p test/diff

diff <(sort test/expected/first.csv) <(sort client/results/first.csv) >test/diff/first.diff
diff <(sort test/expected/second.csv) <(sort client/results/second.csv) >test/diff/second.diff
diff <(sort test/expected/third.csv) <(sort client/results/third.csv) >test/diff/third.diff
diff <(sort test/expected/fourth.csv) <(sort client/results/fourth.csv) >test/diff/fourth.diff
