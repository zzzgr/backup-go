package model

import (
	"encoding/json"
	"net/http"
	"time"
)

// Response API通用响应结构
type Response struct {
	Code int         `json:"code"` // 状态码，200表示成功
	Msg  string      `json:"msg"`  // 消息
	Data interface{} `json:"data"` // 数据
	T    int64       `json:"t"`    // 时间戳
}

// NewResponse 创建响应
func NewResponse(code int, msg string, data interface{}) *Response {
	return &Response{
		Code: code,
		Msg:  msg,
		Data: data,
		T:    time.Now().UnixMilli(),
	}
}

// Success 成功响应
func Success(data interface{}) *Response {
	return NewResponse(200, "success", data)
}

// Error 错误响应
func Error(code int, msg string) *Response {
	return NewResponse(code, msg, nil)
}

// SuccessResponse 简化的成功响应
func SuccessResponse(data interface{}) *Response {
	return Success(data)
}

// FailResponse 简化的失败响应
func FailResponse(msg string) *Response {
	return Error(400, msg)
}

// ErrorWithCode 指定代码的错误响应
func ErrorWithCode(code int, msg string) *Response {
	return Error(code, msg)
}

// SuccessWithMsg 带消息的成功响应
func SuccessWithMsg(data interface{}, msg string) *Response {
	return &Response{
		Code: 200,
		Msg:  msg,
		Data: data,
		T:    time.Now().UnixMilli(),
	}
}

// WriteJSON 将结构体转换为JSON输出到响应
func WriteJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
