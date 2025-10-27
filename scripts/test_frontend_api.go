package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	// 测试后端API
	baseURL := "http://localhost:7789"

	fmt.Println("=== 测试后端API ===\n")

	// 1. 健康检查
	testEndpoint(baseURL, "GET", "/", nil)

	// 2. 获取空列表
	testEndpoint(baseURL, "GET", "/api/todos", nil)

	// 3. 创建Todo
	todoData := map[string]string{
		"title": "测试Todo",
		"description": "这是一个测试Todo项目",
	}
	jsonData, _ := json.Marshal(todoData)
	testEndpoint(baseURL, "POST", "/api/todos", jsonData)

	// 4. 再次获取列表
	testEndpoint(baseURL, "GET", "/api/todos", nil)

	fmt.Println("\n=== 测试前端代理 ===\n")

	// 测试前端代理到后端
	frontendBaseURL := "http://localhost:3000"
	testEndpoint(frontendBaseURL, "GET", "/api/todos", nil)
}

func testEndpoint(baseURL, method, endpoint string, data []byte) {
	var req *http.Request
	var err error

	url := baseURL + endpoint

	if data != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		fmt.Printf("❌ 创建请求失败: %v\n", err)
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("✅ %s %s - Status: %d\n", method, endpoint, resp.StatusCode)
	if len(body) < 500 {
		fmt.Printf("Response: %s\n", string(body))
	} else {
		fmt.Printf("Response: [Response too large: %d bytes]\n", len(body))
	}
	fmt.Println("---")
}