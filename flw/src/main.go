
package main

import (
	"net/http"
	"os"
	"strconv"

	eurekaclient "example.com/eurekaclient"
	httpdm "example.com/httpdm"
	//httpfl "example.com/httpfl"
	logdm "example.com/logdm"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

//var Userclient *http.Client = nil

// @title Swagger Example API
// @version 1.0
// @description This is a sample server Petstore server.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host petstore.swagger.io
// @BasePath /v2
func main() {
	method := "main():"
	port, err := strconv.Atoi(os.Getenv("DOCKER_FLOW_PORT"))
	if err != nil {
		logdm.WriteLogLine(method + "eureka port in not valid")
	}
	logdm.WriteLogLine(method + "---------------Docker Manager Start---------------")
	defer logdm.WriteLogLine(method + "---------------Docker Manager Stopo---------------")
	logdm.WriteLogLine(method + "eureka url: " + os.Getenv("EUREKA_ZONE"))
	logdm.WriteLogLine(method + "eureka port: " + strconv.Itoa(port))
	eurekaclient.EurekaClient(os.Getenv("EUREKA_ZONE"), port)
	r := mux.NewRouter()
	r.HandleFunc("/v3/api-docs", httpdm.Swagger)
	//docker-flow
	r.HandleFunc("/addstep", httpdm.Addstepipeline)
	r.HandleFunc("/prova", httpdm.Prova)
	r.HandleFunc("/run", httpdm.RunPipeline)
	http.Handle("/", r)
	logdm.WriteLogLine(method + "Starting docker manager server at port: " + strconv.Itoa(port))
	if err := http.ListenAndServe(":"+os.Getenv("DOCKER_FLOW_PORT"), nil); err != nil {
		logdm.WriteLogLine(method + " error opening the dockermanager port " + strconv.Itoa(port))
	}

}
