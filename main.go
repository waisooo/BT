package main

import (
	bencode "bittorrent/decode"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Missing parameter: please provide a file name!")
		return
	}

	filePath := os.Args[1]

	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	val, index, err := bencode.Decode(data)

	fmt.Println(val, index, err)
}
