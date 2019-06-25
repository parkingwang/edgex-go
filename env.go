package edgex

import (
	"os"
	"strconv"
)

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

func EnvGetString(key, defValue string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	} else {
		return defValue
	}
}

func EnvGetInt64(key string, defValue int64) int64 {
	if v, ok := os.LookupEnv(key); ok {
		if iv, err := strconv.ParseInt(v, 10, 64); nil != err {
			log.Fatal("Invalid value to Int64: " + v)
			return 0
		} else {
			return iv
		}
	} else {
		return defValue
	}
}

func EnvGetBoolean(key string, defValue bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		return "true" == v
	} else {
		return defValue
	}
}
