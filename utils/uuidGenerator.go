package utils

import (
	"fmt"
	"time"
)

func Uid() string{
	return fmt.Sprintf("%d",time.Now().UnixNano())
}