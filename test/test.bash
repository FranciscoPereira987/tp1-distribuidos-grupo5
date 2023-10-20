#!/usr/bin/env bash

diff -q <(sort client/results/first.csv) <(sort test/expected/first.csv)
diff -q <(sort client/results/second.csv) <(sort test/expected/second.csv)
diff -q <(sort client/results/third.csv) <(sort test/expected/third.csv)
diff -q <(sort client/results/fourth.csv) <(sort test/expected/fourth.csv)
