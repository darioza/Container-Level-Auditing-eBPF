// This program demonstrates how to attach an eBPF program to a uretprobe.
// The program will be attached to the 'readline' symbol in the binary '/bin/bash' and print out
// the line which 'readline' functions returns to the caller.

//go:build amd64

package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
	"github.com/cilium/ebpf/rlimit"
	"golang.org/x/sys/unix"
)

// $BPF_CLANG and $BPF_CFLAGS are set by the Makefile.
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc $BPF_CLANG -cflags $BPF_CFLAGS -target amd64 -type event bpf uretprobe.c -- -I../headers

const (
	// The path to the ELF binary containing the function to trace.
	// On some distributions, the 'readline' function is provided by a
	// dynamically-linked library, so the path of the library will need
	// to be specified instead, e.g. /usr/lib/libreadline.so.8.
	// Use `ldd /bin/bash` to find these paths.
	binPath = "/bin/bash"
	symbol  = "readline"
)

func main() {
	stopper := make(chan os.Signal, 1)
	signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)

	// Allow the current process to lock memory for eBPF resources.
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal(err)
	}

	// Load pre-compiled programs and maps into the kernel.
	objs := bpfObjects{}
	if err := loadBpfObjects(&objs, nil); err != nil {
		log.Fatalf("loading objects: %s", err)
	}
	defer objs.Close()

	// Open an ELF binary and read its symbols.
	ex, err := link.OpenExecutable(binPath)
	if err != nil {
		log.Fatalf("opening executable: %s", err)
	}

	// Open a Uretprobe at the exit point of the symbol and attach
	// the pre-compiled eBPF program to it.
	up, err := ex.Uretprobe(symbol, objs.UretprobeBashReadline, nil)
	if err != nil {
		log.Fatalf("creating uretprobe: %s", err)
	}
	defer up.Close()

	// Open a perf event reader from userspace on the PERF_EVENT_ARRAY map
	// described in the eBPF C program.
	rd, err := perf.NewReader(objs.Events, os.Getpagesize())
	if err != nil {
		log.Fatalf("creating perf event reader: %s", err)
	}
	defer rd.Close()

	go func() {
		// Wait for a signal and close the perf reader,
		// which will interrupt rd.Read() and make the program exit.
		<-stopper
		log.Println("Received signal, exiting program..")

		if err := rd.Close(); err != nil {
			log.Fatalf("closing perf event reader: %s", err)
		}
	}()

	log.Printf("Listening for events..")

	// bpfEvent is generated by bpf2go.
	var event bpfEvent
	for {
		record, err := rd.Read()
		if err != nil {
			if errors.Is(err, perf.ErrClosed) {
				return
			}
			log.Printf("reading from perf event reader: %s", err)
			continue
		}

		if record.LostSamples != 0 {
			log.Printf("perf event ring buffer full, dropped %d samples", record.LostSamples)
			continue
		}

		// Parse the perf event entry into a bpfEvent structure.
		if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &event); err != nil {
			log.Printf("parsing perf event: %s", err)
			continue
		}

		msg := fmt.Sprintf("%s:%s return value: %s", binPath, symbol, unix.ByteSliceToString(event.Line[:]))
		log.Print(msg)

		// Send over to daemon to create the k8s event
		send(unix.ByteSliceToString(event.Line[:]))
	}
}
func send(msg string) {
	socketPath := "/socket/mysocket.sock"

	// Establish a connection to the Unix domain socket
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Create an example request body
	// requestBody := []byte(`{"message": "example message"}`)
	data := url.Values{}
	data.Set("message", msg)
	requestBody := strings.NewReader(data.Encode())

	// Prepare the HTTP request
	request, err := http.NewRequest("POST", "http://localhost/event", requestBody)
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	log.Printf("sent follwoing request: %q", data.Encode())

	// // Set the necessary headers
	// request.Header.Set("Content-Type", "application/json")
	// request.Header.Set("Connection", "close")

	// Send the HTTP request via the Unix domain socket connection
	err = request.Write(conn)
	if err != nil {
		log.Fatal(err)
	}
}
