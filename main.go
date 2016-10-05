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
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var flagDontShowSummary bool
var flagDontShowTime bool
var flagMaxRows int64
var flagNoFormat bool
var flagOneLine bool
var flagPassthrough bool
var flagShowVersion bool
var flagTimeout int64

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
	flag.BoolVar(&flagDontShowSummary, "nosummary", false, "Don't show summary at the end")
	flag.BoolVar(&flagDontShowTime, "notime", false, "Don't show timestamp on everysecond stats")
	flag.BoolVar(&flagNoFormat, "noformat", false, "Do not format values")
	flag.BoolVar(&flagOneLine, "oneline", false, "Print everysecond stats without newlines")
	flag.BoolVar(&flagPassthrough, "passthrough", false, "Passthrough incoming data to stdout")
	flag.BoolVar(&flagShowVersion, "version", false, "Show version and exit")
	flag.Int64Var(&flagMaxRows, "maxrows", 0, "Exit after N rows were processed")
	flag.Int64Var(&flagTimeout, "timeout", 0, "Exit after N seconds")
	flag.Parse()
}

func main() {
	var aObservations []float64 = make([]float64, 0, 600)
	var nBytes int64 = 0
	var nBytesLastSec int64 = 0
	var nObservations int64 = 0
	var nRows int64 = 0
	var nRowsLastSec int64 = 0

	var nMinRPS int64 = -1
	var n50RPS int64 = 0
	var n80RPS int64 = 0
	var n95RPS int64 = 0
	var n99RPS int64 = 0
	var nMaxRPS int64 = 0

	if flagShowVersion {
		fmt.Fprintf(os.Stderr, "RPS: Version %s\n", "0.0-8")
		return
	}

	var mutex = &sync.Mutex{}

	cInt := make(chan os.Signal, 2)
	signal.Notify(cInt, os.Interrupt, syscall.SIGTERM)
	cDone := make(chan error)
	cQuit := make(chan bool)

	var cTimeout <-chan time.Time
	if flagTimeout > 0 {
		cTimeout = time.NewTimer(time.Duration(flagTimeout) * time.Second).C
	}

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

			mutex.Lock()
			aObservations = append(aObservations, float64(n))
			if 0 < nMinRPS || nMinRPS > n {
				nMinRPS = n
			}
			if n > nMaxRPS {
				nMaxRPS = n
			}
			mutex.Unlock()

			prefix = ""
			if !flagDontShowTime {
				prefix = fmt.Sprintf("%s ", t.Format("2006-01-02 15:04:05"))
				if !flagNoFormat {
					prefix = fmt.Sprintf("%s| ", prefix)
				}
			}

			if flagNoFormat {
				res = fmt.Sprintf("%s%d %d", prefix, n, b)
			} else {
				val1 = fmt.Sprintf("%s RPS", render_number.RenderInteger("#,###.", int64(n)))
				val2 = fmt.Sprintf("%s bytes", render_number.RenderInteger("#,###.", int64(b)))
				res = fmt.Sprintf("%s%s | %s", prefix, val1, val2)
			}

			if flagOneLine {
				res = fmt.Sprintf("\r%20s\r%s", "", res)
			} else {
				res = fmt.Sprintf("%s\n", res)
			}

			fmt.Fprint(os.Stderr, res)
		}
	}()

	tStart := time.Now()
	b := bufio.NewReader(os.Stdin)
	go func() {
		var bRow []byte
		var err error = nil

		// Read data through pipe line-by-line (\n-separated flow)
	RL:
		for err == nil {
			select {
			case <-cQuit:
				break RL
			default:
				bRow, err = ReadRow(b)

				if err != nil && err != io.EOF {
					break RL
				}

				size := len(bRow)
				if size == 0 {
					continue RL
				}

				atomic.AddInt64(&nBytes, int64(size))
				atomic.AddInt64(&nBytesLastSec, int64(size))

				atomic.AddInt64(&nRows, 1)
				atomic.AddInt64(&nRowsLastSec, 1)

				if flagPassthrough {
					os.Stdout.Write(bRow)
				}

				if 0 < flagMaxRows && flagMaxRows <= atomic.LoadInt64(&nRows) {
					// Rows limit reached
					err = fmt.Errorf("Rows limit reached")
					break RL
				}

			}
		}
		cDone <- err
	}()

MainLoop:
	for {
		select {
		// EOF or rows limit reached
		case err := <-cDone:
			if err != nil && err != io.EOF {
				fmt.Fprintln(os.Stderr, "ERR:", err)
			}
			break MainLoop
		// Ctrl+C pressed
		case <-cInt:
			fmt.Fprintln(os.Stderr, "ERR: Interrupted")
			break MainLoop
		// Timeout expired
		case <-cTimeout:
			fmt.Fprintln(os.Stderr, "ERR: Timeout expired")
			break MainLoop
		}
	}
	select {
	case cQuit <- true:
	default:
	}
	ticker.Stop()
	tStop := time.Now()

	tElapsed := tStop.Sub(tStart)
	mutex.Lock()
	nObservations = int64(len(aObservations))
	if nObservations > 0 {
		if flagOneLine {
			fmt.Fprintln(os.Stderr, "")
		}
		sort.Float64s(aObservations)
		n50RPS = int64(aObservations[int64(math.Floor(float64(nObservations)*0.50))])
		n80RPS = int64(aObservations[int64(math.Floor(float64(nObservations)*0.80))])
		n95RPS = int64(aObservations[int64(math.Floor(float64(nObservations)*0.95))])
		n99RPS = int64(aObservations[int64(math.Floor(float64(nObservations)*0.99))])
		if nMinRPS < 0 {
			nMinRPS = 0
		}
	}
	mutex.Unlock()

	if !flagDontShowSummary {
		fmt.Fprintln(os.Stderr, "= Summary: ========================")
		fmt.Fprintf(os.Stderr, "%-16s%s\n", "Start:", tStart.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(os.Stderr, "%-16s%s\n", "Stop:", tStop.Format("2006-01-02 15:04:05"))
		if flagNoFormat {
			fmt.Fprintf(os.Stderr, "Elapsed, sec:\t%0.3f\n", float64(tElapsed.Seconds()))
			fmt.Fprintf(os.Stderr, "Size, bytes:\t%d\n", atomic.LoadInt64(&nBytes))
			fmt.Fprintf(os.Stderr, "Speed, bps:\t%0.0f\n", float64(atomic.LoadInt64(&nBytes))/tElapsed.Seconds())
			fmt.Fprintf(os.Stderr, "Rows:\t\t%d\n", atomic.LoadInt64(&nRows))

			if nObservations > 0 {
				mutex.Lock()
				fmt.Fprintf(os.Stderr, "RPS  Min:\t%d\n", nMinRPS)
				fmt.Fprintf(os.Stderr, "RPS  Avg:\t%0.0f\n", float64(atomic.LoadInt64(&nRows))/float64(nObservations))
				fmt.Fprintf(os.Stderr, "RPS 50th:\t%d\n", n50RPS)
				fmt.Fprintf(os.Stderr, "RPS 80th:\t%d\n", n80RPS)
				fmt.Fprintf(os.Stderr, "RPS 95th:\t%d\n", n95RPS)
				fmt.Fprintf(os.Stderr, "RPS 99th:\t%d\n", n99RPS)
				fmt.Fprintf(os.Stderr, "RPS  Max:\t%d\n", nMaxRPS)
				mutex.Unlock()
			}
		} else {
			fmt.Fprintf(os.Stderr, "%-16s%19s\n", "Elapsed, sec:", render_number.RenderFloat("#,###.###", float64(tElapsed.Seconds())))
			fmt.Fprintf(os.Stderr, "%-16s%19s\n", "Size, bytes:", render_number.RenderInteger("#,###.", atomic.LoadInt64(&nBytes)))
			fmt.Fprintf(os.Stderr, "%-16s%19s\n", "Speed, bps:", render_number.RenderInteger("#,###.", int64(float64(atomic.LoadInt64(&nBytes))/tElapsed.Seconds())))
			fmt.Fprintf(os.Stderr, "%-16s%19s\n", "Rows:", render_number.RenderInteger("#,###.", atomic.LoadInt64(&nRows)))

			if nObservations > 0 {
				mutex.Lock()
				fmt.Fprintf(os.Stderr, "%-16s%19s\n", "RPS  Min:", render_number.RenderInteger("#,###.", nMinRPS))
				fmt.Fprintf(os.Stderr, "%-16s%19s\n", "RPS  Avg:", render_number.RenderFloat("#,###.", float64(atomic.LoadInt64(&nRows))/float64(nObservations)))
				fmt.Fprintf(os.Stderr, "%-16s%19s\n", "RPS 50th:", render_number.RenderInteger("#,###.", n50RPS))
				fmt.Fprintf(os.Stderr, "%-16s%19s\n", "RPS 80th:", render_number.RenderInteger("#,###.", n80RPS))
				fmt.Fprintf(os.Stderr, "%-16s%19s\n", "RPS 95th:", render_number.RenderInteger("#,###.", n95RPS))
				fmt.Fprintf(os.Stderr, "%-16s%19s\n", "RPS 99th:", render_number.RenderInteger("#,###.", n99RPS))
				fmt.Fprintf(os.Stderr, "%-16s%19s\n", "RPS  Max:", render_number.RenderInteger("#,###.", nMaxRPS))
				mutex.Unlock()
			}
		}
	}
}
