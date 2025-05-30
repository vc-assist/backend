package assert

func NotNil(value any) {
	if value == nil {
		panic("expected value to be not nil")
	}
}

func NotEmptyStr(str string) {
	if str == "" {
		panic("expected string to be non-empty")
	}
}
