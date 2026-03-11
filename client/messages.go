package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// Message IDs
const (
	MsgID_C2S_LoginRequest    = 1001
	MsgID_C2S_RegisterRequest = 1002
	MsgID_C2S_MoveRequest     = 1003
	MsgID_C2S_BuildRequest    = 1004
	MsgID_S2C_LoginResponse   = 2001
	MsgID_S2C_RegisterResponse = 2002
	MsgID_S2C_MoveResponse    = 2003
	MsgID_S2C_BuildResponse   = 2004
)

// C2S Messages
type C2S_LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type C2S_RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type C2S_MoveRequest struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

type C2S_BuildRequest struct {
	BuildingType string `json:"building_type"`
	X            int32  `json:"x"`
	Y            int32  `json:"y"`
}

// S2C Messages
type S2C_LoginResponse struct {
	Success    bool       `json:"success"`
	Message    string     `json:"message"`
	PlayerId   uint64     `json:"player_id"`
	PlayerData *PlayerData `json:"player_data,omitempty"`
}

type S2C_RegisterResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	PlayerId uint64 `json:"player_id"`
}

type S2C_MoveResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	X       int32  `json:"x"`
	Y       int32  `json:"y"`
}

type S2C_BuildResponse struct {
	Success  bool      `json:"success"`
	Message  string    `json:"message"`
	Building *Building `json:"building,omitempty"`
}

// Data Structures
type PlayerData struct {
	PlayerId   uint64            `json:"player_id"`
	Username   string            `json:"username"`
	Email      string            `json:"email"`
	Level      int32             `json:"level"`
	Experience int64             `json:"experience"`
	X          int32             `json:"x"`
	Y          int32             `json:"y"`
	Resources  map[string]int64  `json:"resources"`
	Buildings  []*Building       `json:"buildings"`
	CreatedAt  int64             `json:"created_at"`
	LastLogin  int64             `json:"last_login"`
}

type Building struct {
	BuildingId   uint64 `json:"building_id"`
	BuildingType string `json:"building_type"`
	Level        int32  `json:"level"`
	X            int32  `json:"x"`
	Y            int32  `json:"y"`
}

// Packet represents a network packet
type Packet struct {
	MsgID uint32
	Data  json.RawMessage
}

// PrettyPrint prints a packet in a readable format
func (p *Packet) PrettyPrint() {
	fmt.Printf("\n=== Packet (MsgID: %d) ===\n", p.MsgID)
	
	var data map[string]interface{}
	if err := json.Unmarshal(p.Data, &data); err != nil {
		fmt.Printf("Raw Data: %s\n", string(p.Data))
		return
	}
	
	prettyJSON, _ := json.MarshalIndent(data, "", "  ")
	fmt.Printf("Data:\n%s\n", string(prettyJSON))
	fmt.Println("=========================\n")
}

// ParseResponse parses a response packet
func ParseResponse(msgID uint32, data []byte) (interface{}, error) {
	switch msgID {
	case MsgID_S2C_LoginResponse:
		var resp S2C_LoginResponse
		err := json.Unmarshal(data, &resp)
		return resp, err
	case MsgID_S2C_RegisterResponse:
		var resp S2C_RegisterResponse
		err := json.Unmarshal(data, &resp)
		return resp, err
	case MsgID_S2C_MoveResponse:
		var resp S2C_MoveResponse
		err := json.Unmarshal(data, &resp)
		return resp, err
	case MsgID_S2C_BuildResponse:
		var resp S2C_BuildResponse
		err := json.Unmarshal(data, &resp)
		return resp, err
	default:
		return nil, fmt.Errorf("unknown message ID: %d", msgID)
	}
}

// TestClient is a test client for the game server
type TestClient struct {
	client   *Client
	playerId uint64
	username string
	isLoggedIn bool
}

// NewTestClient creates a new test client
func NewTestClient(serverAddr string) *TestClient {
	return &TestClient{
		client: NewClient(serverAddr),
	}
}

// Connect connects to the server
func (tc *TestClient) Connect() error {
	return tc.client.Connect()
}

// Register registers a new account
func (tc *TestClient) Register(username, password, email string) (*S2C_RegisterResponse, error) {
	req := &C2S_RegisterRequest{
		Username: username,
		Password: password,
		Email:    email,
	}

	if err := tc.client.Send(MsgID_C2S_RegisterRequest, req); err != nil {
		return nil, err
	}

	packet, err := tc.client.RecvWithTimeout(10 * time.Second)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[TEST] Register response received (MsgID: %d)\n", packet.MsgID)
	packet.PrettyPrint()

	var resp S2C_RegisterResponse
	if err := json.Unmarshal(packet.Data, &resp); err != nil {
		return nil, err
	}

	if resp.Success {
		tc.playerId = resp.PlayerId
		tc.username = username
	}

	return &resp, nil
}

// Login logs in to the server
func (tc *TestClient) Login(username, password string) (*S2C_LoginResponse, error) {
	req := &C2S_LoginRequest{
		Username: username,
		Password: password,
	}

	if err := tc.client.Send(MsgID_C2S_LoginRequest, req); err != nil {
		return nil, err
	}

	packet, err := tc.client.RecvWithTimeout(10 * time.Second)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[TEST] Login response received (MsgID: %d)\n", packet.MsgID)
	packet.PrettyPrint()

	var resp S2C_LoginResponse
	if err := json.Unmarshal(packet.Data, &resp); err != nil {
		return nil, err
	}

	if resp.Success {
		tc.playerId = resp.PlayerId
		tc.username = username
		tc.isLoggedIn = true
	}

	return &resp, nil
}

// Move sends a move request
func (tc *TestClient) Move(x, y int32) (*S2C_MoveResponse, error) {
	if !tc.isLoggedIn {
		return nil, fmt.Errorf("not logged in")
	}

	req := &C2S_MoveRequest{X: x, Y: y}

	if err := tc.client.Send(MsgID_C2S_MoveRequest, req); err != nil {
		return nil, err
	}

	packet, err := tc.client.RecvWithTimeout(10 * time.Second)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[TEST] Move response received (MsgID: %d)\n", packet.MsgID)
	packet.PrettyPrint()

	var resp S2C_MoveResponse
	if err := json.Unmarshal(packet.Data, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Build sends a build request
func (tc *TestClient) Build(buildingType string, x, y int32) (*S2C_BuildResponse, error) {
	if !tc.isLoggedIn {
		return nil, fmt.Errorf("not logged in")
	}

	req := &C2S_BuildRequest{
		BuildingType: buildingType,
		X:            x,
		Y:            y,
	}

	if err := tc.client.Send(MsgID_C2S_BuildRequest, req); err != nil {
		return nil, err
	}

	packet, err := tc.client.RecvWithTimeout(10 * time.Second)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[TEST] Build response received (MsgID: %d)\n", packet.MsgID)
	packet.PrettyPrint()

	var resp S2C_BuildResponse
	if err := json.Unmarshal(packet.Data, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Close closes the connection
func (tc *TestClient) Close() {
	tc.client.Close()
}

// GetPlayerId returns the player ID
func (tc *TestClient) GetPlayerId() uint64 {
	return tc.playerId
}

// IsLoggedIn returns the login status
func (tc *TestClient) IsLoggedIn() bool {
	return tc.isLoggedIn
}
