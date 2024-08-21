package main

import "fmt"

// Common broker/v1 client API prefix.
const apiPrefix = "/api/broker/v1"

// Prefix for envionment variables.
const envconfigPrefix = "WASIMOFF"

// Print a figlet "wasmoff" banner.
// figlet -f small wasimoff | sed -e 's/\\/\\\\/g' -e 's/.*/log.Println("&")/'
func banner() {
	fmt.Println("                  _            __  __ ")
	fmt.Println("  __ __ ____ _ __(_)_ __  ___ / _|/ _|")
	fmt.Println("  \\ V  V / _` (_-< | '  \\/ _ \\  _|  _|")
	fmt.Println("   \\_/\\_/\\__,_/__/_|_|_|_\\___/_| |_|  ")
	fmt.Println()
}
