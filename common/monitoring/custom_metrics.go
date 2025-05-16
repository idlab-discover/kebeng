package monitoring

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	recorder *fileRecorder
	once     sync.Once
)

// recordEvent is one line in your CSV.
type recordEvent struct {
	Handler  string
	Duration time.Duration
}

// fileRecorder holds the writer and channel.
type fileRecorder struct {
	events        chan recordEvent
	writer        *bufio.Writer
	file          *os.File
	flushInterval time.Duration
	done          chan struct{}
}

// InitFileRecorder sets up the file writer.  Call this once at startup.
func InitFileRecorder(path string, flushInterval time.Duration, bufferSize int) error {
	var err error
	once.Do(func() {
		f, e := os.OpenFile(path,
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if e != nil {
			err = e
			return
		}
		rec := &fileRecorder{
			events:        make(chan recordEvent, bufferSize),
			writer:        bufio.NewWriter(f),
			file:          f,
			flushInterval: flushInterval,
			done:          make(chan struct{}),
		}
		recorder = rec
		go rec.run()
	})
	return err
}

// run is the goroutine that writes events to disk.
func (r *fileRecorder) run() {
	ticker := time.NewTicker(r.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case ev := <-r.events:
			// Write one CSV line: handler,duration_ns
			r.writer.WriteString(fmt.Sprintf("%s,%d\n",
				ev.Handler, ev.Duration.Nanoseconds()))
		case <-ticker.C:
			r.writer.Flush()
		case <-r.done:
			// drain remaining events
			for ev := range r.events {
				r.writer.WriteString(fmt.Sprintf("%s,%d\n",
					ev.Handler, ev.Duration.Milliseconds()))
			}
			r.writer.Flush()
			r.file.Close()
			close(r.done)
			return
		}
	}
}

// RecordToFile enqueues one (handler, duration) for later writing.
// It returns immediately (unless the channel buffer is full).
func RecordToFile(handler string, d time.Duration) {
	if recorder == nil {
		return
	}
	rec := recordEvent{Handler: handler, Duration: d}
	// You could use a non‐blocking select here if you want to drop on overflow:
	// select { case recorder.events <- rec: default: }
	recorder.events <- rec
}

// ShutdownFileRecorder flushes and closes the file.  Call on program exit.
func ShutdownFileRecorder() {
	if recorder == nil {
		return
	}
	// Signal goroutine to finish
	recorder.done <- struct{}{}
	// Wait for it to close
	<-recorder.done
}
