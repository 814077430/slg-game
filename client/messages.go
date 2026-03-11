package main

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
	PlayerId  uint64           `json:"player_id"`
	Username  string           `json:"username"`
	Email     string           `json:"email"`
	Level     int32            `json:"level"`
	Resources map[string]int64 `json:"resources"`
}

type Building struct {
	BuildingType string `json:"building_type"`
	Level        int32  `json:"level"`
	X            int32  `json:"x"`
	Y            int32  `json:"y"`
}
