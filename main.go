package main

import (
	"bufio"
	"flag"
	"fmt"
	render_number "github.com/max2k1/render_number"
	"io"
	"math"
	"os"
	"os/signal"
	"sort"
	"sync/atomic"
	"syscall"
	"time"
)

var flagNoFormat bool
var flagOneLine bool
var flagPassthrough bool
var flagDontShowSummary bool
var flagDontShowTime bool

func ReadRow(b *bufio.Reader) (line []byte, err error) {
	// Use ReadSlice to look for array,
	// accumulating full buffers.
	var frag []byte
	var full [][]byte

R:
	for {
		var e error
		frag, e = b.ReadSlice('\n')

		switch {
		case e == nil:
			//got final fragment
			break R
		case e == bufio.ErrBufferFull:
			// Make a copy of the buffer.
			buf := make([]byte, len(frag))
			copy(buf, frag)
			full = append(full, buf)
		default:
			err = e
			break R
		}
	}

	// Allocate new buffer to hold the full pieces and the fragment.
	n := 0
	for i := range full {
		n += len(full[i])
	}
	n += len(frag)

	// Copy full pieces and fragment in.
	buf := make([]byte, n)
	n = 0
	for i := range full {
		n += copy(buf[n:], full[i])
	}
	copy(buf[n:], frag)
	return buf, err
}

func init() {
	flag.BoolVar(&flagNoFormat, "noformat", false, "Do not format values")
	flag.BoolVar(&flagOneLine, "oneline", false, "Print everysecond stats without newlines")
	flag.BoolVar(&flagPassthrough, "passthrough", false, "Passthrough incoming data to stdout")
	flag.BoolVar(&flagDontShowSummary, "nosummary", false, "Don't show summary at the end")
	flag.BoolVar(&flagDontShowTime, "notime", false, "Don't show timestamp on everysecond stats")
	flag.Parse()
}

func main() {
	var aRPS []float64 = make([]float64, 0, 600)
	var bRow []byte
	var err error = nil
	var nBytes int64 = 0
	var nBytesLastSec int64 = 0
	var nRows int64 = 0
	var nRowsLastSec int64 = 0

	var nMinRPS int64 = -1
	var n50RPS int64 = 0
	var n80RPS int64 = 0
	var n95RPS int64 = 0
	var n99RPS int64 = 0
	var nMaxRPS int64 = 0

	// RPS ticker (rows per second)
	ticker := time.NewTicker(time.Second)
	go func() {
		var prefix string = ""
		var val1 string = ""
		var val2 string = ""
		var res string = ""

		for t := range ticker.C {
			n := atomic.SwapInt64(&nRowsLastSec, 0)
			b := atomic.SwapInt64(&nBytesLastSec, 0)

			aRPS = append(aRPS, float64(n))
			if n < nMinRPS || nMinRPS < 0 {
				nMinRPS = n
			}
			if n > nMaxRPS {
				nMaxRPS = n
			}

			prefix = ""
			if !flagDontShowTime {
				prefix = fmt.Sprintf("%s ", t.Format("2006-01-02 15:04:05"))
			}

			if flagNoFormat {
				res = fmt.Sprintf("%s%d %d", prefix, n, b)
			} else {
				val1 = fmt.Sprintf("%s RPS", render_number.RenderInteger("#,###.", int64(n)))
				val2 = fmt.Sprintf("%s bytes", render_number.RenderInteger("#,###.", int64(b)))
				res = fmt.Sprintf("%s%s / %s", prefix, val1, val2)
			}

			if flagOneLine {
				res = fmt.Sprintf("\r                                                                                \r%s", res)
			} else {
				res = fmt.Sprintf("%s\n", res)
			}

			fmt.Fprint(os.Stderr, res)
		}
	}()

	// Read data through pipe line-by-line (\n-separated flow)
	cInt := make(chan os.Signal, 2)
	signal.Notify(cInt, os.Interrupt, syscall.SIGTERM)
	b := bufio.NewReader(os.Stdin)
	tStart := time.Now()
	for err == nil {
		select {
		case <-cInt:
			fmt.Fprintln(os.Stderr, "")
			break
		default:
			bRow, err = ReadRow(b)
			if err != nil && err != io.EOF {
				panic(err)
				break
			}

			size := len(bRow)
			if size == 0 {
				continue
			}

			nBytes += int64(size)
			atomic.AddInt64(&nBytesLastSec, int64(size))

			nRows += 1
			atomic.AddInt64(&nRowsLastSec, 1)

			if flagPassthrough {
				os.Stdout.Write(bRow)
			}
		}
	}
	tStop := time.Now()
	ticker.Stop()

	tElapsed := tStop.Sub(tStart)
	if len(aRPS) > 0 {
		if flagOneLine {
			fmt.Fprintln(os.Stderr, "")
		}
		sort.Float64s(aRPS)
		n50RPS = int64(aRPS[int64(math.Floor(float64(len(aRPS))*0.50))])
		n80RPS = int64(aRPS[int64(math.Floor(float64(len(aRPS))*0.80))])
		n95RPS = int64(aRPS[int64(math.Floor(float64(len(aRPS))*0.95))])
		n99RPS = int64(aRPS[int64(math.Floor(float64(len(aRPS))*0.99))])
	}

	if !flagDontShowSummary {
		fmt.Fprintln(os.Stderr, "= Summary: ========================")
		fmt.Fprintf(os.Stderr, "Start:\t\t%s\n", tStart.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(os.Stderr, "Stop:\t\t%s\n", tStop.Format("2006-01-02 15:04:05"))
		if flagNoFormat {
			fmt.Fprintf(os.Stderr, "Elapsed, sec:\t%0.3f\n", float64(tElapsed.Seconds()))
			fmt.Fprintf(os.Stderr, "Size, bytes:\t%d\n", nBytes)
			fmt.Fprintf(os.Stderr, "Speed, bps:\t%0.0f\n", float64(nBytes)/tElapsed.Seconds())
			fmt.Fprintf(os.Stderr, "Rows:\t\t%d\n", nRows)

			if len(aRPS) > 0 {
				fmt.Fprintf(os.Stderr, "RPS  Min:\t%d\n", nMinRPS)
				fmt.Fprintf(os.Stderr, "RPS  Avg:\t%0.0f\n", float64(nRows)/float64(len(aRPS)))
				fmt.Fprintf(os.Stderr, "RPS 50th:\t%d\n", n50RPS)
				fmt.Fprintf(os.Stderr, "RPS 80th:\t%d\n", n80RPS)
				fmt.Fprintf(os.Stderr, "RPS 95th:\t%d\n", n95RPS)
				fmt.Fprintf(os.Stderr, "RPS 99th:\t%d\n", n99RPS)
				fmt.Fprintf(os.Stderr, "RPS  Max:\t%d\n", nMaxRPS)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Elapsed, sec:\t%19s\n", render_number.RenderFloat("#,###.###", float64(tElapsed.Seconds())))
			fmt.Fprintf(os.Stderr, "Size, bytes:\t%19s\n", render_number.RenderInteger("#,###.", int64(nBytes)))
			fmt.Fprintf(os.Stderr, "Speed, bps:\t%19s\n", render_number.RenderInteger("#,###.", int64(float64(nBytes)/tElapsed.Seconds())))
			fmt.Fprintf(os.Stderr, "Rows:\t\t%19s\n", render_number.RenderInteger("#,###.", int64(nRows)))

			if len(aRPS) > 0 {
				fmt.Fprintf(os.Stderr, "RPS  Min:\t%19s\n", render_number.RenderInteger("#,###.", nMinRPS))
				fmt.Fprintf(os.Stderr, "RPS  Avg:\t%19s\n", render_number.RenderFloat("#,###.", float64(nRows)/float64(len(aRPS))))
				fmt.Fprintf(os.Stderr, "RPS 50th:\t%19s\n", render_number.RenderInteger("#,###.", n50RPS))
				fmt.Fprintf(os.Stderr, "RPS 80th:\t%19s\n", render_number.RenderInteger("#,###.", n80RPS))
				fmt.Fprintf(os.Stderr, "RPS 95th:\t%19s\n", render_number.RenderInteger("#,###.", n95RPS))
				fmt.Fprintf(os.Stderr, "RPS 99th:\t%19s\n", render_number.RenderInteger("#,###.", n99RPS))
				fmt.Fprintf(os.Stderr, "RPS  Max:\t%19s\n", render_number.RenderInteger("#,###.", nMaxRPS))
			}
		}

	}
}
