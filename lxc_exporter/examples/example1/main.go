package main

import (
	"fmt"
	"regexp"
)

func main() {
	text := "Hello   World"
	re := regexp.MustCompile("\\s\\s+")
	matched := re.FindString(text)
	fmt.Printf("Matched:%vwd\n", matched) // 输出: Matched:
}
