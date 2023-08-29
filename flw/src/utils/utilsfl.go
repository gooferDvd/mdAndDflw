package utils

import (
	keycloak "example.com/keycloak"
	//"fmt"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	logdm "example.com/logdm"
	"fmt"
	"github.com/lib/pq"
	"net/http"
	"os"
	"strconv"
	"time"
	"bytes"
	"io/ioutil"
	"io"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Pipestep struct {
	Pipeline string `json:"Pipeline"`
	Image    string `json:"Image"`
}
type ifStrutture struct {
	okContainerid int
	koContainerid int
	imageOk string
	imageKo string
	executed string
}


//this structure is used to rappresent the pipeline and his step in execution
type StepExecution struct {
	runName        string 
	pipelineName   string
	pipelineId	   int	
	runId		   int
	exeMap		   map[int]string
	mapIf		   map[int]ifStrutture
	executedMap	   map[int]string
	isPipelineBroke bool
}

type Pipesteps []Pipestep

type ContainerStep struct {
	typeStep       string
	container_id   int
	succ_container int
	composed_by    []int
	next_ok        int
	next_ko        int
	exit_status    int
	pipeline       string
	image_name     string
}

type MessageExitQ struct {
	Name string `json:"Name"`
	ExitStatus    int    `json:"ExitStatus"`
	Pipeline	  string  `json:"Pipeline"`
	ContainerID	  int     `json:"ContainerID"`
}

var (				
	user                 string                 = ""
	password             string                 = ""
	dbname               string                 = ""
	schema               string                 = ""
	host                 string                 = ""
	port, err                                   = strconv.ParseInt(os.Getenv("DB_PORT"), 10, 0)
	keypipeline       map[string]interface{} = make(map[string]interface{})
	dbConn               *sql.DB                = nil //this is the connectionn db , it's opened whem is called the launch function and closed when the pipeline finish.
	pipeTableFields                             = []string{"pipeline_id", "name"}
	containerTableFields                        = []string{"container_id", "precs", "fk_pipeline_id", "next_ok", "next_ko", "image_name"}
	runTableFields								= []string{"run_id","name","\"user\"","exit_status","fk_pipeline_id"}
	runDetailTableFields   						= []string{"run_detail_id","exit_status_container","fk_container_id","fk_run_id"}
)
/////
type VolumeCreate struct {
	MountPo    string `json:"MountPo"`
	Volume     string `json:"Volume"`
	VolumeType string `json:"VolumeType"`
}
type ContainerToCreate struct {
	Image      string `json:"Image"`
	Env		   []string `json:"Env"`
 	Name       string `json:"Name"`
	Network    string `json:"Network"`
	Volumes    []VolumeCreate `json:"Volumes"`
	
}
func ResponcHttpStatus(status string, w io.Writer) {
	var res responceHttp
	res.Info = status
	res.toJSONResponce(w)
}

func (p *responceHttp) toJSONResponce(w io.Writer) error {

	e := json.NewEncoder(w)
	return e.Encode(p)
}
type responceHttp struct {
	Info string
}

//////
func ControlAccessfl(ctx context.Context, w http.ResponseWriter, r *http.Request) (string, bool) {
	method := "ControlAccess(): "

	KEYCLOAK_URL := os.Getenv("KEYCLOAK_URI") + "/realms/" + os.Getenv("KEYCLOAK_REALM")
	idTokenVerifier := keycloak.Keycloak(ctx, os.Getenv("DM_CLIENT_ID"), os.Getenv("DM_CLIENT_SECRET"), KEYCLOAK_URL)
	if !keycloak.TokenVerify(ctx, idTokenVerifier, w, r) {
		logdm.WriteLogLine(method + "error in token !")
		return "error in token !", false
	}
	logdm.WriteLogLine(method+" is ok!.")
	return "keycloak auth ok", true
}

func GetToken() (string, error) {
	method := "GetTokenByKeycloak(): "
	KEYCLOAK_URL := os.Getenv("KEYCLOAK_URI") + "/realms/" + os.Getenv("KEYCLOAK_REALM") + "/protocol/openid-connect/token"

	var token *keycloak.TokenResponse
	token, err := keycloak.GetToken(KEYCLOAK_URL, os.Getenv("DM_CLIENT_ID"), os.Getenv("DM_CLIENT_SECRET"), os.Getenv("KEYCLOAK_USER"), os.Getenv("KEYCLOAK_PASSWORD"))
	if err != nil {
		logdm.WriteLogLine(method + " error getting token from keycloak ")
		return "", err
	}
	logdm.WriteLogLine(method + " getting token from keycloak ......")
	return token.AccessToken, nil
}

func setEnv() {
	if host == "" && user == "" && password == "" && dbname == "" && schema == "" {
		host = os.Getenv("DB_HOST")
		user = os.Getenv("DB_USER")
		dbname = os.Getenv("DB_NAME")
		schema = os.Getenv("DM_DB_SCHEMA")
		password = os.Getenv("DB_PASS")
	}
}

func PrintEnv() {
	method := "printEnv():"
	logdm.WriteLogLine(method + "host=" + host + " user=" + user + " schema=" + schema)
}
/*
this is the entrypoint from endpoint run pipeline
*/
func LaunchPipeline(userName string, name string, w http.ResponseWriter) error {
	method := "LaunchPipeline(): "
	logdm.WriteLogLine (method+"------------------Start--------------------")
	defer logdm.WriteLogLine (method+"------------------Stop---------------------")
	err := openDB()
	defer closeIt()
	if err != nil {
		logdm.WriteLogLine(method + "error while opening the db")
		return err
	}
	//load the pipeline from db in a map cache
	err = fillThePipelineMap()
	if err != nil {
		logdm.WriteLogLine("error during adding filling the map " + err.Error())                 
		return err
	}
	if !isKeyPresent(name) {
		logdm.WriteLogLine(method + "the pipeline does not exists")
		return errors.New("the pipeine not exists")
	}

	var stepExe StepExecution 
	// get the pipeline runname
	runName := GetpipelineRunName(name)
	//get the pipeline id by name
	err,pipeline_id := getPipelineIdByName(name)
	if err != nil {
		return err
	}
	////this is the pipeline rules name
	stepExe.pipelineName = name
	// id of the run pipeline in run table
	//stepExe.runId = seqRunId[0]
	//this is the runpipeline rules name
	stepExe.runName = runName
	// the id of the pipeline in pipeline table
	stepExe.pipelineId = pipeline_id
	stepExe.isPipelineBroke = false
	//set the exe map, with roots of the pipeline and launch the root
	err = stepExe.setRootsNode(userName)
	if err != nil {
		return err
	}
	err = stepExe.run()
	if err != nil {
		return err
	}
	
	return nil

}
func (p *StepExecution) setRootsNode (userName string) error {
	method := "setRootsNode(): "
	logdm.WriteLogLine (method+"------------------Start--------------------")
	defer logdm.WriteLogLine (method+"------------------Stop---------------------")
	/*
	check if pipeline starts with more root,n
	getRootPipeline return a map  with key the id of container and value the image of the 
	node with key k.
	*/
	err,rootNode := getRootPipeline(p.pipelineName)
	n := len(rootNode)
	logdm.WriteLogLine (method+" this pipeline "+p.runName+" have "+strconv.Itoa(n)+" roots")
	//get one seq for run entry
	seqRunId,err := getNextSeqId("run",1)
	if (err != nil) {
		return err
	}
	// get n seq for run_detail each for roots.
	seqRunDetailId,err := getNextSeqId("run_detail",n)
	if (err != nil) {
		return err
	}
	// set the id of the runtable for th pipeline
	p.runId = seqRunId[0]
	// insert  record in run table for the pipeline
    err = insertIntoDB("pipelines.run",runTableFields,seqRunId[0],p.runName,userName,-1,p.pipelineId)
	if err != nil {
		return err
	}
	j:=0
	// for every rootNode insert a record in run detail with initial 
	for k := range rootNode {
		err = insertIntoDB("pipelines.run_detail",runDetailTableFields,seqRunDetailId[j],-1,k,seqRunId[0])
		if err != nil {
			return err
		
		}
		j++
	}
	p.executedMap = make(map[int]string)
	p.exeMap =  make(map[int]string)
	//build the i?fmap of the pipeline the key is the id on container table of the if record, 
	// the value is a struct with imageok,imagenok the nextok e nextko. 
	err,p.mapIf = getIfNode(p.pipelineId)
	
    if err != nil {
		return err
	}
	//set the exeMap and launch it
    for k := range rootNode {
		p.exeMap[k]="L"
		err := p.launchContainer(k,rootNode[k])
		if err != nil {
			return err
		}
	}
	p.logTheMap("exe")
	p.logTheMap("if")
	return nil


}
/*
log the map print value
*/
func (p StepExecution) logTheMap (mapName string) {
	
	method := "logTheMap(): "
	if mapName=="exe" {
		logdm.WriteLogLine(method+"==============ExeMap===================")
		for k,v := range p.exeMap {
			logdm.WriteLogLine(method+ "key: "+strconv.Itoa(k)+" value: "+v)
		}
		logdm.WriteLogLine(method+"======================================")
	}
	if mapName=="if" {
		logdm.WriteLogLine(method+"==============ifMap===================")
		for k,v := range p.mapIf {
			logdm.WriteLogLine(method+ "key: "+strconv.Itoa(k)+" value: "+"{ imageKo:"+v.imageKo+" imageOk:"+v.imageOk+" }")
		}
		logdm.WriteLogLine(method+"======================================")
	}
	if mapName=="executed" {	
		logdm.WriteLogLine(method+"==========exeCutedMap===================")
		for k,v := range p.executedMap {
			logdm.WriteLogLine(method+ "key: "+strconv.Itoa(k)+" value: "+v)
		}
		logdm.WriteLogLine(method+"======================================")
	}

}

func ( p *StepExecution)  run() error  {
	method := "run(): "
	logdm.WriteLogLine (method+"------------------Start--------------------")
	defer logdm.WriteLogLine (method+"------------------Stop---------------------")
	err := openDB()
	if err != nil {
		logdm.WriteLogLine(method + "error while opening the db")
		return err
	}

	for ( !p.isPipelineBroke && len (p.exeMap)!=0 ){
		logdm.WriteLogLine(method+ "lunghezza della exeMap is "+strconv.Itoa(len(p.exeMap)) )
		err:= p.checkIfSomeoneFinished()
		if err != nil {
			
			return err 
		}
	}
	exitStatusPipeline := -1
	if (!p.isPipelineBroke) {
		logdm.WriteLogLine(method+"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
		logdm.WriteLogLine(method+" pipeline with name "+p.pipelineName+ " is FINISHED.")
		logdm.WriteLogLine(method+"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
		exitStatusPipeline = 0
	} else {
		logdm.WriteLogLine(method+"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
		logdm.WriteLogLine(method+" pipeline with name "+p.pipelineName+ " is FINISHED in BROKEN STATE.")
		logdm.WriteLogLine(method+"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
		exitStatusPipeline = 1
	}
	//update the run table with the exit status of the pipeline
	err = updateRowintoDB("pipelines.run",[]string{"exit_status"},"name='"+p.runName+"'",strconv.Itoa(exitStatusPipeline))
	if err != nil {
		return err
	}
	//close the pipeline Queue
	err = p.closePipelineQueue()
	p.logTheMap("exe")
	p.logTheMap("executed")
	p.logTheMap("if")
	if err != nil {
		return err
	}
	return nil
}
func (p *StepExecution) checkIfSomeoneFinished() error {
	method := " checkIfSomeoneFinished(): "
	logdm.WriteLogLine (method+"------------------Start--------------------")
	defer logdm.WriteLogLine (method+"------------------Stop---------------------")
	
	//controlla se il job ha finito un container in esecuzione si chiama con nome_run-id_container per il container k che era in esecuzione
	//riporta l'exit status
			
	err,exitStatus,k :=p.CheckContainerExitStatusByQ()
	if err != nil {
		
		return err
	}
	p.logTheMap("exe")
	p.logTheMap("executed")
	p.logTheMap("if")
	if _,ok := p.mapIf[k]; ok {
		logdm.WriteLogLine(method+" node "+strconv.Itoa(k)+" terminated ,is an if")
	}
	logdm.WriteLogLine (method +" container id "+strconv.Itoa(k)+"is finished delete it from map and update exit status")
	//cancella il nodo k
	delete(p.exeMap,k)
	p.executedMap[k]="execute"
	
	err = updateRowintoDB("pipelines.run_detail",[]string{"exit_status_container"},"fk_container_id="+strconv.Itoa(k)+" and fk_run_id="+strconv.Itoa(p.runId),strconv.Itoa(exitStatus))
	if err != nil {
		return err
	}
	
	//sei il nodo k e' una if controlla exit status e lancia le immagini nel caso che finisca bene o male come 
	if _,ok := p.mapIf[k]; ok   {
		logdm.WriteLogLine (method+" the node "+strconv.Itoa(k)+" is an ifnode")
		logdm.WriteLogLine (method+ " this is an if CHOOOOISSSSE ...............................................")
		if exitStatus == 0 {
			imageok := p.mapIf[k].imageOk 
			idok := p.mapIf[k].okContainerid
			seqRunDetailId,err := getNextSeqId("run_detail",1)
		   	err = insertIntoDB("pipelines.run_detail",runDetailTableFields,seqRunDetailId[0],-1,idok,p.runId)
			if err != nil {
				return err
			}
			p.exeMap[idok]="L"
			p.launchContainer (idok,imageok)

		} else {
			imageko := p.mapIf[k].imageKo 
			idko := p.mapIf[k].koContainerid
			seqRunDetailId,err := getNextSeqId("run_detail",1)
		   	err = insertIntoDB("pipelines.run_detail",runDetailTableFields,seqRunDetailId[0],-1,idko,p.runId)
			if err != nil {
				return err
			}
			p.exeMap[idko]="L"
			p.launchContainer (idko,imageko)
		}
		return nil
	} else {
		/*
		se il nodo non e' una if
		*/
		logdm.WriteLogLine (method+" the node "+strconv.Itoa(k)+" is NOT an ifnode")
		//rompi subito la pipeline
	    if exitStatus != 0 {
			logdm.WriteLogLine(method + "container id for pipeline "+strconv.Itoa(p.runId)+"has failed Broken pipeline")
			p.isPipelineBroke = true
			return errors.New("broke pipeline exit code not zero")
		}
		//ottieni i figli del nodo k
		err,next := getNextPipeline(p.pipelineName,k)
		if err != nil {
			return err
		}
		/*
			E decidi quale figlio puo' essere eseguito pulendo lA MAPPA DEI SUCCESSIVI DI QUELLI CHE NON POSSONO ESSERE ESEGUITI perche si trovano su rami morti.
		*/
		err = p.cleankIfNextHaveFatherInExecution(&next,k)
		if err != nil {
			return err
		}
		// esegui i figli che possono essere essere eseguiti, ossia che hanno padri su rami morti, e che non hanno padri in esecuzione
		for toExe,image := range next {
			seqRunDetailId,err := getNextSeqId("run_detail",1)
			if (err != nil) {
				return err
			}
			err = insertIntoDB("pipelines.run_detail",runDetailTableFields,seqRunDetailId[0],-1,toExe,p.runId)
			if err != nil {
				return err
			}
			p.exeMap[toExe]="L"
			p.launchContainer (toExe,image)

		}
	}
	return nil
}
/*
Se un figlio ha tutti i  padri che non verranno mai eseguiti, ossia tutti i padri sono su rami morti
allora lo posso eseguire . allfather e' il nodo che e' stato eseguito ora (k) , next sono i suoi figli
*/
func (p StepExecution) cleankIfNextHaveFatherInExecution(next *map[int]string,allFather int ) error {
	method := "cleankIfNextHaveFatherInExecution(): "
	logdm.WriteLogLine (method+"------------------Start--------------------")
	defer logdm.WriteLogLine (method+"------------------Stop---------------------")
	//per ogni figlio del nodo allfather (k) ottieni i padri
	// next ci sono i figli figli del nodo appena terminato allfather
	//keychild e' un figlio di allfather
	
	for keyChild := range *next {
		err,fatherKeyChild := getFatherForAchild(keyChild)
		if err != nil {
			return err
		}
		//leva dalla mappa il nodo da cui provengo perche sicuramente e' stato eseguito ossia allFather
		logdm.WriteLogLine (method+" node "+strconv.Itoa(keyChild)+" has " + strconv.Itoa(len(fatherKeyChild)) + "  father but this <" + strconv.Itoa(allFather) + "> has been sure executed delete it from the father list")
		delete(fatherKeyChild,allFather)
		
		var isFatherExecuted bool
		/*per ogni padre controlla che sia stato eseguito o che sia in esecuzione 
		se e' cosi leva subito il figlio dalla mappa delle esecuzioni. 
		*/
		for keyFather := range fatherKeyChild { 
			logdm.WriteLogLine (method+"node child "+ strconv.Itoa(keyChild) +" has father " + strconv.Itoa(keyFather))
			//
			err,isFatherExecuted = isNextFatherExecuted(keyFather,p.runId)
			if err != nil {
				return err
			}
			// se il il padre e' non e' stato eseguito
			if  !isFatherExecuted {
				
				if _,ok := p.exeMap[keyFather]; ok {// se il padre e' in esecuzione non sei su ramo morto e devi aspettare che finisca quindi cancella il figlio dalla mappa delle esecuzioni  , non si puo eseguire perche ha un padre che e' in esecuzione
					logdm.WriteLogLine(method+" keyfather "+strconv.Itoa(keyFather)+" is in execution")
					delete (*next,keyChild)
					logdm.WriteLogLine(method+" node "+ strconv.Itoa(keyChild) +" is removed because have father in execution with id "+ strconv.Itoa(keyFather))
					/////////////////break
				} else {
					/* se il padre non e' stato ancora stato eseguito e non e' in esecuzione devo controllare che non e'  su un ramo morto, perche se il ramo e' morto allora 
					posso eseguire keychild altrimenti lo devi levare dalla mappa degli eseguibili.
					*/
					err,fatherDeadBranch := p.isFatherOnDeadBranch(keyFather)
					if err != nil {
						return err
					}
					if fatherDeadBranch {
						//se il padre e'su un ramo morto allora lascia il figlio nella mappa di esecuzione
						logdm.WriteLogLine(method+ " the node "+strconv.Itoa(allFather)+ " have child "+strconv.Itoa(keyChild)+" with father in deadbranch "+strconv.Itoa(keyFather)+" so THE CHILD can be executed.")
					}else {
						// se il padre non e' su un ramo morto, allora il figlio non e' da eseguire lo levo dalla mappa dei figli da eseguire e breaak(?) 
						logdm.WriteLogLine(method+ " the node "+strconv.Itoa(allFather)+ " have child "+strconv.Itoa(keyChild)+" with father Not Executed in a NOT in deadbranch."+strconv.Itoa(keyFather)+" must be execute WAIT IT . delete the CHILD from exe map.")
						delete (*next,keyChild)
						//////////////////////////////break //?

					}
				}
			} else {//
				logdm.WriteLogLine(method+" node "+ strconv.Itoa(keyChild)+" has father "+strconv.Itoa(keyFather)+" Executed .")
			}
		}

	}
	return nil
}
/*
un padre che non e' stato eseguito  si puo trascurare se vogliamo eseguire un suo figlio se i suoi padri sono
trascurabili per la sua esecuzione. un figlio e' su un deadbranch se tutti i suoi padri sono su un deadbranch
*/
func (p StepExecution) isFatherOnDeadBranch( father int ) (error,bool) {
	method :="isFatherInDeadBranch(): "
	/*
	father is non in execution at this point 
	*/
	logdm.WriteLogLine(method+"&&&&&&&&&&&&&&&&&&&&&START&&&&&&&&&&&&&&&&&&&&&&&&&&&")
	defer logdm.WriteLogLine(method+"&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&")
	// prendi i padri del padre corrente
	err,nextFather := getFatherForAchild(father)
	if err != nil {
		return err,false
	}
	//se non ci sono hai raggiunto la radice e non e' un ramo morto
	if len(nextFather) == 0{
		logdm.WriteLogLine(method+" root node reached ! non on dead branch!.")
		return nil,false
	}
	if err != nil {
		logdm.WriteLogLine (method + " error getting nextFatherMap "+err.Error())
		return err,false
	}
	//questa e' la mappa che indichera per ogni padre se e' su un ramo morto o no. un nodo figlio e' su un ramo morto se  tutti i padri sono su rami morti
	isDeathBranchMap := make(map[int]bool)
	// per ogni padre del padre corrente
	for key,_ := range nextFather {
		// se e' una if e e' stato eseguito ho una sequenza nodo non eseguito e poi if eseguita ; e' un ramo morto
		if _,ok := p.mapIf[key]; ok && p.executedMap[key]=="execute"{
			logdm.WriteLogLine (method+ " found an if that is been execute! this node "+ strconv.Itoa(key)+"  is on  dead branch ")
		
			isDeathBranchMap[key]=true
		} else if p.executedMap[key]!="execute" && p.exeMap[key] != "L" {// se non e' stato eseguito e non e' in esecuzione continua la ricorsione
			logdm.WriteLogLine (method+" FOUND A NODE NODEeeeeeeeeeee THAT IS NOT BEEN EXECUTED GO ON.."+strconv.Itoa(key))
			err,result := p.isFatherOnDeadBranch(key)
			if err != nil {
				logdm.WriteLogLine (method + " error "+err.Error())
				return err,false
			}
			//valorizza la mappa dei padri
			isDeathBranchMap[key] = result
		} else {
			logdm.WriteLogLine (method+" FOUND A NODE NODE THAT IS BEEN EXECUTED or in EXECUTION ; "+strconv.Itoa(key)+" STOP this is node is not on dead branch ")
			//isDeathBranchMap[key] = false
			//this can be return false try
			return nil,false
		}
		
	}
	/*
	*/
	logdm.WriteLogLine(method+"||||||||||||||||||||||| father map  for node: "+strconv.Itoa(father)+" ||||||||||||||||||||")
	resusultBranch := true
	for k,v := range isDeathBranchMap {
		resusultBranch = resusultBranch && v
		if v {
			logdm.WriteLogLine(method+ "father node "+strconv.Itoa(k) +"is on dead branch")
		} else {
			logdm.WriteLogLine(method+ "father node "+strconv.Itoa(k) +"NOT on dead branch")
		}
	}
	logdm.WriteLogLine(method+"||||||||||||||||||||||||| result node ||||||||||||||||||||||||||||||||||||||||||||")
	if resusultBranch {
		logdm.WriteLogLine(method+ " node "+ strconv.Itoa(father) + " is on deathBranch")
	} else {
		logdm.WriteLogLine(method+ " node " + strconv.Itoa(father) + " is NOT on deathBranch")
	}
	logdm.WriteLogLine(method+"||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||")
	return nil,resusultBranch
}

func (p StepExecution) closePipelineQueue () error {
	method :="closePipelineQueue(): "
	logdm.WriteLogLine (method+"------------------Start--------------------")
	defer logdm.WriteLogLine (method+"------------------Stop---------------------")
	amqpServerURL := os.Getenv("DM_AMQP_SERVER_URL")
	connMQ, err := amqp.Dial(amqpServerURL)
    if err != nil {
        logdm.WriteLogLine(method + "error during the connection to RABBITMQ : "+err.Error()+" "+amqpServerURL )
        return err
    }
	defer connMQ.Close()
	// apri un canale
	channel, err := connMQ.Channel()
	if err != nil {
		logdm.WriteLogLine(method + "Failed to open a channel: "+err.Error())
	}
	_, err = channel.QueueDelete(
		p.runName, // name of the queue
		false,     // if unused
		false,     // if empty
		false,     // noWait
	)
	if err != nil {
		logdm.WriteLogLine (method+" error while closing the queue  "+p.pipelineName + err.Error())
		return err
	}
	logdm.WriteLogLine(method+" the QUEUE "+ p.runName + " is closed!!!!!!!")
	return nil
}
 /*
	create the queue con il run Name e rimani in attesa del messaggio dal container json.

 */
func (p StepExecution)  CheckContainerExitStatusByQ() (error,int,int) {
	method :="CheckContainerExitStatusByQ(): "
	logdm.WriteLogLine (method+"------------------Start--------------------")
	defer logdm.WriteLogLine (method+"------------------Stop---------------------")
	amqpServerURL := os.Getenv("DM_AMQP_SERVER_URL")
	connMQ, err := amqp.Dial(amqpServerURL)
	exitStatus := -1
    if err != nil {
        logdm.WriteLogLine(method + "error during the connection to RABBITMQ : "+err.Error()+" "+amqpServerURL )
        return err,exitStatus,-1
    }
	defer connMQ.Close()
	// apri un canale
	channel, err := connMQ.Channel()
	if err != nil {
		logdm.WriteLogLine(method + "Failed to open a channel: "+err.Error())
	}
	//args := amqp.Table{"x-message-ttl": int32(20000)}
	queue, err := channel.QueueDeclare(
		p.runName, // sostituisci con il nome della tua coda
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		logdm.WriteLogLine(method + "Failed to declare a queue: " + err.Error())
		return err,exitStatus,-1
	}

	// consuma dalla coda
	msgs, err := channel.Consume(
		queue.Name, // queue
		"",     // consumer
		false,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		logdm.WriteLogLine( method + "Failed to register a consumer: " +err.Error())
		return err,exitStatus,-1
	}
	
	logdm.WriteLogLine(method+ " waiting for message on Queue "+p.runName+".......")
	//aspetta un messaggio si sblocca il canale quando arriva.
	msg := <-msgs
    msg.Ack(false)
	logdm.WriteLogLine(method + " received a message ! on queue "+p.runName) 

	var message MessageExitQ
	err = json.Unmarshal(msg.Body, &message)
	if err != nil {
		logdm.WriteLogLine(method+ "Error decoding container json message "+err.Error())
	return err,exitStatus,-1
	
	}
	

	logdm.WriteLogLine (method+ "****************************************************************************************")
	logdm.WriteLogLine (method+ "container with name"+message.Name+" had terminate with exitstatus " +strconv.Itoa(message.ExitStatus))
	logdm.WriteLogLine (method+ "****************************************************************************************")
	exitStatus = message.ExitStatus
	
	logdm.WriteLogLine(method+" container id message is :  " + strconv.Itoa(message.ContainerID))

	return nil,exitStatus,message.ContainerID

}

/*
launch a container with a image  calling docker-manager ep, set the ENV in container for send responce.
*/

func (p StepExecution) launchContainer (containerId int ,image string) error {
	method := "launchContainer():"
	logdm.WriteLogLine (method+"------------------Start--------------------")
	defer logdm.WriteLogLine (method+"------------------Stop---------------------")
	
	logdm.WriteLogLine(method + "launch container whith image "+image)
	
	token,err := GetToken()
	if err != nil {
		logdm.WriteLogLine(method + "cannot get token by kycloak")
		return err
	}
	containerName := p.runName+"-"+strconv.Itoa(containerId)
	logdm.WriteLogLine(method + "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	logdm.WriteLogLine(method + "launch container with name "+ containerName + " with id "+ strconv.Itoa(containerId))
	logdm.WriteLogLine(method + "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	
	ampExternalURL := os.Getenv("AMQP_EXTERNAL_URL")
	var volume []VolumeCreate
	var envVar []string
	var container ContainerToCreate
	// set property for container to create (OCHIO al parametro network che deve essere preso da env)
	container.Name= p.runName+"-"+strconv.Itoa(containerId)
	container.Image = image
	container.Network =os.Getenv("DOCKER_FLOW_NET")
	keycloak_dm_create_uri:=os.Getenv("KEYCLOAK_URI_DM_CREATE")
	container.Volumes = volume
	//set the env for container job with id  containerId on db  
	envVar = append(envVar, "QUEUE_FLOW="+p.runName)
	envVar = append(envVar, "CONTAINER_NAME="+containerName)
	envVar = append(envVar,"SERVER_QE="+ampExternalURL)
	envVar= append(envVar,"CONTAINER_ID="+strconv.Itoa(containerId))
	container.Env = envVar
	//do the call to docker manager
	requestBody,err := json.Marshal(container)	
	fmt.Println(string(requestBody))
	req, err := http.NewRequest("POST",keycloak_dm_create_uri,  bytes.NewBuffer(requestBody))
	if err != nil {
		logdm.WriteLogLine(method+ "error while launching the container . " + err.Error())
		fmt.Println(err)
		return err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()
  
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(string(body))
	return nil
}

func (p *Pipesteps) AddStepsToPipe(d string) error {
	method := "AddStepsTopipe(): "
	logdm.WriteLogLine (method+"------------------Start--------------------")
	defer logdm.WriteLogLine (method+"------------------Stop---------------------")
	err := json.Unmarshal([]byte(d), p)
	if err != nil {
		logdm.WriteLogLine(method + "cannot unmarshal json to struct!")
		return errors.New("errore nella codifica json!")
	}
	more := len(*p)
	if more > 1 {
		logdm.WriteLogLine (method+" adding more steps to po pipeline")
		p.addStepsInParallel()
	} else {
		logdm.WriteLogLine(method + "adding one element to pipeline")
		err := (*p)[0].AddStepTopipe()
		if err != nil {
			logdm.WriteLogLine(method + " error adding step to pipeline " + err.Error())
			return err
		}
	}
	return nil
}

func (p *Pipesteps) areStepForSamePipeline() (error, bool) {
	method:= "areStepForSamePipeline(): "
	logdm.WriteLogLine (method+"------------------Start--------------------")
	defer logdm.WriteLogLine (method+"------------------Stop---------------------")
	errdb := openDB()
	if errdb != nil {
		return errdb, false
	}
	same := false
	mapname := make(map[string]string)
	for _, k := range *p {
		mapname[k.Pipeline] = "x"
	}
	if len(mapname) == 1 {
		same = true
	}
	logdm.WriteLogLine(method+" all steps are for the same pipeline")
	return nil, same
}

func (p Pipesteps) addStepsInParallel() error {
	method := "addInParallel(): "
	logdm.WriteLogLine (method+"------------------Start--------------------")
	defer logdm.WriteLogLine (method+"------------------Stop---------------------")
	errdb := openDB()
	if errdb != nil {
		return errdb
	}
	if _, ok := p.areStepForSamePipeline(); !ok {
		logdm.WriteLogLine("adding more steps not for the same pipeline ")
		return errors.New("adding steps for different pipeline")

	}
	err = fillThePipelineMap()
	if err != nil {
		logdm.WriteLogLine("error during adding filling the map " + err.Error())
		return err
	}
	
	pipelineName := p[0].Pipeline
	
	if !isKeyPresent(pipelineName) {
	
		n := len(p)
		nextPipeline, err := getNextSeqId("pipeline", 1)
		if err != nil {

			logdm.WriteLogLine("error during adding step to pipeline" + err.Error())
			return err
		}
		
		logdm.WriteLogLine (method+ "adding pipeline "+pipelineName+ " starting with "+strconv.Itoa(n) +" node.")
		err = insertIntoDB("pipelines.pipeline", pipeTableFields, nextPipeline[0], pipelineName)

		if err != nil {
			logdm.WriteLogLine(method + "error in insert elem pipeline" + err.Error())
			return err
		}
		nextContainer, err := getNextSeqId("container", n)
		if err != nil {
			logdm.WriteLogLine(method + "error gettin n seq" + err.Error())
			return err
		}

		for i, k := range p {
			err = insertIntoDB("pipelines.container", containerTableFields, nextContainer[i], nil, nextPipeline[0], nil, nil, k.Image)
			if err != nil {
				logdm.WriteLogLine(method + "error in insert elem pipeline" + err.Error())
				return err
			}
		}
		logdm.WriteLogLine(method + " inserted "+strconv.Itoa(n)+" record with precs to null")
		return nil
	}
	
	err, endPipeline := getEndPipeline(pipelineName)
	if err != nil {
		return err
	}
	err, pipelineId := getPipelineIdByName(pipelineName)
	if err != nil {
		return err
	}
	if len(endPipeline) == len(p) {
		n := len(p)
		nextContainer, err := getNextSeqId("container", n)
		if err != nil {
			return err
		}

		for i, k := range p {
			array := fmt.Sprintf("{%d}", endPipeline[i])
			err = insertIntoDB("pipelines.container", containerTableFields, nextContainer[i], array, pipelineId, nil, nil, k.Image)
			if err != nil {
				logdm.WriteLogLine(method + "error in insert elem pipeline" + err.Error())
				return err
			}
		}
		logdm.WriteLogLine(method+ "added "+ strconv.Itoa(n) +"to"+ strconv.Itoa(n) +"elements to the pipeline")
		return nil
	} else if len(endPipeline) == 1 {
		n := len(p)
		nextContainer, err := getNextSeqId("container", n)
		if err != nil {
			return err
		}
		for i, k := range p {
			err = insertIntoDB("pipelines.container", containerTableFields, nextContainer[i], pq.Array(endPipeline), pipelineId, nil, nil, k.Image)
			if err != nil {
				logdm.WriteLogLine(method + "error in insert elem pipeline" + err.Error())
				return err
			}
		}
		logdm.WriteLogLine(method+ "added 1 to"+ strconv.Itoa(n) +"elements to the pipeline")
		return nil
	} else  {
		logdm.WriteLogLine(method+ " number of adding node is not equals to the number node of the last element in pipeline adding it maually")
		return errors.New("number of adding node is not equals to the number node of the last element in pipeline")

	}
}

func (p *Pipestep) AddStepTopipe() error {
	method := "AddStepTopipe(): "
	logdm.WriteLogLine (method+"------------------Start--------------------")
	defer logdm.WriteLogLine (method+"------------------Stop---------------------")
	errdb := openDB()
	if errdb != nil {
		return errdb
	}
	logdm.WriteLogLine(method + "retried info on pipeline step: name: " + p.Pipeline + " image:" + p.Image)
	err = fillThePipelineMap()
	if err != nil {
		logdm.WriteLogLine("error during adding filling the map " + err.Error())
		return err
	}
	nextPipeline, err := getNextSeqId("pipeline", 1)
	if err != nil {

		logdm.WriteLogLine("error during adding step to pipeline" + err.Error())
		return err
	}

	nextContainer, err := getNextSeqId("container", 1)
	if err != nil {
		logdm.WriteLogLine("error during adding step to container" + err.Error())
		return err
	}

	if isKeyPresent(p.Pipeline) {
		logdm.WriteLogLine(method + "pipeline is already defined. ")
		err, endPipeline := getEndPipeline(p.Pipeline)
		if err != nil {
			return err
		}
		err, pipelineId := getPipelineIdByName(p.Pipeline)
		if err != nil {
			return err
		}
		err = insertIntoDB("pipelines.container", containerTableFields, nextContainer[0], pq.Array(endPipeline), pipelineId, nil, nil, p.Image)
		if err != nil {
			logdm.WriteLogLine(method + "error in insert elem pipeline" + err.Error())
			return err
		}
		return nil
	} else {
		logdm.WriteLogLine(method + "pipeline is NoT already defined. ")

		err := insertIntoDB("pipelines.pipeline", pipeTableFields, nextPipeline[0], p.Pipeline)
		if err != nil {
			logdm.WriteLogLine(method + "error in insert elem pipeline" + err.Error())
			return err
		}

		err = insertIntoDB("pipelines.container", containerTableFields, nextContainer[0], nil, nextPipeline[0], nil, nil, p.Image)
		if err != nil {
			logdm.WriteLogLine(method + "error in insert elem Container" + err.Error())
			return err
		}
		return nil
	}
}

func GetpipelineRunName(name string) string {
	method := "GetpipelineRunName() :"
	logdm.WriteLogLine (method+"------------------Start--------------------")
	defer logdm.WriteLogLine (method+"------------------Stop---------------------")
	currentTime := time.Now()
	formattedDate := currentTime.Format("20060102_150405")
	return name + "_" + formattedDate
}

func GetUserByToken(r *http.Request) string {
	method:="GetUserByToken(): "
	user:=keycloak.GetUserBytoken(r)
	logdm.WriteLogLine (method+ " username is "+user)
	return user
} 
