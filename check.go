package edgex

import "strings"

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

// checkNameFormat 检查命名规则，不允许带/符号。
func checkNameFormat(key, name string) string {
	if "" == name {
		log.Panic(key + "是必须的")
	}
	if strings.Contains(name, "/") {
		log.Panic(key + "中不能包含'/'字符:" + name)
	}
	return name
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
