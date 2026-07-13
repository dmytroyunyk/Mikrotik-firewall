package main

import (
	"flag"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

func main() {
	target := flag.String("target", "192.168.88.1:22", "Router IP:port to test")
	mode := flag.String("mode", "ssh", "Attack type: ssh, scan")
	count := flag.Int("count", 15, "Number of attempts")
	delay := flag.Int("delay", 500, "Delay between attempts in milliseconds")
	flag.Parse()

	fmt.Printf("🎯 Target: %s\n", *target)
	fmt.Printf("⚔️  Mode: %s\n", *mode)
	fmt.Printf("🔄 Attempts: %d\n", *count)
	fmt.Printf("⏱  Delay: %dms\n\n", *delay)

	switch *mode {
	case "ssh":
		simulateSSHBruteForce(*target, *count, *delay)
	case "scan":
		simulatePortScan(*target, *count, *delay)
	default:
		fmt.Printf("Unknown mode: %s\n", *mode)
	}
}

func simulateSSHBruteForce(target string, count, delayMs int) {
	fmt.Print("🔐 Starting SSH brute-force simulation...\n\n")

	success := 0
	failed := 0

	for i := 1; i <= count; i++ {
		fakePassword := fmt.Sprintf("wrong-password-%d", i)

		config := &ssh.ClientConfig{
			User: "admin",
			Auth: []ssh.AuthMethod{
				ssh.Password(fakePassword),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         3 * time.Second,
		}

		fmt.Printf("[%d/%d] SSH login attempt with password '%s'... ", i, count, fakePassword)

		conn, err := ssh.Dial("tcp", target, config)
		if err != nil {
			fmt.Println("❌ Failed (expected)")
			failed++
		} else {
			fmt.Println("⚠️  Connected (unexpected!)")
			conn.Close()
			success++
		}

		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	fmt.Printf("\n📊 Results: %d failed, %d success\n", failed, success)
	fmt.Println("✅ Simulation complete. Check your Telegram for alerts!")
}

func simulatePortScan(target string, count, delayMs int) {
	host, _, err := net.SplitHostPort(target)
	if err != nil || host == "" {
		host = target
	}

	ports := []int{
		21, 22, 23, 25, 53, 80, 110, 143,
		443, 445, 993, 995, 1723, 3306,
		3389, 5432, 5900, 8080, 8443, 8728,
	}

	fmt.Print("🔍 Starting port scan simulation...\n\n")

	open := 0
	closed := 0
	scanned := 0

	for _, port := range ports {
		if scanned >= count {
			break
		}
		scanned++

		address := fmt.Sprintf("%s:%d", host, port)
		fmt.Printf("[%d/%d] Scanning port %d... ", scanned, count, port)

		conn, err := net.DialTimeout("tcp", address, 2*time.Second)
		if err != nil {
			fmt.Println("❌ Closed")
			closed++
		} else {
			fmt.Println("✅ Open")
			conn.Close()
			open++
		}

		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	fmt.Printf("\n📊 Results: %d open, %d closed\n", open, closed)
	fmt.Println("✅ Scan complete. Check your firewall logs!")
}
