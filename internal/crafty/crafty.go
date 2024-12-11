package crafty

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"onemc/internal/utils"
)

var (
	config  utils.Config
	cToken  string
	running bool
	online  int
)

func init() {
	utils.MustLoadConfig(&config)
}

func fetchToken() string {
	url := config.URL + "auth/login"
	body := []byte(`{"username":"` + config.Username + `","password":"` + config.Password + `"}`)

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	type APIResponse struct {
		Status string `json:"status"`
		Data   struct {
			Token string `json:"token"`
		} `json:"data"`
	}

	post, err := httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Println("token", err)
		return ""
	}
	defer post.Body.Close()

	if post.StatusCode != 200 {
		fmt.Printf("error with response: %v", post.Status)
	}
	var apiResponse APIResponse
	if err := json.NewDecoder(post.Body).Decode(&apiResponse); err != nil {
		log.Fatal("token decode", err)
	}

	if apiResponse.Data.Token == "" {
		log.Fatal("token empty")
	}

	return apiResponse.Data.Token
}

func fetchStats() (bool, int) {
	url := fmt.Sprintf("%sservers/%s/stats", config.URL, config.ServerID)
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	type APIResponse struct {
		Status string `json:"status"`
		Data   struct {
			Online  int    `json:"online"`
			Players string `json:"players"`
			Running bool   `json:"running"`
		} `json:"data"`
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("HTTP request failed: %v", err)
		return false, 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected response status: %d", resp.StatusCode)
		return false, 0
	}

	var apiResponse APIResponse
	err = json.NewDecoder(resp.Body).Decode(&apiResponse)
	if err != nil {
		log.Fatalf("Failed to decode response body: %v", err)
	}

	//log.Printf("Response: %+v", apiResponse)

	// Return the parsed values
	return apiResponse.Data.Running, apiResponse.Data.Online
}

func updateStats() {
	running, online = fetchStats()
}

func StopServer() error {
	url := fmt.Sprintf("%sservers/%s/action/stop_server", config.URL, config.ServerID)

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Consider setting up proper TLS for production
			},
		},
	}

	type APIResponse struct {
		Status string `json:"status"`
	}

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers before making the request
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check the HTTP status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error with response: %s", resp.Status)
	}

	var apiResponse APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if apiResponse.Status == "ok" {
		log.Printf("Minecraft server stopped!")
	}

	return nil
}

func StopQuery() bool {
	updateStats()
	if !running {
		log.Println("Minecraft server isn't running")
	}
	if online > 0 {
		log.Printf("%v player(s) are currently online", online)
		return false
	}
	return true
}

func StartServer() error {
	cToken = fetchToken()
	url := fmt.Sprintf("%sservers/%s/action/start_server", config.URL, config.ServerID)

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Consider setting up proper TLS for production
			},
		},
	}

	type APIResponse struct {
		Status string `json:"status"`
	}

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers before making the request
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check the HTTP status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error with response: %s", resp.Status)
	}

	var apiResponse APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if apiResponse.Status == "ok" {
		fmt.Print("Minecraft server started!")
	}

	return nil
}
