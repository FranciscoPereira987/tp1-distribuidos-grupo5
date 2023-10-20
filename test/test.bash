#!/usr/bin/env bash

diff <(sort client/results/first.csv) <(sort test/expected/first.csv)
diff <(sort client/results/second.csv) <(sort test/expected/second.csv)
diff <(sort client/results/third.csv) <(sort test/expected/third.csv)
diff <(sort client/results/fourth.csv) <(sort test/expected/fourth.csv)
