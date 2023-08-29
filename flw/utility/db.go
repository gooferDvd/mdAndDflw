package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq" // Import the PostgreSQL driver
	//"github.com/lib/pq"
)

func main() {
	// Connessione al database PostgreSQL
	host :="192.168.56.20"
	port := 25432
	user :="dev"
	password :="x"
	dbname :="test"
	schema := "pipelines"
	var dbConn *sql.DB = nil
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+"password=%s dbname=%s sslmode=disable search_path=%s", host, port,user, password, dbname, schema)
	dbConn, err:= sql.Open("postgres", psqlInfo)
	if err != nil {
		fmt.Println("error opening the db :" + err.Error() )
		return 
	}
	fmt.Println( "open connection" + psqlInfo)
	err = dbConn.Ping()
	if err != nil {
		fmt.Println( "db in not responding :"+ err.Error())
		fmt.Println ("errore"+err.Error())
		
	} else {
		fmt.Println( " connection  to DB is already open")
	}
   // var tags  []int
    query := `SELECT array_to_string(precs, ',') FROM  pipelines.container where container_id=$1`
	rows, err := dbConn.Query(query,6)
	if err != nil {
		fmt.Println("errore")
	}
	defer rows.Close()

	var precsString string
	for rows.Next() {
		
		err = rows.Scan(&precsString)
		if err != nil {
			fmt.Println("secondo errore")
		}
	}
	fmt.Println(precsString)
}	