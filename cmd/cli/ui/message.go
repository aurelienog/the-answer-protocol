package ui

import "fmt"

const (
	reset   = "\033[0m"
	white   = "\033[97m"
	cyan    = "\033[36m"
	red     = "\033[31m"
	redBold = "\033[1;31m"
	magenta = "\033[35m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	gray    = "\033[90m"
)

func PrintMessage(message string) {
	fmt.Printf("%s%s%s\n", white, message, reset)
}

func PrintInfo(message string) {
	fmt.Printf("%s[INFO] %s%s\n", cyan, message, reset)
}

func PrintError(message string) {
	fmt.Printf("%s[ERROR] %s%s\n", redBold, message, reset)
}

func PrintEvent(message string) {
	fmt.Printf("%s[EVENT] %s%s\n", magenta, message, reset)
}

func PrintConnectionError(err error) {
	PrintError(fmt.Sprintf("Failed to connect to server: %v", err))
}