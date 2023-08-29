package utils

import (
	logdm "example.com/logdm"
	"strconv"
	"strings"
	"fmt"
	"database/sql"
	"errors"
	//"github.com/lib/pq"
	
)

func openDB() error {
	setEnv()
	//PrintEnv()
	method := "openDB(): "
	var err error
	if dbConn == nil {
			psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable search_path=%s", host, port,user, password, dbname, schema)
			dbConn, err = sql.Open("postgres", psqlInfo)
			if err != nil {
				logdm.WriteLogLine(method + "error opening the db :" + err.Error() )
				return err
			}
			logdm.WriteLogLine(method + "open connection" + psqlInfo)
			err = dbConn.Ping()
			if err != nil {
				logdm.WriteLogLine(method + "db in not responding :"+ err.Error())
				logdm.WriteLogLine ("errore"+err.Error())
				return err
			} else {
				logdm.WriteLogLine(method + " connection  to DB is already open")
			}
	} 

	return nil
}
/////control if the pipeline is in cache
func isKeyPresent ( key string) bool {
	method := "isKeyPresent(): "
	present := false
	for k := range keypipeline {
		if k == key {
			logdm.WriteLogLine (method + " key " + key + " already present.")
			present = true
			break
		}
	}
	if !present {
		logdm.WriteLogLine (method + " key " + key + " is NOT already present.")	
	}
	return present
}

func getIfNode (id int ) (error,map[int]ifStrutture) {
	method := "getIfNode() :"
	err := openDB()
	if err != nil {
		return err,nil
	}
	var ifNodes map[int]ifStrutture
	ifNodes = make(map[int]ifStrutture)
	sqlstatement := `select container_id,next_ok,next_ko from pipelines.container where fk_pipeline_id =$1 and type='if'`
	rows,err := dbConn.Query(sqlstatement,id)
	defer rows.Close()
	if err != nil {
        logdm.WriteLogLine(method+ "error in sql query statement "+ sqlstatement +":" + err.Error())
        return err,nil
    }
	key := 0
	nextOk := 0
	nextKo := 0
	returnNoRecord := true 
	for rows.Next() {
		returnNoRecord = false
		ifs:=new(ifStrutture)
        err := rows.Scan(&key,&nextOk,&nextKo)
        if err != nil {
            logdm.WriteLogLine (method + "errore during result query error "+err.Error())
            return err,nil
		}
		(*ifs).okContainerid = nextOk
		(*ifs).koContainerid = nextKo
		(*ifs).imageOk = getImageById(nextOk)
		(*ifs).imageKo = getImageById(nextKo)
    	ifNodes[key]=*ifs
        logdm.WriteLogLine (method+ "adding key "+ strconv.Itoa(key) + "to the map of intial node Value NL AND IMAGEOK = "+(*ifs).imageOk)
	}
	if returnNoRecord {
		logdm.WriteLogLine (method + " no if RECORD for the pipeline ")
		return nil,nil
	}
	/*
	for _,v := range ifNodes {
		imok := getImageById(v.okContainerid)
		imko := getImageById(v.koContainerid)
		v.imageOk = imok
		v.imageKo = imko
	}
	*/
	return nil,ifNodes
}

func getImageById(id int) string {
	err := openDB()
	if err != nil {
		return ""
	}
	sqlstatement := `select image_name from  pipelines.container where container_id=$1`
	row := dbConn.QueryRow(sqlstatement,id)
	image := ""
	_ = row.Scan(&image)
	return image
}

func inizializeExeMap (id int ) (error,map[int]string) {
	method := "inizializeExeMap(): "
	err := openDB()
	if err != nil {
		return err,nil
	}
	
	var nodes map[int]string
	nodes = make(map[int]string)
	sqlstatement := `select container_id from pipelines.container where fk_pipeline_id =$1`
	
	rows,err := dbConn.Query(sqlstatement,id)
	defer rows.Close()
	if err != nil {
        logdm.WriteLogLine(method+ "error in sql query statement "+ sqlstatement +":" + err.Error())
        return err,nil
    }
	key := 0
	for rows.Next() {
        err := rows.Scan(&key)
        if err != nil {
            logdm.WriteLogLine (method + "errore during result query error "+err.Error())
            return err,nil
        }
    	nodes[key]="NL"
        logdm.WriteLogLine (method+ "adding key "+ strconv.Itoa(key) + "to the map of intial node Value NL")
    }
	return nil,nodes
}
//////fill the cache
func fillThePipelineMap () error {
	method := "fillThePipelineMap(): "
	err := openDB()
	if err != nil {
		return err
	}
	if dbConn == nil {
		logdm.WriteLogLine(method + "dbConn is nil!!!!!!")	
	}
	sqlstatement := `select count(*) from pipelines.pipeline`
	row := dbConn.QueryRow(sqlstatement)
	count := 0
	_ = row.Scan(&count)
	if count == 0 {
		logdm.WriteLogLine(method+" first pipeline !")
		return nil
	}
	sqlstatement = `select name from pipelines.pipeline`
	
    rows, err := dbConn.Query(sqlstatement)
	defer rows.Close()
    if err != nil {
		logdm.WriteLogLine ("errore"+err.Error())
		return err
	}
	
	var key string
	for rows.Next() {
		err := rows.Scan(&key)
        if err != nil {
			logdm.WriteLogLine ("errore"+err.Error())
			return err
		}
		keypipeline[key]="present."
		logdm.WriteLogLine (method+ "adding key "+ key + "to the map.")

	}
	return nil

}
////get the sequense
func getNextSeqId (seq string,n int) ([]int,error) {
	method := "getNextSeqId(): "
	var seqSlice []int
	err := openDB()
	if err != nil {
		logdm.WriteLogLine (method + "error opening db ."+err.Error())
		return seqSlice,err
	}
    var sqlstatement string
	if seq == "pipeline" {
		sqlstatement = `select nextval('pipelines.pipeline_id_seq')`
	} else if seq=="container" {
		sqlstatement = `select nextval('pipelines.container_id_seq')`
	} else if seq== "run" {
		sqlstatement = `select nextval('pipelines.run_id_seq')`
	} else if seq=="run_detail"{
		sqlstatement = `select nextval('pipelines.run_detail_id_seq')`
	} else {
		logdm.WriteLogLine (method + "wrong sequence name!!!!!!")
		return seqSlice,errors.New("the seq not exists")	
	}
	
	for k:=0 ; k<n ; k++ {
		rows, err := dbConn.Query(sqlstatement)
		defer rows.Close()
		if err != nil {
			logdm.WriteLogLine (method + "error retrying value for sequence ."+err.Error())
			return seqSlice,err
		}
		next := 0
		for rows.Next() {
			err := rows.Scan(&next)
			if err != nil {
				logdm.WriteLogLine (method + "error reading value  for sequence ."+err.Error())
				return seqSlice,err
			}	
		}
		seqSlice =append(seqSlice,next)
    }
	return seqSlice,nil
}


func getPipelineIdByName(name string) (error,int) {
	method := "getPipelineIdByName(): "
	err := openDB()
	if err != nil {
		logdm.WriteLogLine (method + "error opening db ."+err.Error())
		return err,0
	}
	sqlstatement := `select pipeline_id from pipelines.pipeline where name = $1`
	logdm.WriteLogLine (method + " execute query "+ sqlstatement)
	pipeline_id:=0
	row := dbConn.QueryRow(sqlstatement,name)
	err = row.Scan(&pipeline_id )
	if err != nil {
		logdm.WriteLogLine (method + "error while getting pipeline id. "+err.Error())
		return err,0	
	}
	return nil,pipeline_id
}

func getEndPipeline (name string) (error,[]int) {
	method := "getPipeline(): "
	var lasts[]int
	err := openDB()
	if err != nil {
		logdm.WriteLogLine (method + "error opening db ."+err.Error())
		return err,lasts
	}
   
	err,pipeline_id:= getPipelineIdByName(name)
	if err != nil {
		logdm.WriteLogLine (method + "error getting the id for pipeline "+err.Error())
		return err,lasts
	}
	
	sqlstatement := `select container_id from pipelines.container t1 where not exists ( select 1 from pipelines.container t2 where t1.container_id = any (t2.precs)) and fk_pipeline_id=$1`
	rows,err := dbConn.Query(sqlstatement,pipeline_id)
	defer rows.Close()
	
	for rows.Next() {
		var last int
		err := rows.Scan(&last)
		if err != nil {
			logdm.WriteLogLine(method + "error scanning last ." + err.Error())
            return err, nil
		} 
		lasts =append (lasts,last)
	}
	logdm.WriteLogLine(method + " for pipeline name "+name+" last id inserted ")
	return nil,lasts
}
//return the id on table container ,and the image in a map of the root container
func getRootPipeline ( name string) (error,map[int]string) {
	method := "getRootPipeline(): "
	var root map[int]string
	root = make(map[int]string)
	err := openDB()
	if err != nil {
		logdm.WriteLogLine (method + "error opening db ."+err.Error())
		return err,nil
	}
	//l'inizio della pipeline ha prec a null
	err,id_pipeline := getPipelineIdByName(name) 
	if err != nil {
		return err,nil
	}
	logdm.WriteLogLine(method + " for pipeline name "+name+" his id is "+strconv.Itoa(id_pipeline))
	sqlstatement := `select container_id,image_name from pipelines.container where fk_pipeline_id=$1 and  precs is null  `
	
	rows,err := dbConn.Query(sqlstatement,id_pipeline)
	defer rows.Close()
	if err != nil {
		logdm.WriteLogLine(method+ "error in sql query statement "+ sqlstatement +":" + err.Error())
		return err,nil
	}
	
	key := 0
	image :=""
	for rows.Next() {
        err := rows.Scan(&key,&image)
        if err != nil {
            logdm.WriteLogLine (method + "errore during result query error "+err.Error())
            return err,nil
        }
        root[key]=image
		logdm.WriteLogLine (method+ "adding key "+ strconv.Itoa(key) + "to the map of intial node.")
	}
	/*
	if len(root) != 1 {
		logdm.WriteLogLine(method + " error pipeline start with two node o not present")
		return errors.New("error pipeline start with two node o not present"),nil
	}
	*/
	//logdm.WriteLogLine(method + " for pipeline name "+name+" start in number "+ strconv.Itoa(key) )
	return nil,root
}

func getNextPipeline (name string , father int ) (error,map[int]string) {
	method:="getNextPipeline(): "
	
	var next map[int]string
	next = make(map[int]string)
	err := openDB()
	if err != nil {
		logdm.WriteLogLine (method + "error opening db ."+err.Error())
		return err,nil
	}
	err,id_pipeline := getPipelineIdByName(name)
	if err != nil {
		return err,nil
	}
	sqlstatement :=  `select container_id,image_name from pipelines.container where $1 = any(PRECS) and fk_pipeline_id=$2`
	
	logdm.WriteLogLine(method+"getting child of container "+strconv.Itoa(father)+" for pipeline "+name)
	rows,err := dbConn.Query(sqlstatement,father,id_pipeline)
	defer rows.Close()
	if err != nil {
		logdm.WriteLogLine(method + "error durint select on db "+sqlstatement +"  "+ err.Error())
		return err,nil
	}
	for rows.Next() {
		key := 0
		image :=""
		err := rows.Scan(&key,&image)
		if err != nil {
			logdm.WriteLogLine ("error while read  next containers : "+err.Error())
			return err,nil
		}
		logdm.WriteLogLine(method+" next element for record father with id " +strconv.Itoa(father) +" is "+strconv.Itoa(key))
		next[key]=image
	}
	
	if len(next) == 0 {
		logdm.WriteLogLine(method+" end of graph")
		return nil,nil
	} else {
		return nil,next
	}

}

func insertIntoDB(tableName string, columns []string, values ...interface{}) error {
	method := "insertIntoDB(): "
	err := openDB()
	if err != nil {
		logdm.WriteLogLine (method + "error opening db ."+err.Error())
		return err
	}
    if len(columns) != len(values) {
		logdm.WriteLogLine (method + "the number of columns does not match the number of values."+err.Error())
        return fmt.Errorf("the number of columns does not match the number of values")
    }

    // Create the SQL query
    var placeholders []string
    for i := range values {
		placeholders = append(placeholders, "$"+strconv.Itoa(i+1))
    }
    query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", 
                        tableName, 
                        strings.Join(columns, ","), 
                        strings.Join(placeholders, ","))

	// Execute the query
	//logdm.WriteLogLine (method + " query : " +query)
	_, err = dbConn.Exec(query, values...)
	logdm.WriteLogLine (method + "inserted in table "+tableName+" a record with value ")
	if err != nil {
		logdm.WriteLogLine (method + " error while insert in table "+tableName+" >"+err.Error())
		return err
	}
    return nil
}



func closeIt() {
	method := "closeIt(): "
	logdm.WriteLogLine (method + "closing the db...")
	dbConn.Close()
	dbConn = nil
}

func getFatherForAchild (container_id int) (error,map[int]string) {
	method:="getFatherForAchild(): "
	precMap := make(map[int]string)

	err := openDB()
	if err != nil {
		logdm.WriteLogLine (method + "error opening db ."+err.Error())
		return err,nil
	}
	logdm.WriteLogLine(method+" id container is "+strconv.Itoa(container_id))
	//sqlstatement :=  `SELECT array_to_string(precs, ',') FROM  pipelines.container where container_id=$1`
	sqlstatement :=  `SELECT COALESCE(array_to_string(precs, ','), 'xxx') FROM pipelines.container WHERE container_id=$1`

    rows, err := dbConn.Query(sqlstatement,container_id)
	defer rows.Close()
	if err != nil {
		logdm.WriteLogLine(method+" error while doing "+sqlstatement +" "+err.Error())
		return err,nil
	}
	var precsString string
	for rows.Next() {	
		err = rows.Scan(&precsString)
		if err != nil {
			logdm.WriteLogLine(method+" error while reading result for query "+sqlstatement+" "+err.Error())
			return err,nil
		}
	}
	if precsString=="xxx" {
		logdm.WriteLogLine(method+" container "+strconv.Itoa(container_id)+" is a root node")
		return nil,precMap
	}
	precArray := strings.Split(precsString,",")
	
	logdm.WriteLogLine( method+" this are the fathers of the container "+strconv.Itoa(container_id)+" :"+precsString )

	for _,elem := range precArray {
		elem,err:=strconv.Atoi(elem)
		if err != nil {
			logdm.WriteLogLine(method+" error while converting numeric string element in integer "+err.Error())
			return err,nil
		}
		logdm.WriteLogLine(method+" this is a  father  "+strconv.Itoa(elem)+" for container "+strconv.Itoa(container_id))
		precMap[elem]="x"
	}
	
	return nil,precMap

}

func updateRowintoDB(tablename string, columns[]string,condition string,values ...interface{}) error {
	method := "updateRowintoDB(): "
	err := openDB()
	if err != nil {
		logdm.WriteLogLine (method + "error opening db ."+err.Error())
		return err
	}
	var placeholders []string
    for i := range values {
		placeholders = append(placeholders, columns[i] +"="+"$"+strconv.Itoa(i+1))
	}
	query := fmt.Sprintf("UPDATE %s SET  %s WHERE %s ", 
                        tablename,  
						strings.Join(placeholders, ","),
						condition)
	logdm.WriteLogLine (method + " ------>query  "+ query)
	_, err = dbConn.Exec(query, values...)
	logdm.WriteLogLine (method + "update in table "+tablename+" a record .")
    return err
}

func isNextFatherExecuted( containerId int, runId int) (error,bool){
	method := "isNextFatherExecuted(): "
	err := openDB()
	if err != nil {
		logdm.WriteLogLine (method + "error opening db ."+err.Error())
		return err,false
	}
	var exitStatus int = -1
	sqlstatement :=  `SELECT exit_status_container FROM  pipelines.run_detail where fk_container_id=$1 and fk_run_id=$2`
	logdm.WriteLogLine(method+" SELECT exit_status_container FROM  pipelines.run_detail where fk_container_id="+strconv.Itoa(containerId)+" and fk_run_id="+strconv.Itoa(runId))
	   	
	rows, err := dbConn.Query(sqlstatement,containerId,runId)
	defer rows.Close()
	if err != nil {
		logdm.WriteLogLine(method+" error while doing "+sqlstatement +" "+err.Error())
		return err,false
	}
	/*
	if !rows.Next() {
		logdm.WriteLogLine (method + " the father of the nodeeeeee "+strconv.Itoa(containerId)+" is not already executed")
		return nil,false
	}
	*/
	foundRecord := false

	for rows.Next() {
		foundRecord = true
		err := rows.Scan(&exitStatus)
		if err != nil {
			logdm.WriteLogLine(method + " error while reading result for query " + sqlstatement + " " + err.Error())
			return err, false
		}
	}

	if !foundRecord {
		logdm.WriteLogLine(method + " the father node " + strconv.Itoa(containerId) + " is not already executed")
		return nil, false
	}

	logdm.WriteLogLine(method+" --------------------------> "+strconv.Itoa(exitStatus))
	if exitStatus != 0 {
		logdm.WriteLogLine (method + " the father node "+strconv.Itoa(containerId) +" is not already executed")
		return nil,false
	} else {
		logdm.WriteLogLine (method + " the father node "+strconv.Itoa(containerId) +" is executed")
		return nil,true
	}
}
