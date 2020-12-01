package main

import (
	"github.com/czh/jwa"
	"log"
	"net/http"
	"time"
)

func main() {

	// 设置URL前缀
	// 比如消息名是echo, prefix为/api，那么POST到/api/echo的请求会调用消息处理函数
	jwa.SetURLPrefix("/api")

	// 所有请求均会调用这个函数进行预处理
	// data可以在回调函数（仅限本次请求）与消息处理函数之间传递数据
	// 返回值代表是否继续处理本次请求（直接返回，而非停止调用回调函数）
	jwa.AddPreProcedureCallback(func(data map[interface{}]interface{}, writer http.ResponseWriter, request *http.Request) bool {

		// 框架仅支持POST请求 建议这里对非POST请求返回错误

		if request.Method != http.MethodPost {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return false
		}

		log.Println("PreCallback 1: Check Method")
		return true
	})

	jwa.AddPreProcedureCallback(func(data map[interface{}]interface{}, writer http.ResponseWriter, request *http.Request) bool {

		data["begin"] = time.Now()
		data["path"] = request.URL.RawPath

		log.Println("PreCallback 2: Log timestamp")
		return true
	})

	jwa.AddPostProcedureCallback(func(data map[interface{}]interface{}, writer http.ResponseWriter, request *http.Request) bool {

		// 如果不显式调用，HTTP框架会进行预测，为了提高性能，这里显示指定MIME
		writer.Header().Set("Content-Type", "application/json")

		log.Println("PostCallback 1: cal elapsed")
		return true
	})

	jwa.AddPostProcedureCallback(func(data map[interface{}]interface{}, writer http.ResponseWriter, request *http.Request) bool {

		if begin, ok := data["begin"].(time.Time); ok {
			diff := time.Now().Sub(begin)
			log.Printf("It takes %v us to process %s", diff.Nanoseconds(), data["path"])

		}

		log.Println("PostCallback 2: cal elapsed")
		return true
	})

	/*
			JWA /api/echo
		2019/04/20 15:14:20 PreCallback 1: Check Method
		2019/04/20 15:14:20 PreCallback 2: Log timestamp
		2019/04/20 15:14:20 PostCallback 1: cal elapsed
		2019/04/20 15:14:20 It takes 95571 us to process
		2019/04/20 15:14:20 PostCallback 2: cal elapsed
	*/

	// 正式服建议设置超时时间等参数
	jwa.ListenAndServeWithServer(&http.Server{
		Addr: ":8080",
	})
}
