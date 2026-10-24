package ui

import "fmt"

// PrintBanner displays the TAP ASCII art.
func PrintBanner() {
	fmt.Println(`
  ████████╗ █████╗ ██████╗
  ╚══██╔══╝██╔══██╗██╔══██╗
     ██║   ███████║██████╔╝
     ██║   ██╔══██║██╔═══╝
     ██║   ██║  ██║██║
     ╚═╝   ╚═╝  ╚═╝╚═╝

       THE ANSWER PROTOCOL
     A TCP Multiplayer Adventure
`)
}