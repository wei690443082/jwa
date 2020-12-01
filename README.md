jwa是JSON Web API的缩写，一个简单的HTTP POST收发JSON消息的框架。

# 快速入门
```
czh@mbp18:~$ go get github.com/czh/jwa

czh@mbp18:~$ mkdir demo
czh@mbp18:~$ cd demo/
czh@mbp18:~/demo$ cp $GOPATH/src/github.com/czh/jwa/example/AddNewMessage.sh .

czh@mbp18:~/demo$ ./AddNewMessage.sh echo

czh@mbp18:~/demo$ vim main.go
czh@mbp18:~/demo$ cat main.go
package main

import "github.com/czh/jwa"
import "net/http"

func main() {

	jwa.ListenAndServeWithServer(&http.Server{
		Addr: ":8080",
	})

}

czh@mbp18:~/demo$ go build
czh@mbp18:~/demo$ ./demo
JWA /echo

```

1. 首先克隆库到GOPATH源码目录中，这里直接放到src/目录下
2. 然后拷贝消息生成脚本，执行时带上消息名作为参数，生成模板
3. 编辑模板内的代码（这里不再展示）
4. 编写main函数启动程序
5. 编译程序并执行

# 消息编写
以上述生成文件为例，打开这个文件，可以看到代码如下

```
package main

import (
	"encoding/json"
    "github.com/czh/jwa"
    "net/http"
)

type echoRequest struct {
	Msg string
}

type echoReply struct {
	Msg string
}

func init() {
	jwa.AddMessageHandler("echo", echoHandler)
}

func echoHandler(data map[interface{}]interface{}, payload []byte) (resp interface{}) {

	var request echoRequest

	if err := json.Unmarshal(payload, &request); err != nil {
		return http.StatusBadRequest
	}

	reply := &echoReply{ }

	// write code following here now !

	return reply
}
```

服务器收到客户端发送的JSON消息后，会将其转换成Go的结构体`request`。使用者只需要基于这个`request`进行业务处理并完成对`reply`进行赋值即可。最终reply会返回给客户端（默认200）。

几点建议：
- 一个文件只实现一条协议。如果多个协议属于模块A，建议创建A.go实现模块的函数，然后在协议文件内调用A.go的函数。
- xxRequest/xxReply结构体内的结构体成员可以使用大小字母直接将其作为默认的JSON Key值，避免json tag来指定同名小写Key值，减少错误拼写导致Bug的同时，方便引入其它模块的结构体进来作为回复。
- 不建议修改协议生成文件内的通用代码。

```
czh@mbp18:~/demo$ curl -vvv  localhost:8080/echo -d '{}'
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 8080 (#0)
> POST /echo HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/7.54.0
> Accept: */*
> Content-Length: 2
> Content-Type: application/x-www-form-urlencoded
>
* upload completely sent off: 2 out of 2 bytes
< HTTP/1.1 200 OK
< Date: Sat, 20 Apr 2019 06:17:31 GMT
< Content-Length: 10
< Content-Type: text/plain; charset=utf-8
<
* Connection #0 to host localhost left intact
{"Msg":""}
```

也许你已经注意到了，在回复的HTTP Header中，`Content-Type`的值并非`application/json`. 为了更准备地描述MIME，同时[提高性能](https://golang.org/src/net/http/server.go?s=2977:5840#L118)，我们可以在jwa提供的消息预处理回调函数中显示的指定我们消息回复的类型:

```
jwa.AddPostProcedureCallback(func(data map[interface{}]interface{}, writer http.ResponseWriter, request *http.Request) bool {

		// 如果不显式调用，HTTP框架会进行预测，为了提高性能，这里显示指定MIME
		writer.Header().Set("Content-Type", "application/json")

		return true
	})
```

这个消息预处理函数可以操作`data`以便在使用者所写代码执行期间保存当前请求的上下文信息。我们可以看到`echo`消息的处理函数，第一个参数就是这个`data`

消息处理函数是可以将整形作为返回值的，此时，这个整形值会作为HTTP的Response Code写入Header并结束请求。通常用于错误返回，比如这里解析协议失败时，返回BadRequest错误。大多数情况下，这个消息应当返回`reply`（自定义消息结构体）。

为了避免重复键入消息名，这里不建议使用json tag的方式指定JSON的字段名（通常是全小写）。而应直接使用Go的特性：大写字母开头的结构体成员作为“public成员”，在序列化为JSON时才生效。以此降低typo错误导致的时间浪费。

HTTP消息处理函数一定要返回一个结果作为HTTP请求的响应。同时消息处理函数返回后，框架回立马返回结果。因此在消息处理函数中使用go启动新的goroutine时要再三思考额外注意：它只能启动背景或后台任务，不能延迟返回结果。如果需要延迟返回结果，请配合非buffered chan或者WaitGroup来实现。

# 请求路径
假如我们的服务器域名为api.com，我们注册了echo消息，那么默认情况下，访问`api.com/echo`才会触发消息处理。框架提供了`jwa.SetURLPrefix("/v1")
`函数，可以实现设置消息前缀的功能。比如前边例子中，访问`api.com/v1/echo`才会触发消息处理。

# 消息注册选项
如果某个消息只能由内部系统调用，那么可以采取两个方案：
- 一个单独的http.Server监听另外的端口并在网关设置防火墙规则
- 限制消息的发起者IP地址(socket层的peername)

这里我们采取后者，因为jwa目前暂不支持消息与端口的绑定关系。在上述例子中，jwa使用`jwa.AddMessageHandler("echo", echoHandler)`的方式来注册消息，并提供`jwa.AddPreProcedureCallback`来注册全局消息回调函数。在内部实现中，其实每个消息的回调函数是独立的，全局注册的回调函数会在初始化消息时候拷贝进来。为了提升扩展能力，jwa提供了扩展消息注册函数`AddMessageHandlerExt`：

```
type ProcedureCallbackFunc func(data map[interface{}]interface{}, writer http.ResponseWriter, request *http.Request) bool

type HandlerOptions struct {
	ClearPreCallbacks  bool // 消息不再调用全局预处理回调函数
	ClearPostCallbacks bool // 消息不再调用全局后处理回调函数

	preCallbacks  []ProcedureCallbackFunc
	postCallbacks []ProcedureCallbackFunc
}

func (options *HandlerOptions) AddPreProcedureCallback(cb ProcedureCallbackFunc) {

	options.preCallbacks = append(options.preCallbacks, cb)
}

func (options *HandlerOptions) AddPostProcedureCallback(cb ProcedureCallbackFunc) {

	options.postCallbacks = append(options.postCallbacks, cb)
}


func AddMessageHandlerExt(name string, procedure ProcedureFunc, options HandlerOptions) {
// 省略实现 ...
}
```
开发者可以在注册消息的时候，指定一些消息选项。目前有`ClearPreCallbacks`与`ClearPostCallbacks`两个。如果程序内通过jwa注册的全局预（后）处理函数不想对某消息生效，那么在注册这个消息的时候，可以设置这两个字段为真。同时，消息可以有独立的回调函数，仅在本消息调用（如果复用options，请额外注意）。

框架在实现的时候，仅设置`HandlerOptions`内的非零值字段，所以开发者只需要设置想要的字段即可，无需关系其它字段的内容。

下面是一个限制IP的echo消息实现：

```
package main
import (...)

type echoRequest struct {
}

type echoReply struct {
}

func internalPeernameCheck(data map[interface{}]interface{}, writer http.ResponseWriter, request *http.Request) bool {

	var pass bool
	// 这里我假设了IPv4地址，生产环境请认真编写这部分代码！
	// 考虑使用Go官方库的 net.SplitHostPort
	ip := strings.Split(request.RemoteAddr, ":")[0]
	if ip == "127.0.0.1" {
		pass = true
	} else {
		writer.WriteHeader(http.StatusForbidden)
	}

	log.Println(pass, ip, request.RequestURI)

	return pass
}

func init() {

	options := jwa.HandlerOptions{}
	options.AddPreProcedureCallback(internalPeernameCheck)

	jwa.AddMessageHandlerExt("echo", echoHandler, options)
}

func echoHandler(data map[interface{}]interface{}, payload []byte) (resp interface{}) {
    ...
}
```

```
czh@mbp18:~/demo$ curl localhost:8080/echo -d '{"Token": "abcd"}'
{"Result":0,"Description":"OK"}

czh@mbp18:~/demo$ curl -v 192.168.3.15:8080/echo -d '{"Token": "abcd"}'
*   Trying 192.168.3.15...
* TCP_NODELAY set
* Connected to 192.168.3.15 (192.168.3.15) port 8080 (#0)
> POST /echo HTTP/1.1
> Host: 192.168.3.15:8080
> User-Agent: curl/7.54.0
> Accept: */*
> Content-Length: 17
> Content-Type: application/x-www-form-urlencoded
>
* upload completely sent off: 17 out of 17 bytes
< HTTP/1.1 403 Forbidden
< Date: Mon, 22 Oct 2018 09:57:27 GMT
< Content-Length: 0
<
* Connection #0 to host 192.168.3.15 left intact
```

通常会不止有一个接口需要进行限制，这时可把`internalPeernameCheck`放在其它文件内。同时建议在消息层面增加Token验证。

# data[]
jwa在内部使用三个变量来记录消息处理的状态:
```
data["_response_code"] = http.StatusOK // read & set
data["_response_body"] = []byte("")    // readonly
data["_request_body"] = .....          // readonly
```
使用者可以在消息处理函数中设置`data["_response_code"]`来修改默认的HTTP返回值，比如返回201来表示Created成功）。如果想实现调试时的日志记录功能，通常会用到这两个变量：
```
if debug {

	jwa.AddPreProcedureCallback(func(data map[interface{}]interface{}, w http.ResponseWriter, r *http.Request) bool {

		var buf bytes.Buffer
		buf.WriteString("\n-------------------------------------\n")
		buf.WriteString(fmt.Sprintf("* Requested From %s\n", r.RemoteAddr))

		buf.WriteString(fmt.Sprintf("> %s %v %s\n", r.Method, r.URL, r.Proto))
		buf.WriteString(fmt.Sprintf("> Host %s\n", r.Host))

		for k, v := range r.Header {
			buf.WriteString(fmt.Sprintf("> %s: %s\n", k, v))
		}

        buf.WriteString(fmt.Sprintf(">%s\n", string(data["_request_body"].([]byte))))

		data["debug.verbose"] = buf

		return true
	})

	jwa.AddPostProcedureCallback(func(data map[interface{}]interface{}, w http.ResponseWriter, r *http.Request) bool {

		buf := data["debug.verbose"].(bytes.Buffer)
		code := data["_response_code"].(int)

		buf.WriteString(">\n")

		buf.WriteString(fmt.Sprintf("< %s %d %s\n", r.Proto, code, http.StatusText(code)))

		for k, v := range w.Header() {
			buf.WriteString(fmt.Sprintf("< %s: %s\n", k, v))
		}

		response := data["_response_body"].([]byte)

		buf.WriteString(fmt.Sprintf("< %s\n", string(response)))

		log.Print(buf.String())

		return true
	})
}
```
```
2019/04/22 14:49:46.595074 main.go:155:
-------------------------------------
* Requested From 127.0.0.1:62886
> POST /api/echo HTTP/1.1
> Host localhost:8080
> Accept: [*/*]
> Content-Length: [2]
> Content-Type: [application/x-www-form-urlencoded]
> User-Agent: [curl/7.54.0]
>
< HTTP/1.1 201 Created
< {"Msg":""}
```

由于这个特性是我在写项目临时需要用到的，所以实现上基本没怎么会时间思考，错在缺陷。比如这里返回消息的Header并不是真实消息会返回的。同时因为jwa不是RESTful风格API的框架，因此我也在思考jwa怎么设计会更合理一些。


# ServeMux
jwa暴露了内部使用的http.ServeMux，以便开发者能够实现基于jwa开发时无法完成的事情。比如配合第三方系统的HTTP回调（注意，jwa与http.ServeMux默认情况下是**不会**根据HTTP Method匹配回调函数的，回调函数的匹配对象是请求路径），由于发过来的消息在序列化时可能报错并由框架直接回复，因此需要额外的机制来处理。

```
jwa.ServeMux.Handle(...)
```

# 架设建议
由于jwa只提供了POST处理，相当于一个HTTP动态请求处理器，因此强烈不建议将它作为静态文件服务器来使用，比如返回一张图片。同时因为缺乏对GET请求的支持，也无法在API开发中使用GET方法。

因此对于这种情况，建议在Nginx层面设置反向代理来分发请求，并由Nginx直接处理静态文件。同时，也建议在服务器前SSL offloading，比如同样交给Nginx来做。部分CDN或者负载均衡也提供了类似服务。

对于静态资源文件，建议使用阿里云OSS或其它对象存储服务，或在Nginx层面配置直接处理。

# Tips
* 可以在`jwa.AddPreProcedureCallback`中获取客户端的真实IP地址并放入data中。客户端的真实IP基于部署方式来获取，比如在CDN环境中，可通过`X-Forwarded-For`来取得(注意安全性，参考各家CDN文档定义的客户端源IP头)。
* 建议在启动函数`jwa.ListenAndServeWithServer`中，配置HTTP Server的超时等函数，以便满足生产环境的需要。
* 在框架example目录下有更详细的代码实例可以参考
* API中的token可使用JWT或者UUID配合Redis来实现
* 下面代码是我在实现项目时用到的，供参考
```
jwa.AddPreProcedureCallback(func(data map[interface{}]interface{}, writer http.ResponseWriter, request *http.Request) bool {

	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return false
	}

	SID := request.Header.Get("SessionID")
	data["SessionID"] = SID

	return true
})

jwa.AddPostProcedureCallback(func(data map[interface{}]interface{}, writer http.ResponseWriter, request *http.Request) bool {

	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Cache-Control", "no-store")

	return true
})

```

# 性能测试

```
ab -k -n 1024000 -c 32 -p ./content.txt  http://127.0.0.1:8080/echo

Server Software:
Server Hostname:        127.0.0.1
Server Port:            8080

Document Path:          /echo
Document Length:        31 bytes

Concurrency Level:      16
Time taken for tests:   13.495 seconds
Complete requests:      1024000
Failed requests:        0
Keep-Alive requests:    1024000
Total transferred:      176128000 bytes
Total body sent:        162816000
HTML transferred:       31744000 bytes
Requests per second:    75881.55 [#/sec] (mean)
Time per request:       0.211 [ms] (mean)
Time per request:       0.013 [ms] (mean, across all concurrent requests)
Transfer rate:          12745.73 [Kbytes/sec] received
                        11782.39 kb/s sent
                        24528.12 kb/s total

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.0      0       1
Processing:     0    0   0.1      0       9
Waiting:        0    0   0.1      0       9
Total:          0    0   0.1      0       9

Percentage of the requests served within a certain time (ms)
  50%      0
  66%      0
  75%      0
  80%      0
  90%      0
  95%      0
  98%      0
  99%      0
 100%      9 (longest request)
```
