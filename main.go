package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	addr := flag.String("l", ":443", "Listen address (e.g. :443 or :8443)")
	targetPort := flag.String("p", "443", "Default target port")
	verbose := flag.Bool("v", false, "Enable verbose logging")
	flag.Parse()

	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}

	// 打印 Banner
	log.Printf("======================================")
	log.Printf("  Snishaper Dedicated SNI Proxy (tls-rf)")
	log.Printf("  Listen: %s", *addr)
	log.Printf("  Feature: 4-Record Reassembly & 0x04 Version Check")
	log.Printf("======================================")

	proxy := NewSNIProxy(*addr)
	proxy.TargetPort = *targetPort
	
	if err := proxy.Start(); err != nil {
		log.Printf("[Fatal] Server failed to start: %v", err)
		os.Exit(1)
	}
}
