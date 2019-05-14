package main

import (
	"bytes"
	"github.com/nextabc-lab/edgex"
	"github.com/yoojia/go-value"
	"io/ioutil"
	"net/http"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func main() {
	edgex.Run(func(ctx edgex.Context) error {
		config := ctx.LoadConfig()
		name := value.Of(config["Name"]).String()
		rpcAddress := value.Of(config["RpcAddress"]).String()
		broadcast := value.Of(config["Broadcast"]).BoolOrDefault(false)

		opts := edgex.EndpointOptions{
			Name:    name,
			RpcAddr: rpcAddress,
		}
		endpoint := ctx.NewEndpoint(opts)

		sockOpts := value.Of(config["HttpClientOptions"]).MustMap()
		targetUrl := value.Of(sockOpts["targetUrl"]).String()
		contentType := value.Of(sockOpts["contentType"]).String()

		client := &http.Client{
			Timeout: value.Of(sockOpts["clientTimeout"]).DurationOfDefault(time.Second * 5),
		}

		endpoint.Serve(func(in edgex.Message) (out edgex.Message) {
			resp, err := client.Post(targetUrl, contentType, bytes.NewReader(in.Bytes()))
			if nil != err {
				ctx.Log().Error("创建Post请求出错: ", err)
				return edgex.NewMessageString("EX=ERR:" + err.Error())
			}
			if broadcast {
				return edgex.NewMessageString("EX=OK:BROADCAST")
			}
			if bs, err := ioutil.ReadAll(resp.Body); nil != err {
				ctx.Log().Error("读取Post请求响应出错: ", err)
				return edgex.NewMessageString("EX=ERR:" + err.Error())
			} else {
				return edgex.NewMessageBytes(bs)
			}
		})

		endpoint.Startup()
		defer endpoint.Shutdown()

		return ctx.TermAwait()
	})
}
