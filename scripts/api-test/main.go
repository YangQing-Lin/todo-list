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
	baseURL := "http://localhost:7789"

	// 等待服务器启动
	time.Sleep(2 * time.Second)

	fmt.Println("=== Go Todo List API 测试 ===\n")

	// 测试1: 健康检查
	fmt.Println("1. 测试健康检查端点 /")
	TestEndpoint(baseURL, "GET", "/", nil)

	// 测试2: 获取空todo列表
	fmt.Println("\n2. 测试获取待办事项列表 /api/todos")
	TestEndpoint(baseURL, "GET", "/api/todos", nil)

	// 测试3: 创建新的todo
	fmt.Println("\n3. 测试创建新的待办事项")
	todoData := map[string]string{
		"title":       "学习Go语言",
		"description": "完成第一个Go项目",
	}
	jsonData, _ := json.Marshal(todoData)
	TestEndpoint(baseURL, "POST", "/api/todos", jsonData)

	// 测试4: 再次获取todo列表
	fmt.Println("\n4. 再次获取待办事项列表")
	TestEndpoint(baseURL, "GET", "/api/todos", nil)

	fmt.Println("\n=== 测试完成 ===")
}

func TestEndpoint(baseURL, method, endpoint string, data []byte) {
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
	fmt.Printf("Response: %s\n", string(body))
}