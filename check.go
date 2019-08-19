package edgex

import "strings"

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// checkIDFormat 检查命名规则，不允许带/符号。
func checkIDFormat(id, keyName string) string {
	if strings.Contains(id, "/") || strings.Contains(id, ":") {
		log.Panic(keyName + "中不能包含 '/' 或 ':' 字符:" + id)
	}
	return id
}

// checkRequired 检查配置值是否有效；无效则Panic；
func checkRequired(value, message string) string {
	if "" == value {
		log.Panic(message)
	}
	return value
}

func checkRequiredId(id, keyName string) string {
	checkRequired(id, keyName+"是必须的参数")
	return checkIDFormat(id, keyName)
}
