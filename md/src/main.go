package main

import (
	"net/http"
	"os"
	"strconv"

	eurekaclient "example.com/eurekaclient"
	httpdm "example.com/httpdm"
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
	port, err := strconv.Atoi(os.Getenv("DOCKER_MANAGER_PORT"))
	if err != nil {
		logdm.WriteLogLine(method + "eureka port in not valid")
	}
	logdm.WriteLogLine(method + "---------------Docker Manager Start---------------")
	defer logdm.WriteLogLine(method + "---------------Docker Manager Stopo---------------")
	logdm.WriteLogLine(method + "eureka url: " + os.Getenv("EUREKA_ZONE"))
	logdm.WriteLogLine(method + "eureka port: " + strconv.Itoa(port))
	eurekaclient.EurekaClient(os.Getenv("EUREKA_ZONE"), port)
	r := mux.NewRouter()
	r.HandleFunc("/list", httpdm.GetAllPermitted)
	r.HandleFunc("/restart", httpdm.RestartContainer)
	r.HandleFunc("/log", httpdm.GetLogByNameLines)
	r.HandleFunc("/searchbyname", httpdm.SearchByName)
	r.HandleFunc("/listbyimage", httpdm.ListByImage)
	r.HandleFunc("/stopit", httpdm.StopIt)
	r.HandleFunc("/pruneitbyimage", httpdm.RemoveExitByImage)
	
	r.HandleFunc("/stopallcontainerbyimage", httpdm.StopAllContainerByImage)
	r.HandleFunc("/exited", httpdm.GetExited)
	r.HandleFunc("/create", httpdm.CreateContainer)
	r.HandleFunc("/volumeprunes", httpdm.VolumePrunes)
	r.HandleFunc("/volumeslist", httpdm.VolumeList)
	r.HandleFunc("/volumecreate", httpdm.VolumeCreate)
	r.HandleFunc("/reloadenv", httpdm.ReloadEnv)
	r.HandleFunc("/printenv", httpdm.PrintEnv)
	r.HandleFunc("/getvolumeinstate", httpdm.Getvolumeinstate)
	r.HandleFunc("/getlistcontainerbind", httpdm.GetListContainerBind)
	r.HandleFunc("/getbinds", httpdm.GetBinds)
	r.HandleFunc("/getbindsrfl", httpdm.GetBindsRfl)
	r.HandleFunc("/killit", httpdm.KillIt)
	r.HandleFunc("/deletevolume", httpdm.DeleteVolume)
	r.HandleFunc("/prune", httpdm.Prune)
	r.HandleFunc("/v3/api-docs", httpdm.Swagger)
	http.Handle("/", r)
	logdm.WriteLogLine(method + "Starting docker manager server at port: " + strconv.Itoa(port))
	if err := http.ListenAndServe(":"+os.Getenv("DOCKER_MANAGER_PORT"), nil); err != nil {
		logdm.WriteLogLine(method + " error opening the dockermanager port " + strconv.Itoa(port))
	}

}
