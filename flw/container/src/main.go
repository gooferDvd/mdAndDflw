
package main

import (
	"fmt"
	"os"
	//"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"encoding/json"
	"time"
	"math/rand"
	"strconv"

)

type MessageExitQ struct {
	Name string `json:"Name"`
	ExitStatus    int    `json:"ExitStatus"`
	Pipeline	  string  `json:"Pipeline"`
	ContainerID	  int     `json:"ContainerID"`
}
var (
	containerName string = os.Getenv("CONTAINER_NAME")
	amqpServerURL string =os.Getenv("SERVER_QE")
	QUEUE string = os.Getenv("QUEUE_FLOW")
	containerID = os.Getenv("CONTAINER_ID")
	contid,_ = strconv.Atoi(containerID)
)
func main() {
	method := "main(): "
	err,exitStatus := doJob()
	if err !=nil  {
		fmt.Println(method +"error while doing the job!")
		exitStatus = 2
	}
	err = sendMessage(exitStatus)
	if err != nil {
		fmt.Println(method+ "error while sendmessage ")
		return
	}
	fmt.Println(method+" message is sent")
	return
}
func doJob( ) (error,int) {
	method := "doJob() :"
	for i := 1 ; i<10; i++ {
		fmt.Println (method + "hello im a container for the QUEUE"+ QUEUE )
	}
	
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(8) + 3
	//n:=10
	time.Sleep(time.Duration(n) * time.Second)
	exitStatus :=0
	return nil,exitStatus
}

func sendMessage(exitStatus int) error {
	method := "sendMessage(): "
	connMQ, err := amqp.Dial(amqpServerURL)
	if err != nil {
		fmt.Println(method + "error during the connection to RABBITMQ : "+amqpServerURL +err.Error())
		return err
	}
	defer connMQ.Close()
	ch,err := connMQ.Channel()
	if err != nil {
		fmt.Println(method + "error opening channel to RabbitMQ")
		return err
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
		fmt.Println(method + "error declaring queue name to RabbitMQ: ", err)
		return err
	}
	
	
	body,err := createRmqMessage(containerName,exitStatus,QUEUE,contid)
	if err != nil {
		fmt.Println(method + "error creating json message")
		return  err
	}
	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  /// mandatory
		false,  // immediate
		amqp.Publishing {
		  ContentType: "text/plain",
		  Body:        []byte(body),
		})
	if err != nil {
		fmt.Println(method + "error publishing a message  to RabbitMQ")
		return err
	}
	fmt.Println(method + "message ( "+body+") has been sent "+" to Rabbitmq Queue "+QUEUE)
	return nil
}

func createRmqMessage (containerName string,exitStatus int, pipeline string, contid int) (string,error) {
	method := "CreateRmqMessageJson():"
	queueMsg := make(map[string]interface{})
	queueMsg["Name"]=containerName
	queueMsg["ExitStatus"]=exitStatus
	queueMsg["Pipeline"]=pipeline
	queueMsg["ContainerID"]=contid
	jsonRmqMsg, err := json.Marshal(queueMsg)

	if err != nil {
		fmt.Println(method+ "error while mashalling map to json")
		return "",err
	}
	jsonMessage := string(jsonRmqMsg)
	fmt.Println(method+ "generate Json for message->"+jsonMessage)
	return jsonMessage,nil
}
