package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var pids = make(chan int, 100)

var paths = make(chan string, 100)

func handleHello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello from the other side\n")
}

func handlePID(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	path := r.FormValue("path")
	paths <- path
}

func process(path string) error {
	log.Printf("starting to process path: %s", path)

	var pid int
	for i := 0; i < 10; i++ {
		log.Printf("trying to read path: %s", path)
		time.Sleep(1 * time.Second)
		bytes, err := os.ReadFile(path)
		if err != nil {
			log.Printf("error reading file: %v", err)
			continue
		}

		pid, err = strconv.Atoi(strings.TrimSpace(string(bytes)))
		if err != nil {
			log.Printf("error parsing to integer: %v", err)
			continue
		}

		if pid > 0 {
			break
		}
	}

	if pid < 1 {
		return fmt.Errorf("failed to get a valid pid: %d", pid)
	}

	log.Printf("starting nsenter on pid: %d", pid)
	cmd := exec.Command("nsenter", "--target", fmt.Sprintf("%d", pid), "-a", "/data/uretprobe")

	log.Printf("logging to: /tmp/nsenter-%d.txt", pid)
	outfile, err := os.Create(fmt.Sprintf("/tmp/nsenter-%d.txt", pid))
	if err != nil {
		return err
	}
	defer outfile.Close()
	cmd.Stdout = outfile
	cmd.Stderr = outfile

	// Fire and forget
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to run nsenter: %v", err)
	}

	// log.Printf("Waiting pid: %d", pid)

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("failed to wait command: %v", err)
	}

	fmt.Printf("done with pid %d\n", pid)
	return nil
}

func main() {
	kubeconfig := flag.String("kubeconfig", "", "path to the kubeconfig file")
	flag.Parse()

	// Check if kubeconfig path is provided as a command-line argument
	if *kubeconfig == "" {
		// If not provided, try fetching from environment variable
		*kubeconfig = os.Getenv("KUBECONFIG")
	}

	// Use the default kubeconfig path if still not provided
	if *kubeconfig == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		*kubeconfig = homeDir + "/.kube/config"
	}

	// Build the client config
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Printf("waiting for paths channel")
		path := <-paths
		log.Printf("pid  channel %s", path)

		go func() {
			err := process(path)
			if err != nil {
				log.Printf("error processing path: %v", err)
			}
		}()
	}()

	socketPath := "/tmp/mysocket.sock"

	// Remove the socket file if it already exists
	_ = os.Remove(socketPath)

	// Create a Unix listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	// Make sure file is acessible for other users
	err = os.Chmod(socketPath, 0666)
	if err != nil {
		log.Fatal(err)
	}

	// Create a new ServeMux
	mux := http.NewServeMux()

	// Handle the /hello endpoint
	mux.HandleFunc("/pid2", handlePID)
	mux.HandleFunc("/event", handleEvent(clientset))
	mux.HandleFunc("/hello", handleHello)

	// Create an HTTP server with the ServeMux as the handler
	server := &http.Server{
		Handler: mux,
	}

	fmt.Printf("HTTP server listening on Unix socket: %s\n", socketPath)

	// Serve HTTP requests
	err = server.Serve(listener)
	if err != nil {
		log.Fatal(err)
	}

	// http.HandleFunc("/pid2", handlePID)
	// http.HandleFunc("/event", handleEvent(clientset))
	// http.ListenAndServe("0.0.0.0:8081", nil)
}

func handleEvent(clientset *kubernetes.Clientset) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		msg := r.FormValue("message")
		log.Printf("Got event message: %s", msg)

		event := &corev1.Event{
			ObjectMeta: v1.ObjectMeta{
				Name:      fmt.Sprintf("example-event-%d", rand.Int()),
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind:      "Pod",
				Namespace: "default",
				Name:      "nginx-deployment-89c6ff86b-92ndx",
			},
			Reason:  "ShellCommandExecutedReason",
			Message: fmt.Sprintf("Command executed: %q", msg),
			Type:    corev1.EventTypeNormal,
		}

		// Create the event using the clientset
		createdEvent, err := clientset.CoreV1().Events("default").Create(context.Background(), event, v1.CreateOptions{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			// http.Error(w, "failedblah", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Event created: %s\n", createdEvent.Name)
	}
}
