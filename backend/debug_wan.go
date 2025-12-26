package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"monik-enterprise/internal/config"
	"monik-enterprise/internal/service"

	"github.com/go-routeros/routeros/v3"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("=== WAN Interface Detection Debug Tool ===")

	// Load .env file manually
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Load configuration
	cfg := config.Load()
	fmt.Printf("Router IP: %s\n", cfg.Router.IP)
	fmt.Printf("Router Port: %d\n", cfg.Router.Port)
	fmt.Printf("Router Username: %s\n", cfg.Router.Username)
	fmt.Printf("Router Password: %s\n", cfg.Router.Password)
	fmt.Printf("WAN Detection Enabled: %t\n", cfg.WAN.Enabled)
	fmt.Printf("Detection Method: %s\n", cfg.WAN.DetectionMethod)

	// Test MikroTik connection
	fmt.Println("\n=== Testing MikroTik Connection ===")
	// Use the correct IP and port from .env
	routerIP := cfg.Router.IP
	routerPort := cfg.Router.Port
	fmt.Printf("Connecting to: %s:%d\n", routerIP, routerPort)

	client, err := routeros.Dial(routerIP+":"+fmt.Sprintf("%d", routerPort), cfg.Router.Username, cfg.Router.Password)
	if err != nil {
		fmt.Printf("❌ Failed to connect to MikroTik: %v\n", err)
		return
	}
	defer client.Close()

	fmt.Println("✅ MikroTik connection successful")

	// Login
	err = client.Login(cfg.Router.Username, cfg.Router.Password)
	if err != nil {
		fmt.Printf("❌ Failed to login to MikroTik: %v\n", err)
		return
	}
	fmt.Println("✅ MikroTik login successful")

	// Test WAN detection
	fmt.Println("\n=== Testing WAN Detection ===")
	wanService := service.NewWANDetectionService(cfg.WAN)
	wanService.SetRouterClient(client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	wan, err := wanService.DetectWANInterface(ctx)
	if err != nil {
		fmt.Printf("❌ WAN detection failed: %v\n", err)
	} else {
		fmt.Printf("✅ WAN interface detected: %s\n", wan.Name)
		fmt.Printf("   Method: %s\n", wan.Method)
		fmt.Printf("   Confidence: %.2f\n", wan.Confidence)
		fmt.Printf("   ISP: %s\n", wan.ISPName)
		fmt.Printf("   Traffic: %d bytes\n", wan.Traffic)
	}

	// Test route-based detection specifically
	fmt.Println("\n=== Testing Route-Based Detection ===")
	reply, err := client.RunContext(ctx, "/ip/route/print", "?dst-address=0.0.0.0/0")
	if err != nil {
		fmt.Printf("❌ Failed to get routes: %v\n", err)
	} else {
		fmt.Printf("✅ Got %d routes\n", len(reply.Re))
		for i, re := range reply.Re {
			fmt.Printf("   Route %d: %v\n", i+1, re.Map)
		}
	}

	// Test interface listing
	fmt.Println("\n=== Testing Interface Listing ===")
	interfaces, err := client.RunContext(ctx, "/interface/print")
	if err != nil {
		fmt.Printf("❌ Failed to get interfaces: %v\n", err)
	} else {
		fmt.Printf("✅ Got %d interfaces\n", len(interfaces.Re))
		for i, re := range interfaces.Re {
			fmt.Printf("   Interface %d: %s (status: %s)\n", i+1, re.Map["name"], re.Map["running"])
		}
	}

	fmt.Println("\n=== Debug completed ===")
}
