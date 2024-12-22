package assert

import "fmt"

func NotNil(value any, valueName string) {
	if value == nil {
		panic(fmt.Sprintf("%s was nil", valueName))
	}
}

// func Eq(value, mustBe any, valueName string) {
// 	if value != mustBe {
// 		panic(fmt.Sprintf("%v (%s) was not equal to %v", value, valueName, mustBe))
// 	}
// }
