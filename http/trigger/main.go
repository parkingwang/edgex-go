package main

import (
	"github.com/nextabc-lab/edgex"
	"github.com/yoojia/go-value"
	"io/ioutil"
	"net/http"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
// 使用HttpServer接收POST方式发送的原始数据，并推送到MQTT服务器。

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		config := ctx.LoadConfig()
		name := value.Of(config["Name"]).String()
		topic := value.Of(config["Topic"]).String()

		trigger := ctx.NewTrigger(edgex.TriggerOptions{
			Name:  name,
			Topic: topic,
		})

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if "POST" == r.Method {
				if data, err := ioutil.ReadAll(r.Body); nil != err {
					ctx.Log().Error("读取HttpBody出错: ", err)
				} else {
					if err := trigger.Triggered(edgex.NewMessage([]byte(name), data)); nil != err {
						ctx.Log().Error("触发事件出错: ", err)
					}
				}
			} else {
				w.WriteHeader(403)
				if _, err := w.Write([]byte("Method now allowed")); nil != err {
					ctx.Log().Error("返回Http响应出错: ", err)
				}
			}
		})

		opts := value.Of(config["HttpServerOptions"]).MustMap()
		serverAddr := value.Of(opts["listenAddress"]).String()
		server := &http.Server{
			Addr:           serverAddr,
			Handler:        handler,
			ReadTimeout:    value.Of(opts["readTimeout"]).DurationOfDefault(3 * time.Second),
			WriteTimeout:   value.Of(opts["writeTimeout"]).DurationOfDefault(3 * time.Second),
			MaxHeaderBytes: 1 << 20,
		}

		// 启用Trigger服务
		trigger.Startup()
		defer trigger.Shutdown()

		ctx.Log().Debug("开启Http服务端: ", serverAddr)
		defer ctx.Log().Debug("停止Http服务端")
		return server.ListenAndServe()
	})
}
