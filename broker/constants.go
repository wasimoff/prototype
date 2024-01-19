package main

import "log"

// Print a figlet "wasmoff" banner.
// figlet -f small wasimoff | sed -e 's/\\/\\\\/g' -e 's/.*/log.Println("&")/'
func banner() {
	log.Println("                _            __  __ ")
	log.Println("__ __ ____ _ __(_)_ __  ___ / _|/ _|")
	log.Println("\\ V  V / _` (_-< | '  \\/ _ \\  _|  _|")
	log.Println(" \\_/\\_/\\__,_/__/_|_|_|_\\___/_| |_|  ")
	log.Println()
}
