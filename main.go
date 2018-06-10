package main

import (
	"math/rand"
	"flag"
	"fmt"
	"strings"

	"github.com/lukaspj/go-chat/chat"

	"github.com/lukaspj/go-logging/logging"
	"bufio"
	"os"
	"time"
)

var logger = logging.GetLogger()

func main() {
	rand.Seed(time.Now().UnixNano())

	port1 := flag.Int("sp", (rand.Int() % 1000) + 4300, "Source port")
	dest := flag.String("dest", "", "Destination address")

	flag.Parse()

	logger.SetLevel(logging.INFO)
	logger.AddLogOutput(logging.GetFileOutput(fmt.Sprintf("chat_%v_log.log", *port1)))
	logger.Info("Logging initialized")

	port2 := *port1 + 1000

	chatClient := chat.NewChatClient(*port1, port2)

	if *dest == "" {
		chatClient.StartRoom()
	} else {
		chatClient.Connect(*dest)
	}

	go func() {
		defer chatClient.Close()
		var input string
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Write lines here: ")
		for {
			input, _ = reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if strings.HasPrefix(input, "/") {
				if input == "/exit" {
					break
				}
			} else {
				chatClient.SendMessage(input)
			}
		}
	}()


	<-make(chan struct{})
}