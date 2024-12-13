package crafty

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"onemc/internal/aws"
	"onemc/internal/utils"
	"time"
)

var (
	config             utils.Config
	cToken             string
	mcCurrentlyRunning bool
	Running            = &mcCurrentlyRunning
	countOnline        int
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
			Online  int    `json:"countOnline"`
			Players string `json:"players"`
			Running bool   `json:"mcCurrentlyRunning"`
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

func StopQuery() bool {
	UpdateStats()
	if !mcCurrentlyRunning {
		log.Println("Minecraft server isn't mcCurrentlyRunning")
	}
	if countOnline > 0 {
		log.Printf("%v player(s) are currently countOnline", countOnline)
		return false
	}
	return true
}

func StartMCServer() error {
	cToken = fetchToken()
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
		fmt.Print("Minecraft server started!")
	}

	return nil
}

func AutoShutdown(id string) {
	autoStopFlag := false

	for autoStopFlag == false {
		time.Sleep(30 * time.Second)
		UpdateStats()
		switch {
		case aws.IsAWSInstanceRunning(id) == false:
			autoStopFlag = false
		case (mcCurrentlyRunning == true && countOnline == 0) == true:
			autoStopFlag = true
		}
	}

	//goland:noinspection GoDfaConstantCondition
	for autoStopFlag == true {
		var elapsed bool
		timer := time.NewTimer(10 * time.Minute)
		go func() {
			<-timer.C
			elapsed = true
		}()

		for elapsed == false {
			time.Sleep(10 * time.Second)
			if (mcCurrentlyRunning && countOnline > 0) == false {
				autoStopFlag = false
				timer.Stop()
				fmt.Printf("Elapsed: %v\n", elapsed)
				autoStopFlag = true
				// TODO: Finish reset clause
			}
		}
		<-timer.C
		time.Sleep(1 * time.Second)
		fmt.Printf("Timer elapsed. Stopping...\n")
		err := StopMCServer()
		if err != nil {
			log.Printf("Failed to stop server: %v", err)
		}
		// TODO: Finish autostop. Mainly aws stuff. Handle cases where server isn't stopping.
	}
}
