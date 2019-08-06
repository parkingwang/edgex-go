package main

import (
	"fmt"
	"github.com/nextabc-lab/edgex-go"
	"time"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func main() {
	edgex.Run(func(ctx edgex.Context) error {

		opts := edgex.DriverOptions{
			EventTopics: []string{
				"example/+",
				"scheduled/+",
			},
			CustomTopics: []string{
				edgex.TopicSubscribeNodesInspect,
			},
		}

		const testEndpointNodeId = "DEV-ENDPOINT"

		ctx.Initial("DEV-DRIVER")
		driver := ctx.NewDriver(opts)

		log := ctx.Log()

		driver.Process(func(topic string, inMsg edgex.Message) {
			//execStart := time.Now()
			//_, err := driver.Execute(testEndpointNodeId, inMsg, time.Second*3)
			//if nil != err {
			//	log.Error("Execute发生错误: ", err)
			//} else {
			//	log.Debug("Execute用时: ", time.Since(execStart))
			//}
		})

		log.Debugf("创建Driver节点: [%s]", ctx.NodeId())

		driver.Startup()
		defer driver.Shutdown()

		timer := time.NewTicker(time.Second)
		for {
			select {
			case <-timer.C:
				log.Debug("----> Send Exec")
				execStart := time.Now()
				msg := driver.NextMessageBy(ctx.NodeId(), []byte(fmt.Sprintf("%v", time.Now().UnixNano())))
				rep, err := driver.Execute(testEndpointNodeId, msg, time.Second)
				if nil != err {
					log.Error("ScheduleExecute发生错误: ", err)
				} else {
					log.Debug("ScheduleExecute用时: ", time.Since(execStart))
					sid := rep.VirtualNodeId()
					body := string(rep.Body())
					log.Debug("接收到响应：VirtualNodeId: " + sid + ", Body: " + body)
				}

			case <-ctx.TermChan():
				log.Debug("接收到系统终止信号")
				timer.Stop()
				return nil
			}
		}

	})
}
