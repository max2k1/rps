# RPS
Utility to measure RPS (rows, or requests per second) and BPS for incoming from pipe flow.
Primary usage: measure server's load, doing "tail -f" from access-logs.

It reads from STDIN line-by-line (separated by '\n') and every second prints current RPS and BPS.
When STDIN is closed, or program terminated by Ctrl-C, it prints summary information about input stream characteristics.

Command-line arguments:
```
$ rps -h
Usage of rps:
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
```

Usage:
```
$ tail -f /var/log/nginx/access.log | rps
2016-10-01 19:12:58 205 RPS / 13,530 bytes
2016-10-01 19:12:59 208 RPS / 13,728 bytes
2016-10-01 19:13:00 208 RPS / 13,728 bytes
2016-10-01 19:13:01 208 RPS / 13,728 bytes
2016-10-01 19:13:02 207 RPS / 13,662 bytes
2016-10-01 19:13:03 205 RPS / 13,530 bytes
2016-10-01 19:13:04 208 RPS / 13,728 bytes
2016-10-01 19:13:05 207 RPS / 13,662 bytes
2016-10-01 19:13:06 199 RPS / 13,134 bytes
^C= Summary: ========================
Start:		2016-10-01 19:12:57
Stop:		2016-10-01 19:13:06
Elapsed, sec:	          9.043
Bytes:		            122,958
Speed, bps:	             13,596
Lines:		              1,864
MIN  RPS:	                199
AVG  RPS:	                207
50th RPS:	                207
80th RPS:	                208
95th RPS:	                208
99th RPS:	                208
MAX  RPS:	                208
```
