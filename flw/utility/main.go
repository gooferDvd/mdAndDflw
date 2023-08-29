
package main

import (
	"fmt"
	"os"
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"encoding/json"
	"time"

)

type MessageExitQ struct {
	Name string `json:"Name"`
	ExitStatus    int    `json:"ExitStatus"`
	Pipeline	  string  `json:"Pipeline"`
}

func main() {
	/*
	envVar = append(envVar, "QUEUE_FLOW="+p.runName)
	envVar = append(envVar, "CONTAINER_NAME="+containerName)
	envVar = append(envVar,"SERVER_QE="+amqpServerURL)
	*/
	method := "main(): "
	containerName := os.Getenv("CONTAINER_NAME")

	amqpServerURL:="amqp://guest:guest@192.168.56.20:15672/"
	QUEUE := os.Getenv("QUEUE_FLOW")
	for i := 1 ; i<1000; i++ {
		fmt.Println ("hello im a container for the QUEUE"+ QUEUE )
	}
	exitStatus :=0
	connMQ, err := amqp.Dial(amqpServerURL)
	if err != nil {
		fmt.Println(method + "error during the connection to RABBITMQ : "+amqpServerURL )
		return 
	}
	defer connMQ.Close()
	ch,err := connMQ.Channel()
	if err != nil {
		fmt.Println(method + "error opening channel to RabbitMQ")
		return 
	}
	defer ch.Close()
	q,err := ch.QueueDeclare(
		QUEUE, // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		fmt.Println(method + "error declaring queue name  to RabbitMQ")
		return 
	}
	ctxMq, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	body,err := createRmqMessage(containerName,exitStatus,QUEUE )
	if err != nil {
		fmt.Println(method + "error creating json message")
		return  
	}
	err = ch.PublishWithContext(ctxMq,
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing {
		  ContentType: "text/plain",
		  Body:        []byte(body),
		})
	if err != nil {
		fmt.Println(method + "error publishing a message  to RabbitMQ")
		return 
	}
	fmt.Println(method + "message ( "+body+") has been sent "+" to Rabbitmq Queue "+QUEUE)
}

func createRmqMessage (containerName string,exitStatus int, pipeline string) (string,error) {
	method := "CreateRmqMessageJson():"
	queueMsg := make(map[string]interface{})
	queueMsg["Name"]=containerName
	queueMsg["ExitStatus"]=exitStatus
	queueMsg["Pipeline"]=pipeline
	jsonRmqMsg, err := json.Marshal(queueMsg)

	if err != nil {
		fmt.Println(method+ "error while mashalling map to json")
		return "",err
	}
	jsonMessage := string(jsonRmqMsg)
	fmt.Println(method+ "generate Json for message->"+jsonMessage)
	return jsonMessage,nil

}