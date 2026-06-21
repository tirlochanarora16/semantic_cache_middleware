package helpers

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func GetUserInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Please enter your query")
	fmt.Println("------------------------")

	input, err := reader.ReadString('\n')

	if err != nil {
		return "", err
	}

	fullInput := strings.TrimSpace(input)

	return fullInput, nil

}
