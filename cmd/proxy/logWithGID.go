package main

import (
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
)

func logWithGID(message string) {
	buf := make([]byte, 64)
	n := runtime.Stack(buf, false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}

	log.Printf("[GID %d] %s", id, message)
}
