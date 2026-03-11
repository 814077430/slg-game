package main

import (
	"flag"
	"fmt"
	"log"
	"time"
)

func main() {
	serverAddr := flag.String("server", "localhost:8080", "Server address")
	testMode := flag.String("test", "all", "Test mode: all, login, register, move, build")
	flag.Parse()

	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║          SLG Game Server Test Client                   ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Printf("Server: %s\n", *serverAddr)
	fmt.Printf("Test Mode: %s\n\n", *testMode)

	switch *testMode {
	case "all":
		runAllTests(*serverAddr)
	case "login":
		runLoginTest(*serverAddr)
	case "register":
		runRegisterTest(*serverAddr)
	case "move":
		runMoveTest(*serverAddr)
	case "build":
		runBuildTest(*serverAddr)
	default:
		fmt.Printf("Unknown test mode: %s\n", *testMode)
		fmt.Println("Available modes: all, login, register, move, build")
	}
}

func runAllTests(serverAddr string) {
	fmt.Println("=== Running All Tests ===\n")

	// Test 1: Register
	fmt.Println("[TEST 1] Register new account")
	client := NewTestClient(serverAddr)
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	username := fmt.Sprintf("testuser_%d", time.Now().Unix())
	regResp, err := client.Register(username, "password123", "test@example.com")
	if err != nil {
		log.Printf("Register failed: %v", err)
	} else if !regResp.Success {
		log.Printf("Register failed: %s", regResp.Message)
	} else {
		fmt.Printf("✓ Register successful! Player ID: %d\n\n", regResp.PlayerId)
	}

	// Test 2: Login
	fmt.Println("[TEST 2] Login")
	loginResp, err := client.Login(username, "password123")
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	if !loginResp.Success {
		log.Fatalf("Login failed: %s", loginResp.Message)
	}
	fmt.Printf("✓ Login successful! Player ID: %d\n\n", loginResp.PlayerId)

	// Test 3: Move
	fmt.Println("[TEST 3] Move player")
	moveResp, err := client.Move(100, 200)
	if err != nil {
		log.Printf("Move failed: %v", err)
	} else if !moveResp.Success {
		log.Printf("Move failed: %s", moveResp.Message)
	} else {
		fmt.Printf("✓ Move successful! Position: (%d, %d)\n\n", moveResp.X, moveResp.Y)
	}

	// Test 4: Build
	fmt.Println("[TEST 4] Build structure")
	buildResp, err := client.Build("farm", 50, 50)
	if err != nil {
		log.Printf("Build failed: %v", err)
	} else if !buildResp.Success {
		log.Printf("Build failed: %s", buildResp.Message)
	} else {
		fmt.Printf("✓ Build successful! Building: %s at (%d, %d)\n\n",
			buildResp.Building.BuildingType,
			buildResp.Building.X,
			buildResp.Building.Y)
	}

	// Test 5: Multiple moves
	fmt.Println("[TEST 5] Multiple moves")
	positions := []struct{ x, y int32 }{
		{10, 10},
		{20, 20},
		{30, 30},
		{40, 40},
		{50, 50},
	}

	for i, pos := range positions {
		fmt.Printf("  Move %d/%d to (%d, %d)... ", i+1, len(positions), pos.x, pos.y)
		moveResp, err := client.Move(pos.x, pos.y)
		if err != nil {
			fmt.Printf("✗ Failed: %v\n", err)
		} else if moveResp.Success {
			fmt.Printf("✓ Success\n")
		} else {
			fmt.Printf("✗ Failed: %s\n", moveResp.Message)
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n=== All Tests Completed ===")
}

func runLoginTest(serverAddr string) {
	fmt.Println("=== Login Test ===\n")

	client := NewTestClient(serverAddr)
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	username := "testuser"
	password := "password123"

	fmt.Printf("Attempting login for user: %s\n", username)
	resp, err := client.Login(username, password)
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}

	if resp.Success {
		fmt.Printf("\n✓ Login successful!\n")
		fmt.Printf("  Player ID: %d\n", resp.PlayerId)
		fmt.Printf("  Username: %s\n", resp.PlayerData.Username)
		fmt.Printf("  Level: %d\n", resp.PlayerData.Level)
		fmt.Printf("  Resources: Gold=%d, Wood=%d, Food=%d\n",
			resp.PlayerData.Resources["gold"],
			resp.PlayerData.Resources["wood"],
			resp.PlayerData.Resources["food"])
	} else {
		fmt.Printf("\n✗ Login failed: %s\n", resp.Message)
	}
}

func runRegisterTest(serverAddr string) {
	fmt.Println("=== Register Test ===\n")

	client := NewTestClient(serverAddr)
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	username := fmt.Sprintf("newuser_%d", time.Now().Unix())
	password := "password123"
	email := "test@example.com"

	fmt.Printf("Registering new user: %s\n", username)
	resp, err := client.Register(username, password, email)
	if err != nil {
		log.Fatalf("Register failed: %v", err)
	}

	if resp.Success {
		fmt.Printf("\n✓ Registration successful!\n")
		fmt.Printf("  Player ID: %d\n", resp.PlayerId)
		fmt.Printf("  Username: %s\n", username)
	} else {
		fmt.Printf("\n✗ Registration failed: %s\n", resp.Message)
	}
}

func runMoveTest(serverAddr string) {
	fmt.Println("=== Move Test ===\n")

	client := NewTestClient(serverAddr)
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Login first
	username := "testuser"
	password := "password123"

	fmt.Printf("Logging in as: %s\n", username)
	loginResp, err := client.Login(username, password)
	if err != nil || !loginResp.Success {
		log.Fatalf("Login failed. Please register first.")
	}

	// Test moves
	testMoves := []struct {
		x, y int32
		desc string
	}{
		{0, 0, "Origin"},
		{100, 100, "Northeast"},
		{-50, -50, "Southwest"},
		{999, 999, "Far location"},
	}

	for _, move := range testMoves {
		fmt.Printf("\nMoving to %s (%d, %d)...\n", move.desc, move.x, move.y)
		resp, err := client.Move(move.x, move.y)
		if err != nil {
			fmt.Printf("  ✗ Error: %v\n", err)
		} else if resp.Success {
			fmt.Printf("  ✓ Success! Position: (%d, %d)\n", resp.X, resp.Y)
		} else {
			fmt.Printf("  ✗ Failed: %s\n", resp.Message)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func runBuildTest(serverAddr string) {
	fmt.Println("=== Build Test ===\n")

	client := NewTestClient(serverAddr)
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Login first
	username := "testuser"
	password := "password123"

	fmt.Printf("Logging in as: %s\n", username)
	loginResp, err := client.Login(username, password)
	if err != nil || !loginResp.Success {
		log.Fatalf("Login failed. Please register first.")
	}

	// Test builds
	testBuilds := []struct {
		buildingType string
		x, y         int32
		desc         string
	}{
		{"farm", 10, 10, "Farm"},
		{"lumber_mill", 20, 20, "Lumber Mill"},
		{"mine", 30, 30, "Mine"},
		{"barracks", 40, 40, "Barracks"},
	}

	for _, build := range testBuilds {
		fmt.Printf("\nBuilding %s at (%d, %d)...\n", build.desc, build.x, build.y)
		resp, err := client.Build(build.buildingType, build.x, build.y)
		if err != nil {
			fmt.Printf("  ✗ Error: %v\n", err)
		} else if resp.Success {
			fmt.Printf("  ✓ Success! Building: %s (Level: %d)\n",
				resp.Building.BuildingType,
				resp.Building.Level)
		} else {
			fmt.Printf("  ✗ Failed: %s\n", resp.Message)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
