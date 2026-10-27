package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"the-answer-protocol/internal/config"
	"the-answer-protocol/internal/desktop"
)

func main() {
	_ = config.LoadConfig()

	app := desktop.NewApp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/session", app.HandleSession)
	mux.HandleFunc("/api/connect", app.HandleConnect)
	mux.HandleFunc("/api/command", app.HandleCommand)
	mux.HandleFunc("/api/stream", app.HandleStream)
	mux.Handle("/", http.FileServer(http.Dir("./frontend/gui")))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("Failed to start desktop GUI bridge: %v", err)
	}
	defer listener.Close()

	url := fmt.Sprintf("http://%s", listener.Addr().String())
	log.Printf("TAP Desktop GUI started! Window URL: %s", url)

	// Create a temporary profile folder for the game window,
	// so that Chrome runs as a separate process and we can close it using Ctrl+C!
	tempProfile, _ := os.MkdirTemp("", "tap-gui-*")

	var browserCmd *exec.Cmd
	go func() {
		browserCmd = openNativeWindow(url, tempProfile)
		if browserCmd != nil {
			// If the user closes the window using the [X] button, the browser process will terminate automatically!
			_ = browserCmd.Wait()
			_ = os.RemoveAll(tempProfile)
			app.Shutdown()
		}
	}()

	// Intercepting Ctrl+C (SIGINT / SIGTERM) in the terminal!
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("\nCtrl+C received! Closing GUI window and disconnecting...")
		if browserCmd != nil && browserCmd.Process != nil {
			_ = browserCmd.Process.Kill()
		}
		_ = os.RemoveAll(tempProfile)
		app.Shutdown()
	}()

	if err := http.Serve(listener, mux); err != nil {
		log.Fatalf("GUI server stopped: %v", err)
	}
}

func openNativeWindow(url string, profileDir string) *exec.Cmd {
	time.Sleep(200 * time.Millisecond)

	chromeFlags := []string{
		"--app=" + url,
		"--window-size=1366,840",
		"--user-data-dir=" + profileDir,
		"--no-first-run",
		"--no-default-browser-check",
	}

	candidates := []struct {
		bin  string
		args []string
	}{
		{"google-chrome", chromeFlags},
		{"chromium", chromeFlags},
		{"chromium-browser", chromeFlags},
		{"brave-browser", chromeFlags},
		{"firefox", []string{"-new-instance", "-profile", profileDir, "-new-window", url}},
		{"xdg-open", []string{url}},
	}

	for _, c := range candidates {
		if path, err := exec.LookPath(c.bin); err == nil {
			cmd := exec.Command(path, c.args...)
			if err := cmd.Start(); err == nil {
				return cmd
			}
		}
	}
	return nil
}
