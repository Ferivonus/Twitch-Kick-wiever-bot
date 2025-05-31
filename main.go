package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	threads        = 10
	proxyFile      = "dataa.txt"
	mpvPath        = `C:\Path\To\mpv.exe`
	streamlinkPath = `C:\Path\To\streamlink.exe`
	logOutput      = true
)

func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetFlags(log.Ltime | log.Lshortfile)

	// graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("[Main] Interrupt received, shutting down...")
		cancel()
	}()

	// get URL
	targetURL := promptURL()
	if targetURL == "" {
		log.Fatal("[Main] No URL entered, exiting.")
	}

	// load proxies
	proxies := readProxies(proxyFile)
	if len(proxies) == 0 {
		log.Println("[Main] No proxies found; running without proxies.")
	}

	// verify streamlink exists
	if _, err := exec.LookPath(streamlinkPath); err != nil {
		log.Fatalf("[Main] streamlink not found at %s: %v", streamlinkPath, err)
	}

	var wg sync.WaitGroup
	log.Printf("[Main] Launching %d workers…", threads)
	for i := 0; i < threads; i++ {
		if i > 0 {
			delay := time.Duration(rand.Intn(5)+1) * time.Second
			log.Printf("[Main] Waiting %v before worker %d…", delay, i+1)
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
				// Continue
			}
		}

		// Check for shutdown signal before launching a worker
		select {
		case <-ctx.Done():
			log.Printf("[Main] Worker launch cancelled during shutdown.")
			return
		default:
			// Continue launching
		}

		wg.Add(1)
		proxy := ""
		if len(proxies) > 0 {
			proxy = proxies[i%len(proxies)]
		}
		go worker(i+1, targetURL, proxy, ctx, &wg)
	}

	log.Println("[Main] All workers launched. Press Ctrl+C to stop.")
	wg.Wait()
	log.Println("[Main] All sessions ended.")
}

func promptURL() string {
	fmt.Print("Please enter the Twitch/Kick link to watch: ")
	reader := bufio.NewReader(os.Stdin)
	u, _ := reader.ReadString('\n')
	return strings.TrimSpace(u)
}

func readProxies(file string) []string {
	f, err := os.Open(file)
	if err != nil {
		log.Printf("[Proxy] Could not open %s: %v", file, err)
		return nil
	}
	defer f.Close()

	var out []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("[Proxy] Error reading %s: %v", file, err)
	}
	log.Printf("[Proxy] %d proxies loaded.", len(out))
	return out
}

func worker(id int, targetURL, proxy string, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	pfx := fmt.Sprintf("[Worker %02d]", id)

	// Arguments specifically for MPV player, to be passed as a single string to Streamlink's --player-args
	mpvPlayerArgs := []string{
		"--no-terminal",
		"--no-fs",
		"--pause=no",
		"--really-quiet",
		"--untimed",
		"--cache=no",
		"--profile=low-latency",
		"--force-seekable=no",
	}

	// Calculate window geometry dynamically for each worker
	startX := 100 // Initial X position for the first window
	startY := 50  // Initial Y position for the first window
	windowWidth := 640
	windowHeight := 480
	horizontalSpacing := 20 // Horizontal spacing between windows

	xPos := startX + (id-1)*(windowWidth+horizontalSpacing)
	yPos := startY // All windows start at the same Y position for simplicity

	geometryArg := fmt.Sprintf("--geometry=%dx%d+%d+%d", windowWidth, windowHeight, xPos, yPos)
	mpvPlayerArgs = append(mpvPlayerArgs, geometryArg) // Add dynamic geometry argument

	// Join MPV arguments into a single string, separated by spaces
	mpvPlayerArgsString := strings.Join(mpvPlayerArgs, " ")

	// basic Streamlink args
	args := []string{
		"--player", mpvPath,
		"--retry-open", "1",
		"--retry-streams", "1",
		"--retry-max", "1",
		// Add other Streamlink specific arguments here if needed, e.g.,
		// "--hls-segment-timeout", "30",
		// "--hls-segment-attempts", "3",
		// "--http-stream-timeout", "3600",
	}

	// proxy handling: schemeless → socks5
	if proxy != "" {
		if !strings.Contains(proxy, "://") {
			proxy = "socks5://" + proxy // Default to socks5 if no scheme is specified
		}
		// Only check for error from url.Parse, no need to store the parsed URL
		if _, err := url.Parse(proxy); err == nil {
			args = append(args, "--http-proxy", proxy)
			log.Printf("%s Using proxy %s", pfx, proxy)
		} else {
			log.Printf("%s Invalid proxy %q, skipping: %v", pfx, proxy, err) // Log the error for invalid proxy
		}
	} else {
		log.Printf("%s No proxy", pfx)
	}

	// Add the --player-args option with the joined MPV arguments string.
	if mpvPlayerArgsString != "" {
		args = append(args, "--player-args", mpvPlayerArgsString)
	}

	// target URL + worst quality
	args = append(args, targetURL, "worst")

	// Create the command with context for cancellation
	cmd := exec.CommandContext(ctx, streamlinkPath, args...)

	// Configure stdout and stderr based on logOutput constant
	if logOutput {
		cmd.Stdout = os.Stdout // Direct Streamlink's standard output to console
		cmd.Stderr = os.Stderr // Direct Streamlink's standard error to console
	} else {
		cmd.Stdout = io.Discard // Discard standard output
		cmd.Stderr = io.Discard // Discard standard error
	}

	log.Printf("%s Launching: %s %s", pfx, streamlinkPath, strings.Join(args, " "))

	// Run the command and handle potential errors
	err := cmd.Run()
	if err != nil {
		select {
		case <-ctx.Done():
			// If the context was cancelled, it means the program is shutting down
			log.Printf("%s Stopped by signal (context cancelled).", pfx)
		default:
			// Handle execution errors from Streamlink
			if exitErr, ok := err.(*exec.ExitError); ok {
				// Streamlink exited with a non-zero status
				log.Printf("%s Streamlink terminated unexpectedly. Exit Code: %d - Error: %v", pfx, exitErr.ExitCode(), err)
				// You can inspect exitErr.Stderr or exitErr.Stdout for more details if logOutput is false
			} else {
				// Other errors (e.g., command not found, permission issues)
				log.Printf("%s Error running Streamlink: %v", pfx, err)
			}
		}
	} else {
		// Command executed successfully
		log.Printf("%s Session ended normally.", pfx)
	}
}
