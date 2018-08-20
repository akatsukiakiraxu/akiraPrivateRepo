package main

import (
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
)

// Sangoを使ってるならダッシュボード、自分で立てたならそのIPです
const MQTT_BROKER = "192.168.61.135:1883"

func createMQTTClient(brokerAddr, clientId, username, password string) MQTT.Client {
	opts := MQTT.NewClientOptions().AddBroker(brokerAddr)
	opts.SetClientID(clientId)
	opts.SetUsername(username)
	opts.SetPassword(password)
	client := MQTT.NewClient(opts)
	return client
}

func subscribe(client MQTT.Client, sub chan<- MQTT.Message) {
	fmt.Println("start subscribing...")
	// forループしなくても勝手に中でループしててくれます。なんかそういうのってあんまりGolangっぽくない気がしますけど。
	subToken := client.Subscribe(
		"test", // Sangoのダッシュボードからコピペしましょう
		0,
		func(client MQTT.Client, msg MQTT.Message) {
			sub <- msg
		})
	if subToken.Wait() && subToken.Error() != nil {
		fmt.Println(subToken.Error())
		os.Exit(1)
	}
}

func publish(client MQTT.Client, input string) {
	token := client.Publish("test", 0, true, input)
	if token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
}
func input(pub chan<- string) {
	for {
		var input string
		fmt.Scanln(&input)
		pub <- input
	}
}

func main() {
	/*
		m := map[string][]byte{"apple": {0, 1, 2, 44, 44, 4}, "banana": {1, 2, 4}, "lem    on": {5, 6, 7}}
		var a uint64
		a = 1
		fmt.Println(len(m))
		fmt.Println(len(m["apple"]))
		fmt.Println(unsafe.Sizeof(a))
		const MaxUint = ^uint16(0)
		fmt.Println(MaxUint)
		ch := "ch1"
		chnum, _ := strconv.ParseUint(strings.Replace(ch, "ch", "0", 1), 10, 32)
		fmt.Println(chnum)
	*/
	log.Println("====test mqtt=====")
	fmt.Print("your id: ")
	var id string
	fmt.Scanln(&id)
	client := createMQTTClient(MQTT_BROKER, id, "b", "a")
	defer client.Disconnect(250)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	sub := make(chan MQTT.Message)
	go subscribe(client, sub)
	pub := make(chan string)
	go input(pub)
	for {
		select {
		case s := <-sub:
			msg := string(s.Payload())
			fmt.Printf("\nmsg: %s\n", msg)
		case p := <-pub:
			publish(client, p)
		}
	}
}
