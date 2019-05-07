package edgex

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

type Pipeline interface {

	Incomes() <-chan Frame

	Forward(req Frame)
}