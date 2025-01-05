package crafty

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"onemc/internal/aws"
	"onemc/internal/utils"
	"time"
)

var (
	config             utils.Config
	cToken             string
	mcCurrentlyRunning bool
	countOnline        int
)

func init() {
	utils.MustLoadConfig(&config)
}

func extractDomain(inputURL string) string {
	// Parse the URL
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
	}

	// Get the hostname (including subdomain if present)
	host := parsedURL.Host

	return host
}

func CheckRunning() bool {
	timeout := 2 * time.Second
	_, err := net.DialTimeout("tcp", extractDomain(config.URL), timeout)
	if err != nil {
		log.Println("Site unreachable, error: ", err)
		return false
	}
	return true
}

func fetchToken() string {
	fmt.Println("Fetching token")
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
	if cToken == "" {
		cToken = fetchToken()
	}
	fmt.Println("Checking stats")
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

	return apiResponse.Data.Running, apiResponse.Data.Online
}

func UpdateStats() {
	mcCurrentlyRunning, countOnline = fetchStats()
}

func StopMCServer() error {
	if cToken == "" {
		cToken = fetchToken()
	}
	url := fmt.Sprintf("%sservers/%s/action/stop_server", config.URL, config.ServerID)

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
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

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

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

func StartMCServer() error {
	if cToken == "" {
		cToken = fetchToken()
	}
	url := fmt.Sprintf("%sservers/%s/action/start_server", config.URL, config.ServerID)

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
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

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error with response: %s", resp.Status)
	}

	var apiResponse APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if apiResponse.Status == "ok" {
		fmt.Println("Minecraft server started!")
	}

	return nil
}

func AutoShutdown(id string) {
	fmt.Println("Auto Shutdown Monitoring")
START:
	autoStopFlag := false

	for autoStopFlag == false {
		time.Sleep(10 * time.Second)
		UpdateStats()
		fmt.Printf("Status: %v\nCount: %v\n", mcCurrentlyRunning, countOnline)
		switch {
		case aws.IsAWSInstanceRunning(id) == false:
			autoStopFlag = false
			fmt.Println("AWS instance not running")
		case (mcCurrentlyRunning == true && countOnline == 0) == true:
			autoStopFlag = true
			fmt.Println("Autostop triggered")
		}
	}

	//goland:noinspection GoDfaConstantCondition
	for autoStopFlag == true {
		for i := 0; i < 30; i++ {
			time.Sleep(10 * time.Second)
			UpdateStats()
			if countOnline > 0 == true {
				fmt.Println("Players detected! Resetting.")
				goto START
			}
		}
		autoStopFlag = false
		fmt.Printf("Stopping...\n")
		err := StopMCServer()
		if err != nil {
			log.Printf("Failed to stop server: %v. AWS instance still running.", err)
		} else {
			// Only run if minecraft instance stops
			time.Sleep(10 * time.Second)
			aws.StopAWSInstanceByID(id)
		}
		cToken = ""
	}
}
