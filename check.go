package edgex

import "strings"

//
// Author: 陈哈哈 chenyongjia@parkingwang.com, yoojiachen@gmail.com
//

func checkNameFormat(name string) string {
	if "" == name || strings.Contains(name, "/") {
		log.Panic("名称中不能包含'/'字符:" + name)
	}
	return name
}

func checkRequired(value interface{}, message string) {
	switch value.(type) {
	case string:
		if "" == value {
			log.Panic(message)
		}

	case []string:
		if 0 == len(value.([]string)) {
			log.Panic(message)
		}

	default:
		if nil == value {
			log.Panic(message)
		}
	}

}
