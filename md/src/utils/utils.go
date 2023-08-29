package utils

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	keycloak "example.com/keycloak"
	logdm "example.com/logdm"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"reflect"
	"time"
	amqp "github.com/rabbitmq/amqp091-go"
)

/////////////////////////////////////////////////////// pure utility
func InitSocket() http.Client {
	method := "initSocket():"
	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
	}
	logdm.WriteLogLine(method + " getting docker socket... ")
	return httpc
}
func GetDockerString() (bool, string) {
	var vers string
	vers = os.Getenv("DOCKER_VERSION")
	if vers == "" {
		return false, "xxx"
	} else {
		conn := "http://127.0.0.1/" + vers
		return true, conn
	}
}
func ControlAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) (string, string, bool) {
	method := "ControlAccess()"
	isPresentDockerVers, dockerString := GetDockerString()
	if !isPresentDockerVers {
		logdm.WriteLogLine(method + "the docker version var is not set!")
		return "the docker version var is not set!", "xxx", false

	}
	KEYCLOAK_URL := os.Getenv("KEYCLOAK_URI") + "/realms/" + os.Getenv("KEYCLOAK_REALM")
	idTokenVerifier := keycloak.Keycloak(ctx, os.Getenv("DM_CLIENT_ID"), os.Getenv("DM_CLIENT_SECRET"), KEYCLOAK_URL)
	if !keycloak.TokenVerify(ctx, idTokenVerifier, w, r) {
		logdm.WriteLogLine(method + "error in token !")
		return "error in token !", "xxx", false
	}
	return "keycloak auth ok", dockerString, true
}

func CleanString(r rune) rune {

	if r == 1 || r == 3 || r == 2 || r == 0 || r > unicode.MaxASCII {
		return -1
	}
	return r
}
func createRmqMessage (username string,container string,action string) (string,error) {
	method := "CreateRmqMessageJson():"
	queueMsg := make(map[string]interface{})
	queueMsg["User"]=username
	queueMsg["Container"]=container
	queueMsg["Action"]=action
	jsonRmqMsg, err := json.Marshal(queueMsg)

	if err != nil {
		logdm.WriteLogLine(method+ "error while mashalling map to json")
		return "",err
	}
	jsonMessage := string(jsonRmqMsg)
	logdm.WriteLogLine(method+ "generate Json for message->"+jsonMessage)
	return jsonMessage,nil

}
func LoggingUser(username string, containername string, operation string) bool {
	method := "LoggingUser():"
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASS")
	dbname := os.Getenv("DB_NAME")
	port, err := strconv.ParseInt(os.Getenv("DB_PORT"), 10, 0)
	schema := os.Getenv("DM_DB_SCHEMA")

	amqpServerURL := os.Getenv("DM_AMQP_SERVER_URL")
	queueName := os.Getenv("DM_AMQP_NAME")

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable search_path=%s", host, port, user, password, dbname, schema)
	psqlInfoLog := fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable search_path=%s", host, port, user, password, dbname, schema)
	logdm.WriteLogLine(method + psqlInfoLog)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		logdm.WriteLogLine(method + "error opening the db")
		fmt.Print(err)
		return false
	}
	err = db.Ping()
	if err != nil {
		logdm.WriteLogLine(method + "db in not responding")
		return false
	}

	defer db.Close()
     
	sqlStatement := `insert into containers.container(id,containername,timestamp,username,operation) values ((select nextval('containers.container_id_seq')), $1, current_timestamp,$2, $3) returning id`
	logdm.WriteLogLine(method + "executing this stm on db : " + sqlStatement)
	id := 0

	err = db.QueryRow(sqlStatement, containername, username, operation).Scan(&id)
	if err != nil {
		logdm.WriteLogLine(method + "error inserting the access record")
		return false
	}
	logdm.WriteLogLine(method + "New record ID is inserted:" + strconv.Itoa(id))

	//rabbit 
	connMQ, err := amqp.Dial(amqpServerURL)
	if err != nil {
		logdm.WriteLogLine(method + "error during the connection to RABBITMQ : "+amqpServerURL )
		return false
	}
	defer connMQ.Close()
	ch,err := connMQ.Channel()
	if err != nil {
		logdm.WriteLogLine(method + "error opening channel to RabbitMQ")
		return false
	}
	defer ch.Close()
	q,err := ch.QueueDeclare(
		queueName, // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		logdm.WriteLogLine(method + "error declaring queue name  to RabbitMQ")
		return false
	}
	ctxMq, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	body,err := createRmqMessage(username,containername,operation )
	if err != nil {
		logdm.WriteLogLine(method + "error creating json message")
		return false 
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
		logdm.WriteLogLine(method + "error publishing a message  to RabbitMQ")
		return false
	}
	logdm.WriteLogLine(method + "message ( "+body+") has been sent "+" to Rabbitmq Queue "+queueName)
	//

	return true
}

//// use in httdm/containercreate
func CheckContainerCreateCond(volume string, mount string, volumeType string) (bool, string) {
	if !(volumeType == "Bind" || volumeType == "Volume" ) {
		message := "ERROR IN JSON create TYPE CONTAINER !"
		return false, message
	}
	
	if volumeType == "Bind" && !strings.HasPrefix(volume, "/") {
		message := " source bind path must be absolute"
		return false, message

	}
	if volumeType == "Volume" && strings.HasPrefix(volume, "/") {
		message := "volume name cannot start with /"
		return false, message
	}
	
	return true, "ok"

}

/////////////////////////////////////////////////////// used struct
///////this for create a container with volume (from struct to json)
type NetworkingConfig struct {
	EndpointsConfig map[string]interface{} `json:”EndpointsConfig”`
}
type ToJsonContainer struct {
	Image            string                 `json:”Image”`
	Env              []string				`json:”Env”`
	HostConfig       map[string]interface{} `json:”HostConfig"`
	NetworkingConfig NetworkingConfig       `json:”NetworkingConfig”`
}

////////struct to create a volume

type VolumeToCreate struct {
	Name string `json:"Name"`
	//Size  string  `json:"Size"`
}
/////// struct list bind volume
type BindVolume struct {
	Destination string `json:"Destination"`
	RW bool `json:"RW"`
	Source string `json:"Source"`

}
type ContainerWithBinds struct {
	ContainerName       string `json:"ContainerName"`
	BindsVolumes []BindVolume `json:"BindVolumes"`
}
///////struct for list volumes
type Volume struct {
	CreatedAt string `json:"CreatedAt"`
	Driver    string `json:"Driver"`
	//Labels  interface{} `json:"Labels"`
	Mountpoint string `json:"Mountpoint"`
	Name       string `json:"Name"`
	//Options interface{} `json:"Options"`
	Scope string `json:"Scope"`
	//Status interface{} `json:"Status"`
}

type Volumes struct {
	Volumes []Volume `json:"Volumes"`
}

////////struct containing the container json info passed in http to create container ,to a struct containing  container info (json --->struct ) by method
//   (p* ContainerToCreate)func GetContainerToCreateInfo (d string) (error)() fill the struct from json
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

type VolumesPrune struct {
	VolumesDeleted []string `json:"VolumesDeleted"`
	SpaceReclaimed int      `json:"SpaceReclaimed"`
}

type responceHttp struct {
	Info string
}

type loggerLine struct {
	Line    int
	Content string
}
type LoggerLineSlice []loggerLine

type Mount struct {
	Name string `json:"Name"`
	Destination string `json:"Destination"`
	Mode        string `json:"Mode"`
	RW          bool   `json:"RW"`
	Source      string `json:"Source"`
}
type Port struct {
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort"`
	Type        string `json:"Type"`
}
type NetworkInfo struct {
	Gateway     string `json:"Gateway"`
	IPAddress   string `json:"IPAddress"`
	MacAddress  string `json:"MacAddress"`
	Networkname string `json:"NetWorkName"`
}

type Container struct {
	Command         string      `json:"Command"`
	NetworkSettings interface{} `json:"NetworkSettings"`
	Created         int32       `json:"Created"`
	Id              string      `json:"Id"`
	Image           string      `json:"Image"`
	Mounts          []Mount     `json:"Mounts"`
	Names           []string    `json:"Names"`
	Ports           []Port      `json:"Ports"`
	State           string      `json:"State"`
	Status          string      `json:"Status"`
}

type NewEnv struct {
	ImagePerm string  `json:"ImagePerm"`
	NetworkPerm string  `json:"NetworkPerm"`
}

type Containers []Container

type Networks struct {
	network map[string]interface{} `json:"Networks"`
}

func getNetworkNames(p map[string]interface{}) []string {
	var networknames []string

	for k := range p {
		networknames = append(networknames, k)
	}
	return networknames
}
func (p *NewEnv) GetNeWEnv (d string) error {
	method := "GetNeWEnv():"
	err := json.Unmarshal([]byte(d), p)
	if err != nil {
		logdm.WriteLogLine(method + "errore in codifica json!")
		return errors.New("error converting in struct , the json passed for newenv.")
	}
	logdm.WriteLogLine(method + "received JSON for permitted images: " + p.ImagePerm + " and for permitted netorks " + p.NetworkPerm)
	
	return nil
}

/// construct an slice of networkInfo[] with the all the network info of a container.
func (p Container) getAllInfo() []NetworkInfo {
	if _, ok := p.NetworkSettings.(map[string]interface{}); !ok {
		//networksetting e' interface{} se non e' istanza di map[string]interface{} allora non ho appena letto da json,
		//e quindi sono chiamate successive di mettodi di containers
		networks := p.NetworkSettings.([]NetworkInfo)
		return networks
	} else {
		networks := (p.NetworkSettings.(map[string]interface{})["Networks"]).(map[string]interface{})
		var allNet []NetworkInfo
		netnames := getNetworkNames(networks)
		for _, k := range netnames {
			l := new(NetworkInfo)
			nw := networks[k].(map[string]interface{})
			ip := nw["IPAddress"].(string)
			gw := nw["Gateway"].(string)
			ma := nw["MacAddress"].(string)
			l.IPAddress = ip
			l.Gateway = gw
			l.MacAddress = ma
			l.Networkname = k
			allNet = append(allNet, *l)
		}
		return allNet
	}
}

/////////////////////////Image permitted in env
func getImageByEnv() []string {
	method := "getImageByEnv():"
	var containerList string = os.Getenv("PERMITTED_IMAGES")
	logdm.WriteLogLine(method + "getting permitted image. ")
	s := strings.Split(containerList, ",")
	return s
}

func CanUseImage(imageName string) bool {
	method := "CanUseImage():"
	if os.Getenv("PERMITTED_IMAGES")=="all" {
		logdm.WriteLogLine(method + "all images are permitted. ")
		return true
	}
	for _, name := range getImageByEnv() {
		if name == imageName {
			logdm.WriteLogLine(method + " this image is in permitted list :" + imageName)
			return true
		}
	}
	logdm.WriteLogLine(method + "image is not in permitted list :" + imageName)
	return false
}

////////////////////// networks permitt in env
func getNetByEnv() []string {
	method := "getNetByEnv():"
	var containerList string = os.Getenv("PERMITTED_NETWORKS")
	logdm.WriteLogLine(method + "getting permitted NETWORKS. ")
	s := strings.Split(containerList, ",")
	fmt.Println(s)
	return s
}
func CanUseNetWork(networkName string) bool {
	method := "CanUseNetWork():"
	fmt.Println("------------------------------->"+os.Getenv("PERMITTED_NETWORKS"))
	if os.Getenv("PERMITTED_NETWORKS") == "all" {
		logdm.WriteLogLine(method + " all networks are permitted list :" + networkName)
		return true
	}
	logdm.WriteLogLine(method + " inspecting network " + networkName)
	for _, nName := range getNetByEnv() {
		if nName == networkName {
			logdm.WriteLogLine(method + " " + networkName + " this network is permitted network")
			return true
		}
	}
	logdm.WriteLogLine(method + " " + networkName + " this network is not permitted networks ")
	return false
}

func CanUseNetworks(nws []NetworkInfo) bool {
	method := "CanUseNetworks():"
	var useNet bool = true
	for _, net := range nws {
		logdm.WriteLogLine(method + " NETWORK --------> " + net.Networkname + " network")
		if !CanUseNetWork(net.Networkname) {
			useNet = false
			break
		}
	}
	return useNet
}

//////////////////////
func ResponcHttpStatus(status string, w io.Writer) {
	var res responceHttp
	res.Info = status
	res.toJSONResponce(w)
}

func (p *responceHttp) toJSONResponce(w io.Writer) error {

	e := json.NewEncoder(w)
	return e.Encode(p)
}

//////////////////////
func (p *LoggerLineSlice) GetJSONLogsByLine(d string, w io.Writer) error {

	var i int = 0
	var regLog = regexp.MustCompilePOSIX(`\n`)
	loglines := regLog.Split(d, -1)
	for _, line := range loglines {
		var elem loggerLine
		elem.Line = i
		elem.Content = line
		*p = append(*p, elem)
		i++
	}
	e := json.NewEncoder(w)
	return e.Encode(p)

}
/////////////////////////////////
func (p *LoggerLineSlice) GetJsonEnv(d []string, w io.Writer) error {
	var i int = 0
	for _, line := range d {
		var elem loggerLine	
		token := strings.Split(line,"=")
		if token[0] != "DB_PASS" {
			elem.Line = i
			elem.Content = line
		    *p = append(*p, elem)
			i++
		}	
	}
	e := json.NewEncoder(w)
	return e.Encode(p)

}
///////////////////////////////////////////////////////selector function on containers

//// use in utils/GetSearchInContainers
//// use in utils/GetSearchInContainersByimage

func (p *Containers) searchByName(name string) (bool, Containers) {
	method := "searchByName()"
	var found bool = false
	resultSearch := Containers{}
	nameJson := "/" + name
	for _, k := range *p {
		if k.Names[0] == nameJson {
			logdm.WriteLogLine(method + "container is FOUND name: " + name)

			k.NetworkSettings = k.getAllInfo()

			resultSearch = append(resultSearch, k)
			found = true
			break
		}
	}
	return found, resultSearch
}

//// use in  utils/GetAllPermittedByImgNet
//// use in utils/GetSearchInContainersByimage
//// use in utils/GetSearchInStopContainersByName(
////// use for endpoint : RemoveExitByImage
////// use for endpoint : listbyimage
////// use for endpoint : list
func (p *Containers) selectPermittedByImage() Containers {
	method := "selectPermittedByImage():"
	resultSearch := Containers{}
	var i int = 0
	for _, k := range *p {
		imageJson := k.Image
		nameContainerJson := strings.ReplaceAll(k.Names[0], "/", "")
		k.Names[0] = nameContainerJson
		logdm.WriteLogLine(method + "Considering image ->" + imageJson)
		if CanUseImage(imageJson) {
			i++
			k.NetworkSettings = k.getAllInfo()
			resultSearch = append(resultSearch, k)
		}
	}
	logdm.WriteLogLine(method + "added " + strconv.Itoa(i) + " permitted container from env.")
	return resultSearch
}

//// use in  utils/GetAllPermittedByImgNet
//// use in utils/GetSearchInContainersByimage
//// use in utils/GetSearchInStopContainersByName(
////// use for endpoint : RemoveExitByImage
////// use for endpoint : listbyimage
////// use for endpoint : list
func (p *Containers) selectPermittedNetwork() Containers {
	method := "selectPermittedNetwork():"
	resultSearch := Containers{}
	var i int = 0
	for _, k := range *p {
		logdm.WriteLogLine(method + " inspecting container  ++++++++> " + k.Names[0] + " <++++++++ for permitted network.")
		nws := k.getAllInfo()
		nameContainerJson := strings.ReplaceAll(k.Names[0], "/", "")
		k.Names[0] = nameContainerJson
		useNets := CanUseNetworks(nws)
		if useNets {
			logdm.WriteLogLine(method + "container ++++++++>" + k.Names[0] + " <++++++++ is on permitted permitted networks")
			resultSearch = append(resultSearch, k)
			i++
		} else {
			logdm.WriteLogLine(method + "container ++++++++> " + k.Names[0] + " <++++++++ is NOT On permitted permitted networks")
		}

	}
	logdm.WriteLogLine(method + "add to select " + strconv.Itoa(i) + " container that are on permitted network.")
	return resultSearch
}

//// use in utils/GetSearchInStopContainersByName(
////// use for endpoint : RemoveExitByImage
func (p *Containers) searchByImageNameAndStop(imagename string) Containers {
	method := "searchByImageNameAndStop():"
	resultSearch := Containers{}
	for _, k := range *p {

		if k.Image == imagename && k.State == "exited" {
			nameContainerJson := strings.ReplaceAll(k.Names[0], "/", "")
			logdm.WriteLogLine(method + "container with name" + nameContainerJson + " is in exited state.")
			logdm.WriteLogLine(method + "container with image : " + imagename + " found , with names " + nameContainerJson + " and state " + k.State)
			k.Names[0] = nameContainerJson
			resultSearch = append(resultSearch, k)
			logdm.WriteLogLine(method + " container " + nameContainerJson + " added to image list.")
		}
	}
	return resultSearch
}

func (p *Containers) searchInStop() Containers {
	method := "searchInStop():"
	resultSearch := Containers{}
	for _, k := range *p {

		if k.State == "exited" {
			nameContainerJson := strings.ReplaceAll(k.Names[0], "/", "")
			logdm.WriteLogLine(method + "container with name" + nameContainerJson + " is in exited state.")
			k.Names[0] = nameContainerJson
			resultSearch = append(resultSearch, k)
			logdm.WriteLogLine(method + " container " + nameContainerJson + " added to exit list.")
		}
	}
	return resultSearch
}

//// use in utils/GetSearchInContainersByimage
////// use for endpoint : listbyimage
func (p *Containers) searchByImageName(imagename string) Containers {
	method := "searchByNameImage():"
	resultSearch := Containers{}
	for _, k := range *p {
		if k.Image == imagename {
			nameContainerJson := strings.ReplaceAll(k.Names[0], "/", "")
			logdm.WriteLogLine(method + "container with image : " + imagename + "found , with names " + nameContainerJson)
			k.Names[0] = nameContainerJson

			k.NetworkSettings = k.getAllInfo()

			resultSearch = append(resultSearch, k)
			logdm.WriteLogLine(method + " container " + nameContainerJson + " added to image list.")
		}
	}
	return resultSearch
}

///////////////////////////////////////////////////use by httpdm

//// use in httpdm SearchByName()
//// use in endpoint : searchbyname
func (p *Containers) GetSearchInContainers(d string, name string, w io.Writer) (error, bool) {
	method := "GetSearchInContainers()"
	//logdm.WriteLogLine(method + " json container list :" + d)
	err := json.Unmarshal([]byte(d), p)
	if err != nil {
		logdm.WriteLogLine(method + "errore in codifica json!")
		return errors.New("errore nella codifica json!"), false
	}
	logdm.WriteLogLine(method + "")
	found, containerSelected := p.searchByName(name)
	e := json.NewEncoder(w)
	return e.Encode(&containerSelected), found
}

//// use in httpdm/checkifCanUseContainer
//// use in endpoint : StopIt
func (p *Containers) getImageFromName(cname string) string {
	method := "getImageFromName():"
	logdm.WriteLogLine(method + " search image of container :" + cname)
	var image string = ""
	cnameJson := "/" + cname
	for _, k := range *p {
		logdm.WriteLogLine(method + " name in search : " + k.Names[0])
		if k.Names[0] == cnameJson {

			logdm.WriteLogLine(method + " FOUND container with name : " + cname + ",  with image " + k.Image)
			image = k.Image
			break
		}
	}
	return image
}

////use in httpdm/checkifCanUseContainer
//// use in endpoint : StopIt
func (p *Containers) getNetworksFromName(cname string) []NetworkInfo {
	method := "getNetworksFromName():"
	logdm.WriteLogLine(method + " search networks for container " + cname)
	var network []NetworkInfo
	cnameJson := "/" + cname
	for _, k := range *p {
		logdm.WriteLogLine(method + " name in search : " + k.Names[0])
		if k.Names[0] == cnameJson {
			logdm.WriteLogLine(method + " Found container with name : " + cname + ", getting networks ")
			network = k.getAllInfo()
			break
		}
	}
	return network
}

//// use in httpdm/GetAllPermitted
///// use in endpoint : list
func (p *Containers) GetAllPermittedByImgNet(d string, w io.Writer) error {

	method := "GetAllPermittedContainersByImageAndNet():"
	err := json.Unmarshal([]byte(d), p)
	if err != nil {
		logdm.WriteLogLine(method + err.Error())
		logdm.WriteLogLine(method + "JSON : " + d)
		logdm.WriteLogLine(method + "errore in codifica json!")
		return errors.New("errore nella codifica json!")
	}
	//containerSelectedByImg := p.selectPermittedByImage()
	//containerSelected := containerSelectedByImg.selectPermittedNetwork()
	containerSelectedByNet := p.selectPermittedNetwork()
	containerSelected := containerSelectedByNet.selectPermittedByImage()
	e := json.NewEncoder(w)
	return e.Encode(&containerSelected)
}

//// use in httpdm/checkIfCanUseContainer
//// use in httpdm/StopIt
///// use in endpoint : stopit
func (p *Containers) GetImageAndNetwsFromCname(d string, cname string) (string, []NetworkInfo) {
	method := "GetImageAndNetwsFromCname():"
	var networksSelected []NetworkInfo
	err := json.Unmarshal([]byte(d), p)
	if err != nil {
		logdm.WriteLogLine(method + "errore in codifica json!")
		return "", networksSelected
	}
	imageSelected := p.getImageFromName(cname)
	networksSelected = p.getNetworksFromName(cname)

	return imageSelected, networksSelected
}
func (p *ContainerToCreate) GetContainerToCreateInfo(d string) error {
	method := "GetContainerToCreateInfo():"
	err := json.Unmarshal([]byte(d), p)
	if err != nil {
		logdm.WriteLogLine(method + "errore in codifica json!")
		return errors.New("error converting in struct , the json passed for create container.")
	}
	logdm.WriteLogLine(method + "received JSON container " + p.Name + " ON NETWORK " + p.Network)
	for _,k:= range p.Volumes {
		logdm.WriteLogLine(method + " volume "+k.Volume+" mount point "+k.MountPo) 
	}
	return nil
}

//// use in httpdm/ListByImage
///// unse in endpoint : listbyimage
func (p *Containers) GetSearchInContainersByimage(d string, imgname string, w io.Writer) (error, Containers) {
	method := "GetSearchInContainersByimage():"
	var containerSelectedByName Containers
	err := json.Unmarshal([]byte(d), p)
	if err != nil {
		logdm.WriteLogLine(method + "errore in codifica json!")
		return errors.New("errore nella codifica json!"), containerSelectedByName
	}
	containerSelectedByImg := p.selectPermittedByImage()
	containerSelectedByNws := containerSelectedByImg.selectPermittedNetwork()
	containerSelectedByName = containerSelectedByNws.searchByImageName(imgname)
	e := json.NewEncoder(w)
	return e.Encode(&containerSelectedByName), containerSelectedByName
}

//// use in httpdm/RemoveExitByImage
//// use in httpdm/exited
///// use in endpoint : removexitbyimage
///// use in endpoint : exited
func (p *Containers) GetSearchInStopContainersByNameOrNot(d string, name string, w io.Writer) (error, Containers) {
	method := "GetSearchInStopContainersByNameOrNot():"
	err := json.Unmarshal([]byte(d), p)
	logdm.WriteLogLine(method + " searching for container with name: " + name + ", in exit state")
	var containerSelected Containers
	if err != nil {
		logdm.WriteLogLine(method + "errore in codifica json!")
		return nil, containerSelected
	}
	if name == "" || name =="all" {
		if name == "" {
			logdm.WriteLogLine(method + " name= ")
			containerSelectByImage := p.selectPermittedByImage()
			containerSelectByNet := containerSelectByImage.selectPermittedNetwork()
			containerSelected = containerSelectByNet.searchInStop()
		} else {
			logdm.WriteLogLine(method + " name="+name+" prune")
			containerSelectByNet := p.selectPermittedNetwork()
			containerSelected = containerSelectByNet.searchInStop()
		}
		
	} else {
		logdm.WriteLogLine(method + " name="+name)
		containerSelectByImage := p.selectPermittedByImage()
		containerSelectByNet := containerSelectByImage.selectPermittedNetwork()
		containerSelected = containerSelectByNet.searchByImageNameAndStop(name)
	}
	e := json.NewEncoder(w)
	return e.Encode(&containerSelected), containerSelected
}

//// use in : httpdm/VolumePrunes
//// use in endpoint: volumeprunes
func (p *VolumesPrune) VolumesPrune(d string, w io.Writer) error {
	method := "VolumesPrune:"
	err := json.Unmarshal([]byte(d), p)
	if err != nil {
		logdm.WriteLogLine(method + err.Error())
		logdm.WriteLogLine(method + "JSON : " + d)
		logdm.WriteLogLine(method + "errore in codifica json!")
		return errors.New("errore nella codifica json!")
	}
	e := json.NewEncoder(w)
	return e.Encode(p)
}

func (p *Volumes) IsInVolumeByName(name string) bool {
	method := "IsInVolumeByName():"
	result := false
	z := p.Volumes
	for _, k := range z {
		logdm.WriteLogLine(method + "inspecting volume :" + k.Name)
		if k.Name == name {

			logdm.WriteLogLine(method + "volume" + name + " found.")
			result = true
			break
		}
	}
	return result
}

/*
func (p *Containers) GetContainerByName(name string,d string, w io.Writer) error {

	method := "GetAllPermittedContainersByImageAndNet():"
	err := json.Unmarshal([]byte(d), p)
	if err != nil {
		logdm.WriteLogLine(method + err.Error())
		logdm.WriteLogLine(method + "JSON : " + d)
		logdm.WriteLogLine(method + "errore in codifica json!")
		return errors.New("errore nella codifica json!")
	}
	//containerSelectedByImg := p.selectPermittedByImage()
	//containerSelected := containerSelectedByImg.selectPermittedNetwork()
	containerSelectedByNet := p.selectPermittedNetwork()
	containerSelectedByImage := containerSelectedByNet.selectPermittedByImage()
	containerSelectedByName, _ := containerSelectedByImage.searchByName(name)
	
	e := json.NewEncoder(w)
	return e.Encode(&containerSelectedN)
}
*/

func (p *Volumes) SearchVolumeByName(d string, name string) bool {
	method := "searchVolumeByName():"
	err := json.Unmarshal([]byte(d), p)
	if err != nil {
		fmt.Println(err.Error)
		logdm.WriteLogLine(method + "errore in codifica json!")
		return false
	}
	result := p.IsInVolumeByName(name)
	return result
}

func (p *Volumes) VolumesList(d string, w io.Writer) error {
	method := "VolumeList():"
	err := json.Unmarshal([]byte(d), p)
	if err != nil {
		logdm.WriteLogLine(method + err.Error())
		logdm.WriteLogLine(method + "JSON : " + d)
		logdm.WriteLogLine(method + "error in json code!")
		return errors.New("errore in the json code!")
	}
	e := json.NewEncoder(w)
	return e.Encode(p)
}

////  use in : httpdm/VolumeCreate
////
func (p *VolumeToCreate) GetVolumeToCreateInfo(d string) error {
	method := "GetVoluleToCreateInfo():"
	err := json.Unmarshal([]byte(d), p)
	if err != nil {
		logdm.WriteLogLine(method + "errore in codifica json!")
		return errors.New("errore nella codifica json!")
	}
	name := p.Name
	logdm.WriteLogLine(method + "retried info on volume " + name)
	return nil
}

//// use in : httpdm/CreateContainer
////create from a struct rappresenting the container the json

func (p *ToJsonContainer) MAKEJsonContainer(imageName string, netw string, volume []VolumeCreate,env []string) (bool,string) {
	method := "MAKEJsonContainer()"
	p.Image = imageName
	var binds []string
	var envVar []string
	for _,vol := range volume {
		bind := vol.Volume+":"+vol.MountPo	
		ok, message := CheckContainerCreateCond(vol.Volume,vol.MountPo,vol.VolumeType)
		if !ok {
			logdm.WriteLogLine(method + "error check mount point and volume :"+message+ " vol :"+ vol.Volume + "mp : "+ vol.MountPo)
			return false,"error checking mount point and volume"
		}
		binds = append(binds,bind)
	}
	if len(volume)==0 {
		logdm.WriteLogLine(method + "is to create a container without volume")
		p.HostConfig = map[string]interface{}{}
	} else {
		p.HostConfig = map[string]interface{}{"Binds": binds}
		logdm.WriteLogLine(method + "is to create a container with volume ")
	}
	if len(env) ==0 {
		logdm.WriteLogLine(method + "the container dont have env")
		p.Env = envVar
	} else {
		logdm.WriteLogLine(method + "the container have env Var")
		p.Env = env
	}
	p.NetworkingConfig.EndpointsConfig = map[string]interface{}{
		netw: map[string]interface{}{},
	}
	jsonData, err := json.Marshal(p)
	if err != nil {
		logdm.WriteLogLine(method + "Error converting the new container struct to JSON")
		return false,"Error converting the new container struct to JSON"
	}
	return true,string(jsonData)
}


func (p *Containers) searchbByBind() Containers {
	method := "searchbByBind():"
	resultSearch := Containers{}
	for _, k := range *p {
		nameContainerJson := strings.ReplaceAll(k.Names[0], "/", "")
		if len(k.Mounts) != 0 {
			for _,vol := range k.Mounts {
				if vol.Name == "" {
					k.NetworkSettings = k.getAllInfo()
					resultSearch = append(resultSearch,k)
					logdm.WriteLogLine(method + "found and added container with a Bind Volume : "  + nameContainerJson + "src: "+ vol.Source +" dest:"+vol.Destination)
					break
				}
			}
		} else {
			logdm.WriteLogLine(method + "container without volumes: "  + nameContainerJson)
		}
		 
	}
	return resultSearch
}
func (p *Containers) GetSearchInContainersWithBinds(d string, w io.Writer) (error, Containers) {
	method := "GetSearchInContainersWithBinds():"
	var containerSelectedByName Containers
	err := json.Unmarshal([]byte(d), p)
	if err != nil {
		logdm.WriteLogLine(method + "errore in codifica json!")
		return errors.New("errore nella codifica json!"), containerSelectedByName
	}
	containerSelectedByImg := p.selectPermittedByImage()
	containerSelectedByNws := containerSelectedByImg.selectPermittedNetwork()
	containerSelectedByName = containerSelectedByNws.searchbByBind()
	e := json.NewEncoder(w)
	return e.Encode(&containerSelectedByName), containerSelectedByName
}

func (p *Containers) GetBindsList(d string, w io.Writer) error {
	method := "GetSearchInContainersWithBinds():"
	var containerSelectedByBind Containers
	err := json.Unmarshal([]byte(d), p)
	if err != nil {
		logdm.WriteLogLine(method + "errore in codifica json!")
		return errors.New("errore nella codifica json!")
	}
	containerSelectedByImg := p.selectPermittedByImage()
	containerSelectedByNws := containerSelectedByImg.selectPermittedNetwork()
	containerSelectedByBind = containerSelectedByNws.searchbByBind()
	var results  []ContainerWithBinds
	for _,cont := range containerSelectedByBind {
		var c ContainerWithBinds
		c.ContainerName = cont.Names[0]
		var d []BindVolume
		insert := false
		for _,vol := range cont.Mounts {
			if vol.Name == "" {
				insert=true
				var e BindVolume
				e.Destination = vol.Destination
				e.Source = vol.Source
				e.RW = vol.RW
				d = append(d,e)
			}
		}
		if insert {
			c.BindsVolumes = d
			results = append(results,c)
		}
	}
	e := json.NewEncoder(w)
	return e.Encode(&results)
}

func structToMap(i interface{},rc int ) map[string]interface{} {
	method := "structToMap(): "
	rc++
	logdm.WriteLogLine(method + "---------<Ricorsion room "+ strconv.Itoa(rc) +">-------------")
	defer logdm.WriteLogLine(method + "------>END Ricorsion room "+ strconv.Itoa(rc) +"<-------------")
	k := make(map[string]interface{})
	iVal := reflect.ValueOf(i)//torna il Value di i
	t := iVal.Type() //torna il type perche da questo si possono desumere il nome dei campi
	for i := 0; i < iVal.NumField(); i++ {//nymfield info che e' anche in value
		ft := t.Field(i)//da ft prendo il campo i esimo ---> questo mi da il nome del campo 
		fival := iVal.Field(i)//e il Value del campo --> questo mi da il  il tipo e l'oggetto prossimo della ricorsione
		switch fival.Kind() {
		case  reflect.Struct: 
			logdm.WriteLogLine(method + "STRUCT activation New ricorsion Room")
			k[ft.Name] = structToMap(fival.Interface(),rc)
		default:
			logdm.WriteLogLine(method + "Base step type: "+fival.Kind().String())
			k[ft.Name] = fival.Interface()
		}
	}
	return k 	
}
func (p *Containers) getMapContainerSelectedByBind() []map[string]interface{} {
	method := "getMapContainerSelectedByBind():"
	var resultSearch []map[string]interface{}
	logdm.WriteLogLine(method + "get container permitted:")
	for _, k := range *p {
		logdm.WriteLogLine(method + " container ->: "+ k.Names[0])
		ricorsionCount :=0
		elem := make(map[string]interface{})
		elemout := make(map[string]interface{})
		elem = structToMap(k,ricorsionCount)
		elemout["Names"] = elem["Names"]
		elemout["Mount"] = elem["Mounts"]
		resultSearch = append(resultSearch,elemout ) 
	}
	return resultSearch
}
//json --> struct/map unmarhall
//struct/map --> json marshall
func (p *Containers) GetBindsListRfl(d string, w io.Writer) error {
	method := "GetBindsListRfl():"
	var containerSelectedByBind Containers
	var mapContainerSelectedByBind []map[string]interface{}
	err := json.Unmarshal([]byte(d), p)
	if err != nil {
		logdm.WriteLogLine(method + "errore in codifica json!")
		return errors.New("errore nella codifica json!")
	}
	containerSelectedByImg := p.selectPermittedByImage()
	containerSelectedByNws := containerSelectedByImg.selectPermittedNetwork()
	containerSelectedByBind = containerSelectedByNws.searchbByBind()
	mapContainerSelectedByBind = containerSelectedByBind.getMapContainerSelectedByBind()
	e := json.NewEncoder(w)
	return e.Encode(&mapContainerSelectedByBind)
}