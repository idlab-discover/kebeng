package monitoring

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"
)

type recordEvent struct {
	Handler  string
	Duration time.Duration
}

type fileRecorder struct {
	events        chan recordEvent
	writer        *bufio.Writer
	file          *os.File
	flushInterval time.Duration
	done          chan struct{}
	wg            sync.WaitGroup // <-- track in-flight events
}

var (
	mu       sync.Mutex
	recorder *fileRecorder
)

func InitFileRecorder(path string, flushInterval time.Duration, bufferSize int) error {
	mu.Lock()
	defer mu.Unlock()

	if recorder != nil {
		recorder.shutdown() // clean up any existing one
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
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
	return nil
}

// run writes events and calls wg.Done() for each
func (r *fileRecorder) run() {
	ticker := time.NewTicker(r.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case ev, ok := <-r.events:
			if !ok {
				// channel closed → finalize
				r.writer.Flush()
				r.file.Close()
				close(r.done)
				return
			}
			// write and mark this event done
			r.writer.WriteString(
				fmt.Sprintf("%s,%d\n", ev.Handler, ev.Duration.Nanoseconds()))
			r.wg.Done()
		case <-ticker.C:
			r.writer.Flush()
		}
	}
}

// RecordToFile adds one event to the queue (thread-safe)
func RecordToFile(handler string, d time.Duration) {
	mu.Lock()
	rec := recorder
	mu.Unlock()
	if rec == nil {
		return
	}
	// increment wg before enqueue
	rec.wg.Add(1)
	select {
	case rec.events <- recordEvent{handler, d}:
		// enqueued; run() will call wg.Done() once written
	default:
		// channel full: drop and decrement
		rec.wg.Done()
	}
}

// shutdown waits for all events to be processed, then closes
func (r *fileRecorder) shutdown() {
	// wait until run() has processed every Add()
	r.wg.Wait()
	// now safe to close channel and let run() clean up
	close(r.events)
	<-r.done
}

// ShutdownFileRecorder exposes that cleanup
func ShutdownFileRecorder() {
	mu.Lock()
	defer mu.Unlock()
	if recorder != nil {
		recorder.shutdown()
		recorder = nil
	}
}
