package httpdm

import (
	//"context"
	"fmt"
	"io/ioutil"
	"net/http"
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

func Addstepipeline( w http.ResponseWriter, r *http.Request) {
	method := "Addstepipeline(): "
	logdm.WriteLogLine(method + "-------start-----")
	defer logdm.WriteLogLine(method + "----------------")
	/*
	ctx := context.Background()
	logdm.WriteLogLine(method + "ADDING step to pipeline .")
	message, canAccess := utils.ControlAccessfl(ctx, w, r)
	
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return
	}
	*/
	if r.Method == http.MethodPost {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cannot read json from request", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		strJSON := string(body)
		var steps utils.Pipesteps
		err = steps.AddStepsToPipe(strJSON)
		if err != nil {
			logdm.WriteLogLine(method+"boooooom "+err.Error())
			http.Error(w, "cannot add pipe to step", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		utils.ResponcHttpStatus("OK",w)
	}
}

func RunPipeline( w http.ResponseWriter, r *http.Request) {
	method := "RunPipeline():"
	logdm.WriteLogLine(method + "-------start-----")
	defer logdm.WriteLogLine(method + "----------------")
	/*
	ctx := context.Background()
	message, canAccess := utils.ControlAccessfl(ctx, w, r)
	
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return
	}
	*/
	if r.Method == http.MethodGet {
		pipelineName := r.URL.Query().Get("name")
		userName := utils.GetUserByToken(r)
		err := utils.LaunchPipeline(userName,pipelineName,w)
		if err != nil {
			http.Error(w, "error while launching the pipeline", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		utils.ResponcHttpStatus("OK",w)
	}
}



func Prova( w http.ResponseWriter, r *http.Request) {
	method := "Prova():"
	logdm.WriteLogLine(method + "provaaaa")
	/*
	ctx := context.Background()
	message, canAccess := utils.ControlAccessfl(ctx, w, r)
	if !canAccess {
		http.Error(w, message+"\n", http.StatusForbidden)
		return
	}
	*/
	if r.Method == http.MethodGet {
		token,err := utils.GetToken()
		if err != nil {
			logdm.WriteLogLine(method + "cannot get token by kycloak")
			http.Error(w, "cannot unmarshall json from request", http.StatusInternalServerError)
			utils.ResponcHttpStatus("KO", w)
			return
		}
		req, err := http.NewRequest("GET", "http://keycloak.test/api/dm/list?state=all", nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		req.Header.Add("Authorization", "Bearer "+token)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(string(body))

		utils.ResponcHttpStatus("OK",w)
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

