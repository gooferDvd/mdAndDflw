package httpdm

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	keycloak "example.com/keycloak"
	logdm "example.com/logdm"
	utils "example.com/utils"
)

// godoc GetLogByNameLines
// @Summary      get Logs from a container name
// @Description  get  logs from a permitted container with state exit or run.
// @Produce      json
// @Param	     name		path		string false	"container name"
// @Param	     rows		path		int	 false	"rows log number"
// @Param	     state		path		string	 false	"state of container"
// @Success      200  {object} []utils.loggerLine
// @Failure      500  {object} utils.responceHttp
// @Failure      403 {object} utils.responceHttp
// @Router       /log [get]
func GetLogByNameLines(w http.ResponseWriter, r *http.Request) {
	method := "GetLogByNameLines():"
	logdm.WriteLogLine(method + ": ---------------start request---------------")
	defer logdm.WriteLogLine(method + ": -------------------------------------------")
	ctx := context.Background()
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return
	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	if r.Method == http.MethodGet {
		lenght := r.URL.Query().Get("rows")
		namec := r.URL.Query().Get("name")
		state := r.URL.Query().Get("state")
		var onlyRunning bool = true
		std := "stdout"
		if state == "all" {
			onlyRunning = false
			std = "stderr"
		} else {
			state = "running"
		}
		logdm.WriteLogLine(method + ": container name ->" + namec + " for container in state ->" + state)
		if namec == "" {
			logdm.WriteLogLine(method + ": please provide the container name")
			http.Error(w, "error in request for container logs\n", http.StatusBadRequest)
			utils.ResponcHttpStatus("KO", w)
			return
		}
        use,notExists := checkIfCanUseContainer(namec, dockerString, httpc, onlyRunning)
		if use {
			logdm.WriteLogLine(method + ":LOG FOR CONTAINER: " + namec + " lines n. " + lenght)
			//req, err := http.NewRequest("GET", dockerString+"/containers/"+namec+"/logs?"+std+"=1&tail="+lenght, nil)
			req, err := http.NewRequest("GET", dockerString+"/containers/"+namec+"/logs?stdout=1&stderr=1&tail="+lenght, nil)
			logdm.WriteLogLine(method + ": " + dockerString + "/containers/" + namec + "/logs?" + std + "=1&tail=" + lenght)
			req.Header.Set("Accept", "application/json")
			if err != nil {
				logdm.WriteLogLine(method + ": " + err.Error())
				http.Error(w, "error in request for container logs\n", http.StatusInternalServerError)
				utils.ResponcHttpStatus("KO", w)
				return
			}
			response, err := httpc.Do(req)
			if err != nil {
				logdm.WriteLogLine(method + ": " + err.Error())
			}
			var d string

			if response.StatusCode == 200 {
				b, err := io.ReadAll(response.Body)
				if err != nil {
					logdm.WriteLogLine(method + ": " + err.Error())
				}

				d = strings.Map(utils.CleanString, string(b))
			} else {
				d = "something  goes wrong !"
				http.Error(w, "internal error!", http.StatusInternalServerError)
				utils.ResponcHttpStatus("KO", w)

				return
			}
			loglines := utils.LoggerLineSlice{}
			err = loglines.GetJSONLogsByLine(d, w)
			if err != nil {
				http.Error(w, "Unable to masrshal json", http.StatusInternalServerError)
				utils.ResponcHttpStatus("KO", w)

				return
			}
		} else {
			if notExists {
				http.Error(w, "container not exitsts!", http.StatusNotFound)
				utils.ResponcHttpStatus("KO", w)
			} else {
				http.Error(w, "Operation not permitted!", http.StatusForbidden)
				utils.ResponcHttpStatus("KO", w)
			}
			return
		}
	}

}

// godoc RestartContainer
// @Summary      restart a container by name
// @Description  restart a permitted container by name with state exit or run.
// @Produce      json
// @Param	     name		path		string false	"container name"
// @Param	     state		path		string	 false	"state of container"
// @Success      200  {object} utils.responceHttp
// @Failure      500  {object} utils.responceHttp
// @Failure      403 {object} utils.responceHttp
// @Router       /restart [post]
func RestartContainer(w http.ResponseWriter, r *http.Request) {
	method := "RestartContainer():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	logdm.WriteLogLine(method + " RESTART Container ")
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	name := keycloak.GetUserBytoken(r)
	if name == "" {
		http.Error(w, "no name token!", http.StatusInternalServerError)
		utils.ResponcHttpStatus("KO", w)
		return
	}

	logdm.WriteLogLine(method + "---------------------------")
	logdm.WriteLogLine(method + "restarting by user: " + name)
	logdm.WriteLogLine(method + "---------------------------")
	if r.Method == http.MethodPost {
		logdm.WriteLogLine(method + "try to restart container...")
		namec := r.URL.Query().Get("name")
		state := r.URL.Query().Get("state")
		var onlyRunning bool = true

		if state == "all" {
			onlyRunning = false

		} else {
			state = "running"
		}
		logdm.WriteLogLine(method + ": container name ->" + namec + " for container in state ->" + state)
		payload := strings.NewReader(`{}`)

		isLoggedRequest := utils.LoggingUser(name, namec, "restart")
		if !isLoggedRequest {
			http.Error(w, "request is not logged check db!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		use,notExists:=checkIfCanUseContainer(namec, dockerString, httpc, onlyRunning);
		if use {
			logdm.WriteLogLine(method + "restarting  the container : " + namec)
			response, err := httpc.Post(dockerString+"/containers/"+namec+"/restart", "application/octet-stream", payload)
			if err != nil {
				http.Error(w, "cannot restart the container!", http.StatusInternalServerError)
				utils.ResponcHttpStatus("KO", w)
				return
			}

			if response.StatusCode == 204 {
				logdm.WriteLogLine(method + "container restarted!")
				utils.ResponcHttpStatus("OK", w)
				return
			} else {

				http.Error(w, "container not restarted!", http.StatusInternalServerError)
				utils.ResponcHttpStatus("KO", w)
				return
			}
		} else {
			if notExists {
				http.Error(w, "container not Exists!", http.StatusNotFound)
				utils.ResponcHttpStatus("KO", w)
			} else {
				http.Error(w, "operation is forbidden", http.StatusForbidden)
				utils.ResponcHttpStatus("KO", w)
			}
			return
		}
	}
}
func Swagger(w http.ResponseWriter, r *http.Request) {

	f, err := ioutil.ReadFile("./docs/swagger.json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(f)
	return
}

// godoc SearchByName
// @Summary      search a permitted container by image.
// @Description  search a permitted container by name with state exit or run.
// @Produce      json
// @Param	     name		path		string false	"container name"
// @Param	     state		path		string	 false	"state of container"
// @Success      200  {object} utils.Container
// @Failure      500  {object} utils.responceHttp
// @Failure      403 {object} utils.responceHttp
// @Router       /searchbyname [get]
func SearchByName(w http.ResponseWriter, r *http.Request) {
	method := "SearchByName():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()

	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()
	name := r.URL.Query().Get("name")
	state := r.URL.Query().Get("state")
	exit := r.URL.Query().Get("exit")
	logdm.WriteLogLine(method + "SEARCHING container BY name: " + name)
	var onlyRunning bool = true
	var add string = ""
	if state == "all" {
		onlyRunning = false
		add = "?all=true"
		if exit == "exit0" {//                            |exit status
			add = add + "&filters=%7B%22exited%22%3A%5B%220%22%5D%7D"
		}
	} else {
		state = "running"
	}
	
	logdm.WriteLogLine(method + " In containers with state ->" + state)
	canUse,notExists := checkIfCanUseContainer(name, dockerString, httpc, onlyRunning)
	if notExists {
		http.Error(w, "container not exists.", http.StatusNotFound)
		utils.ResponcHttpStatus("KO", w)
		return
	} else {
		if !canUse {
			http.Error(w, "container is Not Permitted", http.StatusForbidden)
			utils.ResponcHttpStatus("KO", w)
			return
		}
	}

	if r.Method == http.MethodGet {
		response, err := httpc.Get(dockerString + "/containers/json" + add)
		if err != nil {
			logdm.WriteLogLine(method + "error! getting list of all container")
			http.Error(w, "cannot get list container", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var d string
		if response.StatusCode == 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				logdm.WriteLogLine(method + err.Error())
			}
			d = string(b)
		} else {
			d = "something  goes wrong !"
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		containers := utils.Containers{}
		err, _ = containers.GetSearchInContainers(d, name, w)
		if err != nil {
			http.Error(w, "Unable to masrshal json", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

	}

}

// godoc ListByImage
// @Summary      lists all permitted with an image.
// @Description  lists all permitted container by name with state exit or run.
// @Produce      json
// @Param	     name		path		string false	"image name"
// @Param	     state		path		string	 false	"state of container"
// @Success      200  {object} []utils.Container
// @Failure      500  {object} utils.responceHttp
// @Failure      403 {object} utils.responceHttp
// @Router       /listbyimage [get]
func ListByImage(w http.ResponseWriter, r *http.Request) {
	method := "SearchByImage():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	logdm.WriteLogLine(method + "SEARCHING container by image:")
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()
	imgname := r.URL.Query().Get("imgname")

	state := r.URL.Query().Get("state")

	var add string = ""
	if state == "all" {
		add = "?all=true"
		logdm.WriteLogLine(method + "this is a search in container by image IN ALL CONTAINER with image" + imgname)
	} else {
		logdm.WriteLogLine(method + "this is a search in container by image IN ALL RUNNING CONTAINER with image " + imgname)
	}

	if !utils.CanUseImage(imgname) {
		http.Error(w, "Operation not permitted!", http.StatusForbidden)
		utils.ResponcHttpStatus("KO", w)
		return
	}

	if r.Method == http.MethodGet {

		response, err := httpc.Get(dockerString + "/containers/json" + add)
		if err != nil {
			logdm.WriteLogLine(method + "error! getting list of all container")
			http.Error(w, "cannot get list container", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var d string
		if response.StatusCode == 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				logdm.WriteLogLine(method + err.Error())
			}
			d = string(b)
		} else {
			d = "something  goes wrong !"
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		containers := utils.Containers{}
		err, _ = containers.GetSearchInContainersByimage(d, imgname, w)
		if err != nil {
			http.Error(w, "Unable to masrshal json", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

	}

}
/*
func GetInfoByNname (w http.ResonseWriter, r *http.Request) {
	method := "GetInfoByNname():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	keycloakUser := keycloak.GetUserBytoken(r)
	logdm.WriteLogLine(method + "getting info about a container:")
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	name := r.URL.Query().Get("name")
	logdm.WriteLogLine(method + "container :" + name)

	if r.Method == http.MethodGet {
		keyCkUserName := keycloak.GetUserBytoken(r)
		if keyCkUserName == "" {
			http.Error(w, "no name token!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		response, err := httpc.Get(dockerString + "/containers/json?all=true")
		if err != nil {
			logdm.WriteLogLine(method + "error! getting list of all container")
			http.Error(w, "cannot get list container", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var d string
		if response.StatusCode == 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				logdm.WriteLogLine(method + err.Error())
			}
			d = string(b)
		} else {
			d = "something  goes wrong !"
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

	}
	
}
*/
// godoc StopIt
// @Summary      stop a permitted container by name.
// @Description  stop a permitted container by name.
// @Produce      json
// @Param	     name		path		string	 false	"name of container"
// @Success      200  {object} utils.responceHttp
// @Failure      500  {object} utils.responceHttp
// @Failure      403 {object} utils.responceHttp
// @Router       /searchbyname [post]
func StopIt(w http.ResponseWriter, r *http.Request) {
	method := "StopItWithPermImage():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	logdm.WriteLogLine(method + "STOP Container by image.")
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	name := r.URL.Query().Get("name")

	logdm.WriteLogLine(method + "container :" + name)
	if r.Method == http.MethodPost {
		keyCkUserName := keycloak.GetUserBytoken(r)
		if keyCkUserName == "" {
			http.Error(w, "no name token!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		payload := strings.NewReader(`{}`)
		response, err := httpc.Get(dockerString + "/containers/json")
		if err != nil {
			logdm.WriteLogLine(method + "error! getting list of all container")
			http.Error(w, "cannot get list container", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var d string
		if response.StatusCode == 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				logdm.WriteLogLine(method + err.Error())
			}
			d = string(b)
		} else {
			d = "something  goes wrong !"
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		containers := utils.Containers{}
		imageRes, networks := containers.GetImageAndNetwsFromCname(d, name)
		if imageRes == "" && len(networks) == 0 {
			logdm.WriteLogLine(method + "error! container not Found with name" + name)
			http.Error(w, "cannot get container with name", http.StatusNotFound)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		if !utils.CanUseImage(imageRes) || !utils.CanUseNetworks(networks) {
			http.Error(w, "Operation not permitted!", http.StatusForbidden)
			logdm.WriteLogLine(method + " the container " + name + "has an image" + imageRes + "that is NOT permeitted")
			utils.ResponcHttpStatus("KO", w)
			return
		}

		logdm.WriteLogLine(method + " the container " + name + "has an image" + imageRes + "that is permeitted")
		logdm.WriteLogLine(method + " the container " + name + "is on permitted networks")
		response, err = httpc.Post(dockerString+"/containers/"+name+"/stop", "application/octet-stream", payload)
		if err != nil {
			logdm.WriteLogLine(method + " cannot stop the container.")
			http.Error(w, "cannot stop the container, error doing post!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		logdm.WriteLogLine(method + "")
		if response.StatusCode == 204 {
			isLoggedRequest := utils.LoggingUser(name, keyCkUserName, "stop")
			if !isLoggedRequest {
				logdm.WriteLogLine(method + " request cannot be inserted in db. ERROR")
				http.Error(w, "request is not logged check db!", http.StatusInternalServerError)
				utils.ResponcHttpStatus("KO", w)
				return
			}
			logdm.WriteLogLine(method + "container is STOPPED!")
			utils.ResponcHttpStatus("OK", w)
			return
		} else {
			logdm.WriteLogLine("user :" + keyCkUserName + " has tryed to stop container " + name + " but nothing is done.")
			http.Error(w, "cannot stop the container!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
	}
}

// godoc RemoveExitByImage
// @Summary      remove exited container with an image.
// @Description  remove exited container with a permitted image.
// @Produce      json
// @Param	     name		path		string false	"image name"
// @Success      200  {object} utils.responceHttp
// @Failure      500  {object} utils.responceHttp
// @Failure      403 {object} utils.responceHttp
// @Router       /removexitbyimage [delete]
func RemoveExitByImage(w http.ResponseWriter, r *http.Request) {
	method := "RemoveItbyImage():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	keycloakUser := keycloak.GetUserBytoken(r)
	logdm.WriteLogLine(method + "SEARCHING container in stopped state by image:")
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()
	imgname := r.URL.Query().Get("imgname")
	if !utils.CanUseImage(imgname) || imgname=="" || imgname=="all" {
		http.Error(w, "Operation not permitted!", http.StatusForbidden)
		utils.ResponcHttpStatus("KO", w)
		return
	}
	isLoggedRequest := utils.LoggingUser(keycloakUser, "all container with image : "+imgname, "remove in exit state.")
	if !isLoggedRequest {
		http.Error(w, "request is not logged check db!", http.StatusInternalServerError)
		utils.ResponcHttpStatus("KO", w)
		return
	}
	logdm.WriteLogLine(method + " " + imgname)
	if r.Method == http.MethodDelete {
		logdm.WriteLogLine(method + "this is a search in container by image in stopped state .")
		response, err := httpc.Get(dockerString + "/containers/json?all=true")
		if err != nil {
			logdm.WriteLogLine(method + "error! getting list of all container")
			http.Error(w, "cannot get list container", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var d string
		if response.StatusCode == 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				logdm.WriteLogLine(method + err.Error())
			}
			d = string(b)
		} else {
			d = "something  goes wrong !"
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		containers := utils.Containers{}
		_, containersSelected := containers.GetSearchInStopContainersByNameOrNot(d, imgname, os.Stdout)
		var status bool = true
		var actualStatus bool = true
		if len(containersSelected) == 0 {
			logdm.WriteLogLine(method + " no contatiner to remove in search.")
			w.WriteHeader(http.StatusNotFound)
			utils.ResponcHttpStatus("OK", w)
			return
		}
		count := 0
		for _, k := range containersSelected {
			count++
			containerName := k.Names[0]
			logdm.WriteLogLine(method + "remove this containtainer in stop  state :" + containerName)

			deleteContainerStr := dockerString + "/containers/" + containerName
			logdm.WriteLogLine(method + " creating delete request " + deleteContainerStr)
			req, err := http.NewRequest("DELETE", deleteContainerStr, nil)
			if err != nil {
				logdm.WriteLogLine(method + "error! create request for remove container container " + containerName)
				http.Error(w, "cannot remove container "+containerName, http.StatusInternalServerError)
				utils.ResponcHttpStatus("KO", w)
				return
			}
			response, err := httpc.Do(req)
			if err != nil {
				logdm.WriteLogLine(method + "error! deleting container " + containerName)
				http.Error(w, "cannot remove container "+containerName, http.StatusInternalServerError)
				utils.ResponcHttpStatus("KO", w)
				return
			}
			if response.StatusCode == 204 {
				logdm.WriteLogLine(method + "container is removed! with name " + containerName)
				actualStatus = true
			} else {
				logdm.WriteLogLine(method + "container is not removed! with name " + containerName + "status code http " + strconv.Itoa(response.StatusCode))
				actualStatus = false
			}
			status = actualStatus && status

			response.Body.Close()
		}
		if !status {
			logdm.WriteLogLine(method + "not all container are removed")
			w.WriteHeader(http.StatusBadRequest)
			utils.ResponcHttpStatus("KO", w)
			return
		} else {
			logdm.WriteLogLine(method + "all container are removed with SUCCESS")
			w.WriteHeader(http.StatusOK)
			utils.ResponcHttpStatus(strconv.Itoa(count), w)
			return
		}
	}

}

func checkIfCanUseContainer(name string, dockerString string, httpc http.Client, onlyRun bool) (bool,bool) {
	method := "checkIfCanUseContainer():"
	logdm.WriteLogLine(method + "this is a search in container by image.")
	dockerUrl := dockerString + "/containers/json"
	if !onlyRun {
		dockerUrl = dockerUrl + "?all=true"
		logdm.WriteLogLine(method + " getting list , of all containter running state or exited. " + dockerUrl)
	} else {
		logdm.WriteLogLine(method + " getting list , of all containter running state.")
	}
	response, err := httpc.Get(dockerUrl)
	if err != nil {
		logdm.WriteLogLine(method + "error! getting list of all container")
		return false,false
	}
	var d string
	if response.StatusCode == 200 {
		b, err := io.ReadAll(response.Body)
		if err != nil {
			logdm.WriteLogLine(method + err.Error())
			return false,false
		}
		d = string(b)
	} else {
		logdm.WriteLogLine(method + " return code is " + response.Status)
		return false,false
	}
	containers := utils.Containers{}
	image, networks := containers.GetImageAndNetwsFromCname(d, name)
	if image == "" && len(networks) == 0 {
		logdm.WriteLogLine(method + " container with name " + name + " NOT FOUND")
		return false,true
	} else {
		canUseImg := utils.CanUseImage(image)
		canUseNet := utils.CanUseNetworks(networks)
		use := canUseImg && canUseNet
		if use {
			logdm.WriteLogLine(method + " container with name " + name + " is PERMITTED")
		} else {
			logdm.WriteLogLine(method + " container with name " + name + " is NOT PERMITTED: canUseImg: " + strconv.FormatBool(canUseImg) + " canUseNets: " + strconv.FormatBool(canUseNet))
		}
		return use,false
	}
}

// godoc GetAllPermitted
// @Summary     list all containers with a permitted image.
// @Description  list all containers with a permitted image in state running or exited..
// @Produce      json
// @Param	     state		path		string	 false	"state of container"
// @Success      200  {object} []utils.Container
// @Failure      500  {object} utils.responceHttp
// @Failure      403 {object} utils.responceHttp
// @Router       /list [get]
func GetAllPermitted(w http.ResponseWriter, r *http.Request) {
	method := "GetAllPermitted():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	logdm.WriteLogLine(method + "LIST ALL permitted container.")
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()
	state := r.URL.Query().Get("state")
	var add string = ""
	if state == "all" {
		add = "?all=true"
		logdm.WriteLogLine(method + "get list for ALL CONTAINERS.")
	} else {
		logdm.WriteLogLine(method + "get list for RUNNING CONTAINER Only.")
	}
	if r.Method == http.MethodGet {

		response, err := httpc.Get(dockerString + "/containers/json" + add)
		if err != nil {
			logdm.WriteLogLine(method + "error! getting list of all container")
			http.Error(w, "cannot get list container", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var d string
		if response.StatusCode == 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				logdm.WriteLogLine(method + err.Error())
			}
			d = string(b)
		} else {
			d = "something  goes wrong !"
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

		containers := utils.Containers{}
		err = containers.GetAllPermittedByImgNet(d, w)
		if err != nil {
			http.Error(w, "Unable to masrshal json", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

	}
}

func StopAllContainerByImage(w http.ResponseWriter, r *http.Request) {
	method := "StopAllContainerByImage():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	keycloakUser := keycloak.GetUserBytoken(r)
	logdm.WriteLogLine(method + "SEARCHING container in stopped state by image:")
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()
	imgname := r.URL.Query().Get("imgname")
	isLoggedRequest := utils.LoggingUser(keycloakUser, "all with image "+imgname, "Stop")
	if !isLoggedRequest {
		http.Error(w, "request is not logged check db!", http.StatusInternalServerError)
		utils.ResponcHttpStatus("KO", w)
		return
	}
	logdm.WriteLogLine(method + " " + imgname)
	if r.Method == http.MethodPost {
		logdm.WriteLogLine(method + "this is a STOP containers by image.")
		response, err := httpc.Get(dockerString + "/containers/json")
		if err != nil {
			logdm.WriteLogLine(method + "error! getting list of all container")
			http.Error(w, "cannot get list container", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var d string
		if response.StatusCode == 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				logdm.WriteLogLine(method + err.Error())
			}
			d = string(b)
		} else {
			d = "something  goes wrong !"
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		containers := utils.Containers{}
		err, containersSelected := containers.GetSearchInContainersByimage(d, imgname, os.Stdout)
		payload := strings.NewReader(`{}`)
		var status bool = true
		var actualStatus bool = true
		if len(containersSelected) == 0 {
			logdm.WriteLogLine(method + " no contatiner to stop in search with image: " + imgname)
			w.WriteHeader(http.StatusNotFound)
			utils.ResponcHttpStatus("OK", w)
			return
		}
		count := 0
		for _, k := range containersSelected {
			count++
			containerName := k.Names[0]
			logdm.WriteLogLine(method + "stop this containtainer in running state :" + containerName)

			stopContainerStr := dockerString + "/containers/" + containerName + "/stop"

			logdm.WriteLogLine(method + " creating delete request " + stopContainerStr)
			req, err := http.NewRequest("POST", stopContainerStr, payload)
			if err != nil {
				logdm.WriteLogLine(method + "error! create request for stop container: " + containerName)
				http.Error(w, "cannot remove container "+containerName, http.StatusInternalServerError)
				utils.ResponcHttpStatus("KO", w)
				return
			}
			response, err := httpc.Do(req)
			if err != nil {
				logdm.WriteLogLine(method + "error! stopping container: " + containerName)
				http.Error(w, "cannot stop container "+containerName, http.StatusInternalServerError)
				utils.ResponcHttpStatus("KO", w)
				return
			}
			if response.StatusCode == 204 {
				logdm.WriteLogLine(method + "container is stopped! with name :" + containerName)
				actualStatus = true
			} else {
				logdm.WriteLogLine(method + "container is not stopped! with name " + containerName + "status code http " + strconv.Itoa(response.StatusCode))
				actualStatus = false
			}
			status = actualStatus && status
			response.Body.Close()
		}
		if !status {
			logdm.WriteLogLine(method + "not all container are stopped with image : " + imgname)
			w.WriteHeader(http.StatusBadRequest)
			utils.ResponcHttpStatus("KO", w)
			return
		} else {
			logdm.WriteLogLine(method + "all container are stopped with SUCCESS")
			w.WriteHeader(http.StatusOK)
			utils.ResponcHttpStatus(strconv.Itoa(count), w)
			return
		}
	}

}

//
func Prune(w http.ResponseWriter, r *http.Request) {
	method := "Prune():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	keycloakUser := keycloak.GetUserBytoken(r)
	logdm.WriteLogLine(method + "SEARCHING container in stopped state on permitted network:")
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()
	
	isLoggedRequest := utils.LoggingUser(keycloakUser, "all on permitted network","prune")
	if !isLoggedRequest {
		http.Error(w, "request is not logged check db!", http.StatusInternalServerError)
		utils.ResponcHttpStatus("KO", w)
		return
	}
	
	if r.Method == http.MethodDelete {
		logdm.WriteLogLine(method + "this is a search in container by image in stopped state on permitted network.")
		response, err := httpc.Get(dockerString + "/containers/json?all=true")
		if err != nil {
			logdm.WriteLogLine(method + "error! getting list of all container")
			http.Error(w, "cannot get list container", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var d string
		if response.StatusCode == 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				logdm.WriteLogLine(method + err.Error())
			}
			d = string(b)
		} else {
			d = "something  goes wrong !"
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		containers := utils.Containers{}
		_, containersSelected := containers.GetSearchInStopContainersByNameOrNot(d, "all", os.Stdout)
		var status bool = true
		var actualStatus bool = true
		if len(containersSelected) == 0 {
			logdm.WriteLogLine(method + " no contatiner to remove in search.")
			w.WriteHeader(http.StatusNotFound)
			utils.ResponcHttpStatus("OK", w)
			return
		}
		count := 0
		for _, k := range containersSelected {
			count++
			containerName := k.Names[0]
			logdm.WriteLogLine(method + "remove this containtainer in stop  state :" + containerName)

			deleteContainerStr := dockerString + "/containers/" + containerName
			logdm.WriteLogLine(method + " creating delete request " + deleteContainerStr)
			req, err := http.NewRequest("DELETE", deleteContainerStr, nil)
			if err != nil {
				logdm.WriteLogLine(method + "error! create request for remove container container " + containerName)
				http.Error(w, "cannot remove container "+containerName, http.StatusInternalServerError)
				utils.ResponcHttpStatus("KO", w)
				return
			}
			response, err := httpc.Do(req)
			if err != nil {
				logdm.WriteLogLine(method + "error! deleting container " + containerName)
				http.Error(w, "cannot remove container "+containerName, http.StatusInternalServerError)
				utils.ResponcHttpStatus("KO", w)
				return
			}
			if response.StatusCode == 204 {
				logdm.WriteLogLine(method + "container is removed! with name " + containerName)
				actualStatus = true
			} else {
				logdm.WriteLogLine(method + "container is not removed! with name " + containerName + "status code http " + strconv.Itoa(response.StatusCode))
				actualStatus = false
			}
			status = actualStatus && status

			response.Body.Close()
		}
		if !status {
			logdm.WriteLogLine(method + "not all container are removed")
			w.WriteHeader(http.StatusBadRequest)
			utils.ResponcHttpStatus("KO", w)
			return
		} else {
			logdm.WriteLogLine(method + "all container are removed with SUCCESS")
			w.WriteHeader(http.StatusOK)
			utils.ResponcHttpStatus(strconv.Itoa(count), w)
			return
		}
	}

}


//

func GetExited(w http.ResponseWriter, r *http.Request) {
	method := "GetExited:"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	logdm.WriteLogLine(method + "get all container in exit state.")
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	if r.Method == http.MethodGet {

		response, err := httpc.Get(dockerString + "/containers/json?all=true")
		if err != nil {
			logdm.WriteLogLine(method + "error! getting list of all container")
			http.Error(w, "cannot get list container", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var d string
		if response.StatusCode == 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				logdm.WriteLogLine(method + err.Error())
			}
			d = string(b)
		} else {
			d = "something  goes wrong !"
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

		containers := utils.Containers{}
		err, _ = containers.GetSearchInStopContainersByNameOrNot(d, "", w)
		if err != nil {
			http.Error(w, "Unable to masrshal json", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

	}
}

func VolumePrunes(w http.ResponseWriter, r *http.Request) {
	method := "VolumePrunes():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return
	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	name := keycloak.GetUserBytoken(r)
	if name == "" {
		http.Error(w, "no name token!", http.StatusInternalServerError)
		utils.ResponcHttpStatus("KO", w)
		return
	}
	logdm.WriteLogLine(method + "---------------------------")
	logdm.WriteLogLine(method + "prune volume by user: " + name)
	logdm.WriteLogLine(method + "---------------------------")
	if r.Method == http.MethodPost {
		payload := strings.NewReader(`{}`)
		isLoggedRequest := utils.LoggingUser(name, "volume", "prune")
		if !isLoggedRequest {
			http.Error(w, "request is not logged check db!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		response, err := httpc.Post(dockerString+"/volumes/prune", "application/octet-stream", payload)
		if err != nil {
			http.Error(w, "cannot restart the container!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var d string
		if response.StatusCode == 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				logdm.WriteLogLine(method + err.Error())
			}
			d = string(b)
		} else {
			d = "something  goes wrong !"
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		volumesPrunes := utils.VolumesPrune{}
		err = volumesPrunes.VolumesPrune(d, w)
		if err != nil {
			http.Error(w, "Unable to masrshal json", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

	}
}

func CreateContainer(w http.ResponseWriter, r *http.Request) {
	method := "CreateContainer():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return
	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	name := keycloak.GetUserBytoken(r)
	if name == "" {
		http.Error(w, "no name token!", http.StatusInternalServerError)
		utils.ResponcHttpStatus("KO", w)
		return
	}
	logdm.WriteLogLine(method + "---------------------------")
	logdm.WriteLogLine(method + "create container by user: " + name)
	logdm.WriteLogLine(method + "---------------------------")
	if r.Method == http.MethodPost {

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cannot read json from request", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		strJSON := string(body)
		var containerJSON utils.ContainerToCreate
		err = containerJSON.GetContainerToCreateInfo(strJSON)

		if err != nil || containerJSON.Image == "" || containerJSON.Network == "" || containerJSON.Name == "" {
			http.Error(w, "ERROR IN JSON create CONTAINER!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

		imageName := containerJSON.Image
		nameC := containerJSON.Name
		network := containerJSON.Network
		volumes := containerJSON.Volumes
		env := containerJSON.Env

		//check if i can use container
		if !utils.CanUseImage(imageName) || !utils.CanUseNetWork(network) {
			http.Error(w, "ERROR IN JSON create CONTAINER not Permitted image or network!", http.StatusForbidden)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		//check if container already exists with that name
		containersJsonList, ok := getJsonContainer(nameC, dockerString, httpc, false)
		if !ok {
			http.Error(w, "ERROR GETTING JSON CONTAINER LIST !: ", http.StatusBadRequest)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		c := utils.Containers{}
		_, contExists := c.GetSearchInContainers(containersJsonList, nameC, os.Stdout)
		if contExists {
			http.Error(w, "ERROR container already exists!", http.StatusConflict)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		//for every volume with VolumeType Volume check if the volume exists else exit.
		//get the json volume list
		jsonVolumesList, result := GetJsonVolumes(httpc, dockerString)
		if !result {
			http.Error(w, "ERROR getting the container create JSON volume list ", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		//check every volume type Volume exiSts
		for _, vol := range volumes {
			if vol.VolumeType == "Volume" {
				var volumes utils.Volumes
				volumeExists := volumes.SearchVolumeByName(jsonVolumesList, vol.Volume)
				if !volumeExists {
					http.Error(w, "volume does not exists "+vol.Volume, http.StatusConflict)
					logdm.WriteLogLine(method + " volume does not exists: " + vol.Volume)
					utils.ResponcHttpStatus("KO", w)
					return
				}
			}
		}
		Container := utils.ToJsonContainer{}
		ok, strCreate := Container.MAKEJsonContainer(imageName, network, volumes, env)
		if strCreate == "" || !ok {
			http.Error(w, "ERROR IN JSON create TYPE CONTAINER !: ", http.StatusBadRequest)
			utils.ResponcHttpStatus("KO", w)
			return
		}

		logdm.WriteLogLine(method + " " + strCreate)
		payload := strings.NewReader(strCreate)
		isLoggedRequest := utils.LoggingUser(name, "container", "create container")
		if !isLoggedRequest {
			http.Error(w, "request is not logged check db!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

		response, err := httpc.Post(dockerString+"/containers/create?name="+nameC, "application/json", payload)

		if err != nil {
			var strerr string = err.Error()
			logdm.WriteLogLine(method + " " + strerr)
			http.Error(w, "cannot Create the container!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		logdm.WriteLogLine(method + " container create string " + dockerString + "/containers/create?name=" + nameC)
		if response.StatusCode == 201 {
			logdm.WriteLogLine(method + " container " + nameC + " created.")
		} else {
			strerr := strconv.Itoa(response.StatusCode)
			logdm.WriteLogLine(method + "errore with contaier create http error" + strerr)
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		payloadStart := strings.NewReader(`{}`)

		responseStart, err := httpc.Post(dockerString+"/containers/"+nameC+"/start", "application/octet-stream", payloadStart)
		logdm.WriteLogLine(method + " post done ..." + dockerString + "/containers/create" + nameC + "/start" + " status " + strconv.Itoa(responseStart.StatusCode))
		if err != nil {
			http.Error(w, "cannot restart the container!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		if responseStart.StatusCode == 204 {
			logdm.WriteLogLine(method + " container " + nameC + " STARTED.")
			utils.ResponcHttpStatus("OK", w)
			return
		} else {
			if responseStart.StatusCode == 304 {
				logdm.WriteLogLine(method + " container " + nameC + " ALREADY STARTED.")
				http.Error(w, "internal error!", http.StatusNotModified)
			} else {
				http.Error(w, "internal error!", http.StatusInternalServerError)

			}

			logdm.WriteLogLine(method + " container " + nameC + " not started")
			utils.ResponcHttpStatus("KO", w)
			return
		}

		//
	}
}

func getJsonContainer(name string, dockerString string, httpc http.Client, onlyRun bool) (string, bool) {
	method := "getContainerJsonList():"
	dockerUrl := dockerString + "/containers/json"
	if !onlyRun {
		dockerUrl = dockerUrl + "?all=true"
		logdm.WriteLogLine(method + " getting list , of all containter running state or exited. " + dockerUrl)
	} else {
		logdm.WriteLogLine(method + " getting list , of all containter running state.")
	}
	response, err := httpc.Get(dockerUrl)
	if err != nil {
		logdm.WriteLogLine(method + "error! getting list of all container")
		return "", false
	}
	var d string
	if response.StatusCode == 200 {
		b, err := io.ReadAll(response.Body)
		if err != nil {
			logdm.WriteLogLine(method + err.Error())
			return "", false
		}
		d = string(b)
	} else {
		logdm.WriteLogLine(method + " return code is " + response.Status)
		return "", false
	}
	//logdm.WriteLogLine(method + " json container list :" + d)
	return d, true
}
func GetJsonVolumes(httpc http.Client, dockerString string) (string, bool) {
	method := "GetJsonVolumes():"
	response, err := httpc.Get(dockerString + "/volumes")
	if err != nil {
		logdm.WriteLogLine(method + "error! getting list of all Volumes")
		return "error getting volumes", false
	}
	var d string
	if response.StatusCode == 200 {
		b, err := io.ReadAll(response.Body)
		if err != nil {
			logdm.WriteLogLine(method + err.Error())
			return "error reading bodyresponce", false
		}
		d = string(b)
	} else {
		d = "volume get status is not 200"
		return d, false
	}
	return d, true
}

func VolumeList(w http.ResponseWriter, r *http.Request) {
	method := "VolumeList():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return
	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	name := keycloak.GetUserBytoken(r)
	if name == "" {
		http.Error(w, "no name token!", http.StatusInternalServerError)
		utils.ResponcHttpStatus("KO", w)
		return
	}
	logdm.WriteLogLine(method + "---------------------------")
	logdm.WriteLogLine(method + "list volumes by user: " + name)
	logdm.WriteLogLine(method + "---------------------------")
	if r.Method == http.MethodGet {
		isLoggedRequest := utils.LoggingUser(name, "volume", "list")
		if !isLoggedRequest {
			http.Error(w, "request is not logged check db!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

		response, err := httpc.Get(dockerString + "/volumes")
		if err != nil {
			http.Error(w, "cannot list!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var d string
		if response.StatusCode == 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				logdm.WriteLogLine(method + err.Error())
			}
			d = string(b)
		} else {
			logdm.WriteLogLine(method + ".......-----")
			d = "something  goes wrong !"
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		volumesList := utils.Volumes{}
		err = volumesList.VolumesList(d, w)
		if err != nil {
			http.Error(w, "Unable to masrshal json", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

	}
}

func Getvolumeinstate(w http.ResponseWriter, r *http.Request) {
	method := "Getvolume():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return
	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	name := keycloak.GetUserBytoken(r)
	if name == "" {
		http.Error(w, "no name token!", http.StatusInternalServerError)
		utils.ResponcHttpStatus("KO", w)
		return
	}
	used := r.URL.Query().Get("used")
	queryusedVol,err := strconv.ParseBool(used)
	if err != nil {
		if used == "all" {
			queryusedVol = false
		}else {
			http.Error(w, "bad USED parameter!", http.StatusBadRequest)
			utils.ResponcHttpStatus("KO", w)
			return
		}	
	}
	filters :=""
	msgDb := ""
	if queryusedVol {
		logdm.WriteLogLine(method + "---------------------------")
		logdm.WriteLogLine(method + "list USED volumes by user: " + name)
		logdm.WriteLogLine(method + "---------------------------")
		msgDb ="used volume"
		filters =""
	} else {
		if used == "all" {
			logdm.WriteLogLine(method + "---------------------------")
			logdm.WriteLogLine(method + "list volumes by user: " + name)
			logdm.WriteLogLine(method + "---------------------------")
			msgDb = "volume"
			filters =""
		} else {
			logdm.WriteLogLine(method + "---------------------------")
			logdm.WriteLogLine(method + "list UNUSED Volumes by user: " + name)
			logdm.WriteLogLine(method + "---------------------------")
			msgDb ="unused volume"
			filters ="?filters=%7B%22dangling%22%3A%5B%22true%22%5D%7D"
		}
	}
	
	if r.Method == http.MethodGet {
		isLoggedRequest := utils.LoggingUser(name, msgDb, "list")
		if !isLoggedRequest {
			http.Error(w, "request is not logged check db!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

		response, err := httpc.Get(dockerString + "/volumes"+filters)
		if err != nil {
			http.Error(w, "cannot list!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var d string
		if response.StatusCode == 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				logdm.WriteLogLine(method + err.Error())
			}
			d = string(b)
		} else {
			logdm.WriteLogLine(method + ".......-----")
			d = "something  goes wrong !"
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		volumesList := utils.Volumes{}
		err = volumesList.VolumesList(d, w)
		if err != nil {
			http.Error(w, "Unable to masrshal json", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

	}
}

func VolumeCreate(w http.ResponseWriter, r *http.Request) {
	method := "VolumeCreate():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return
	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	name := keycloak.GetUserBytoken(r)
	if name == "" {
		http.Error(w, "no name token!", http.StatusInternalServerError)
		utils.ResponcHttpStatus("KO", w)
		return
	}
	logdm.WriteLogLine(method + "---------------------------")
	logdm.WriteLogLine(method + "create Volume by user: " + name)
	logdm.WriteLogLine(method + "---------------------------")
	if r.Method == http.MethodPost {

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cannot read json from request", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		strJSON := string(body)

		var volumeJSON utils.VolumeToCreate
		err = volumeJSON.GetVolumeToCreateInfo(strJSON)

		if err != nil {
			http.Error(w, "ERROR IN JSON CONTAINER!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		name := volumeJSON.Name

		jsonVolumesList, result := GetJsonVolumes(httpc, dockerString)
		if !result {
			http.Error(w, "ERROR getting volume list "+name, http.StatusForbidden)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var volumes utils.Volumes
		volumeExists := volumes.SearchVolumeByName(jsonVolumesList, name)
		if volumeExists {
			http.Error(w, "ERROR volume to create already exists! "+name, http.StatusForbidden)
			utils.ResponcHttpStatus("KO", w)
			return
		}

		strCreate := "{\"Name\":\"" + name + "\",\"DriverOpts\":{}}"
		logdm.WriteLogLine(method + " " + strCreate)
		payload := strings.NewReader(strCreate)
		isLoggedRequest := utils.LoggingUser(name, "Volume", "create")
		if !isLoggedRequest {
			http.Error(w, "request is not logged check db!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		response, err := httpc.Post(dockerString+"/volumes/create", "application/json", payload)
		if err != nil {
			http.Error(w, "cannot create Volume!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		if response.StatusCode == 201 {
			logdm.WriteLogLine(method + " Volume: " + name + " created.")
			utils.ResponcHttpStatus("OK", w)
		} else {
			logdm.WriteLogLine(method + " Volume: " + name + " NOT created. http error " + strconv.Itoa(response.StatusCode))
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

	}
}

func ReloadEnv(w http.ResponseWriter, r *http.Request) {
	method := "ReloadEnv():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	message, _, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	logdm.WriteLogLine(method + " RESTART Container ")
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	name := keycloak.GetUserBytoken(r)
	if name == "" {
		http.Error(w, "no name token!", http.StatusInternalServerError)
		utils.ResponcHttpStatus("KO", w)
		return
	}

	logdm.WriteLogLine(method + "---------------------------")
	logdm.WriteLogLine(method + "reloadEnv by user: " + name)
	logdm.WriteLogLine(method + "---------------------------")
	if r.Method == http.MethodPost {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cannot read json from request", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		strJSON := string(body)
		logdm.WriteLogLine(method + "body.." + strJSON)
		var envJSON utils.NewEnv
		
		err = envJSON.GetNeWEnv(strJSON)
		logdm.WriteLogLine(method + "try to reload container...")
		currentEnv := os.Environ()
		
		err1 := os.Setenv("PERMITTED_IMAGES",envJSON.ImagePerm)
		err2 := os.Setenv("PERMITTED_NETWORKS",envJSON.NetworkPerm)
		if err1 != nil || err2 != nil {
			logdm.WriteLogLine(method + "change env vars for PERMITTED_IMAGES="+envJSON.ImagePerm+" PERMITTED_NETWORKS="+envJSON.NetworkPerm + "NOT PERMITTED")
			fmt.Println(currentEnv)
			http.Error(w, "cannot create Volume!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		logdm.WriteLogLine(method + "change env vars for PERMITTED_IMAGES="+envJSON.ImagePerm+" PERMITTED_NETWORKS="+envJSON.NetworkPerm)
		utils.ResponcHttpStatus("OK", w)
		return
	}
}

func PrintEnv(w http.ResponseWriter, r *http.Request){
	method := "PrintEnv():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	message, _, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	logdm.WriteLogLine(method + " Get environment for dockermanager: ")
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	name := keycloak.GetUserBytoken(r)
	if name == "" {
		http.Error(w, "no name token!", http.StatusInternalServerError)
		utils.ResponcHttpStatus("KO", w)
		return
	}

	logdm.WriteLogLine(method + "---------------------------")
	logdm.WriteLogLine(method + "reloadEnv by user: " + name)
	logdm.WriteLogLine(method + "---------------------------")
	if r.Method == http.MethodGet {	
		env := os.Environ()
		var envToJson utils.LoggerLineSlice
		envToJson.GetJsonEnv(env,w)
		
		fmt.Println(env)
	}
	return
	
}


func GetListContainerBind(w http.ResponseWriter, r *http.Request) {
	method := "getListContainerVolumebind():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	logdm.WriteLogLine(method + "list container with  bind:")
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	if r.Method == http.MethodGet {

		response, err := httpc.Get(dockerString + "/containers/json")
		if err != nil {
			logdm.WriteLogLine(method + "error! getting list of all container")
			http.Error(w, "cannot get list container", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var d string
		if response.StatusCode == 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				logdm.WriteLogLine(method + err.Error())
			}
			d = string(b)
		} else {
			d = "something  goes wrong !"
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		containers := utils.Containers{}
		err, _ = containers.GetSearchInContainersWithBinds(d, w)
		if err != nil {
			http.Error(w, "Unable to masrshal json", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

	}

}

func GetBinds(w http.ResponseWriter, r *http.Request) {
	method := " GetBind():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	logdm.WriteLogLine(method + "list  binds :")
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	if r.Method == http.MethodGet {

		response, err := httpc.Get(dockerString + "/containers/json")
		if err != nil {
			logdm.WriteLogLine(method + "error! getting list of all container")
			http.Error(w, "cannot get list container", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var d string
		if response.StatusCode == 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				logdm.WriteLogLine(method + err.Error())
			}
			d = string(b)
		} else {
			d = "something  goes wrong !"
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		containers := utils.Containers{}
		err = containers.GetBindsList(d, w)
		if err != nil {
			http.Error(w, "Unable to masrshal json", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

	}

}

func KillIt(w http.ResponseWriter, r *http.Request) {
	method := "killIt():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	logdm.WriteLogLine(method + "kill a container.")
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	name := r.URL.Query().Get("name")
	state := r.URL.Query().Get("state")
	var add string = ""
	if state == "all" {
		add = "?all=true"
		logdm.WriteLogLine(method + "killing the container with name "+ name +" even if it is stopped ")
	} else {
		logdm.WriteLogLine(method + "killing the container with name " + name +" only if it is running")
	}
	
	
	if r.Method == http.MethodDelete {
		keyCkUserName := keycloak.GetUserBytoken(r)
		if keyCkUserName == "" {
			http.Error(w, "no name token!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		response, err := httpc.Get(dockerString + "/containers/json"+add)
		if err != nil {
			logdm.WriteLogLine(method + "error! getting list of all container")
			http.Error(w, "cannot get list container", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var d string
		if response.StatusCode == 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				logdm.WriteLogLine(method + err.Error())
			}
			d = string(b)
		} else {
			d = "something  goes wrong !"
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		containers := utils.Containers{}
		imageRes, networks := containers.GetImageAndNetwsFromCname(d, name)
		if imageRes == "" && len(networks) == 0 {
			logdm.WriteLogLine(method + "error! container not Found with name" + name)
			http.Error(w, "cannot get container with name", http.StatusNotFound)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		if !utils.CanUseImage(imageRes) || !utils.CanUseNetworks(networks) {
			http.Error(w, "Operation not permitted!", http.StatusForbidden)
			logdm.WriteLogLine(method + " the container " + name + "has an image" + imageRes + "that is NOT permeitted")
			utils.ResponcHttpStatus("KO", w)
			return
		}
		
		logdm.WriteLogLine(method + " the container " + name + "has an image" + imageRes + "that is permitted")
		logdm.WriteLogLine(method + " the container " + name + "is on permitted networks")
		requestDelete, err := http.NewRequest("DELETE",dockerString+"/containers/"+name+"?force=true", nil)
		if err != nil {
			logdm.WriteLogLine(method + " error doing the newrequest() delete")
			http.Error(w, "cannot kill the container, error doing the new request DELETE!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		responseDelete, err := httpc.Do(requestDelete)
		if err != nil {
		logdm.WriteLogLine(method + " error doing request")
		http.Error(w, "cannot kill the container, error doing DELETE!", http.StatusInternalServerError)
		utils.ResponcHttpStatus("KO", w)
		return
		}
		
		if responseDelete.StatusCode == 204 {
			isLoggedRequest := utils.LoggingUser(name, keyCkUserName, "kill "+name)
			if !isLoggedRequest {
				logdm.WriteLogLine(method + " request cannot be inserted in db. ERROR")
				http.Error(w, "request is not logged check db!", http.StatusInternalServerError)
				utils.ResponcHttpStatus("KO", w)
				return
			}
			logdm.WriteLogLine(method + "container is killed!")
			utils.ResponcHttpStatus("OK", w)
			return
		} else {
			logdm.WriteLogLine("user :" + keyCkUserName + " has tryed to KILL the container " + name + " but nothing is done.")
			http.Error(w, "cannot stop the container!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
	}
}

func DeleteVolume(w http.ResponseWriter, r *http.Request) {
	method := "DeleteVolume():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	logdm.WriteLogLine(method + "delete  a Volume.")
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	name := r.URL.Query().Get("name")
	
	
	if r.Method == http.MethodDelete {
		keyCkUserName := keycloak.GetUserBytoken(r)
		if keyCkUserName == "" {
			http.Error(w, "no name token!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		logdm.WriteLogLine(method + "delete the volume :"+name)
		isLoggedRequest := utils.LoggingUser(name, keyCkUserName, "delete volume "+name)
		if !isLoggedRequest {
			logdm.WriteLogLine(method + " request cannot be inserted in db. ERROR")
			http.Error(w, "request is not logged check db!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

		requestDelete, err := http.NewRequest("DELETE",dockerString+"/volumes/"+name, nil)
		if err != nil {
			logdm.WriteLogLine(method + " error doing the newrequest() delete")
			http.Error(w, "cannot delete volume, error doing the new request DELETE!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		responseDelete, err := httpc.Do(requestDelete)
		if err != nil {
		logdm.WriteLogLine(method + " error doing request")
		http.Error(w, "cannot delete the Vomume error doing DELETE!", http.StatusInternalServerError)
		utils.ResponcHttpStatus("KO", w)
		return
		}
		fmt.Println("------------------>"+strconv.Itoa(responseDelete.StatusCode))
		if responseDelete.StatusCode == 204 {	
			logdm.WriteLogLine(method + "volume "+name+" is deleted")
			
			utils.ResponcHttpStatus("OK", w)
			return
		} else if responseDelete.StatusCode == 409 {
			logdm.WriteLogLine(method + "volume "+name+" is USED ")
			http.Error(w, "cannot delete the Volume error doing DELETE! voume is in use.", http.StatusForbidden)
			utils.ResponcHttpStatus("OK", w)
			return
		} else  {
			if responseDelete.StatusCode == 404 {
				logdm.WriteLogLine(method + "volume "+name+" not found ")
				http.Error(w, "cannot delete the Volume error doing DELETE not found volume!", http.StatusNotFound)
				utils.ResponcHttpStatus("OK", w)
				return
			} else {
				logdm.WriteLogLine(method + "volume "+name+" internal error!")
				http.Error(w, "cannot delete the Volume error internal error!", http.StatusInternalServerError)
				utils.ResponcHttpStatus("OK", w)
				
				return
			}
		}
	}
}

func GetBindsRfl(w http.ResponseWriter, r *http.Request) {
	method := " GetBindrfl():"
	logdm.WriteLogLine(method + "---------------start request---------------")
	defer logdm.WriteLogLine(method + "---------------stop  request---------------")
	ctx := context.Background()
	logdm.WriteLogLine(method + "list  binds :")
	message, dockerString, canAccess := utils.ControlAccess(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return

	}
	httpc := utils.InitSocket()
	defer httpc.CloseIdleConnections()

	if r.Method == http.MethodGet {

		response, err := httpc.Get(dockerString + "/containers/json")
		if err != nil {
			logdm.WriteLogLine(method + "error! getting list of all container")
			http.Error(w, "cannot get list container", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		var d string
		if response.StatusCode == 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				logdm.WriteLogLine(method + err.Error())
			}
			d = string(b)
		} else {
			d = "something  goes wrong !"
			http.Error(w, "internal error!", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		containers := utils.Containers{}
		err = containers.GetBindsListRfl(d, w)
		if err != nil {
			http.Error(w, "Unable to masrshal json", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}

	}

}