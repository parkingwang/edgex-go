package edgex

import "strings"

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// checkIdFormat 检查命名规则，不允许带/符号。
func checkIdFormat(key, id string) string {
	if "" == id {
		log.Panic(key + "是必须的")
	}
	if strings.Contains(id, "/") || strings.Contains(id, ":") {
		log.Panic(key + "中不能包含 '/' 或 ':' 字符:" + id)
	}
	return id
}

// checkRequires 检查配置值是否有效；无效则Panic；
func checkRequires(value interface{}, message string) {
	if nil == value {
		log.Panic(message)
	}
	switch value.(type) {
	case string:
		if "" == value {
			log.Panic(message)
		}

	case []string:
		if 0 == len(value.([]string)) {
			log.Panic(message)
		}
	}

}
