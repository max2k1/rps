# RPS

[![Godoc Reference](https://godoc.org/github.com/max2k1/rps?status.svg)](http://godoc.org/github.com/max2k1/rps)
[![Build Status](https://travis-ci.org/max2k1/rps.svg?branch=master)](https://travis-ci.org/max2k1/rps)
[![Coverage Status](https://coveralls.io/repos/github/max2k1/rps/badge.svg?branch=master)](https://coveralls.io/github/max2k1/rps?branch=master)
[![Go Report Card](https://goreportcard.com/badge/max2k1/rps)](https://goreportcard.com/report/max2k1/rps)

Utility to measure RPS (rows, or requests per second) and BPS for incoming from pipe flow.
Primary usage: measure server's load, doing "tail -f" from access-logs.

It reads from STDIN line-by-line (separated by '\n') and every second prints current RPS and BPS.
When STDIN is closed, or program terminated by Ctrl-C, it prints summary information about input stream characteristics.

Command-line arguments:
```
$ rps -h
Usage of ./rps:
  -maxrows int
        Exit after N rows were processed
  -noformat
        Do not format values
  -nosummary
        Don't show summary at the end
  -notime
        Don't show timestamp on everysecond stats
  -oneline
        Print everysecond stats without newlines
  -passthrough
        Passthrough incoming data to stdout
  -timeout int
        Exit after N seconds
  -version
        Show version and exit
```

Usage:
```
$ tail -f /var/log/nginx/access.log | rps
2016-10-03 07:55:41 1,685 RPS / 2,783,074 bytes
2016-10-03 07:55:42 2,114 RPS / 3,618,198 bytes
...
2016-10-03 08:21:56 3,290 RPS / 5,440,330 bytes
2016-10-03 08:21:57 2,361 RPS / 3,989,635 bytes
^C= Summary: ========================
Start:            2016-10-03 07:55:40
Stop:             2016-10-03 08:21:58
Elapsed, sec:               1,577.990
Size, bytes:           10,622,625,816
Speed, bps:                 6,731,745
Rows:                       6,786,319
RPS  Min:                         913
RPS  Avg:                       4,303
RPS 50th:                       4,036
RPS 80th:                       5,721
RPS 95th:                       7,577
RPS 99th:                       9,282
RPS  Max:                      13,052
```
