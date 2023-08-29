package logdm

import "os"
import "log"
import "fmt"

var infofile *os.File = nil
var infologger    *log.Logger = nil

func WriteLogLine (message string) bool {
    err := getLog()
    if err != nil {
        fmt.Println ("error getting  the log.")
        return false
	}
	infologger.Println(message)
    return true
}

func getLog () error {
    if infologger == nil {
		fmt.Println ("logger is building..")
		infofile, err := os.OpenFile("/log/dockermanager.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			fmt.Println ("cannot openfile!")
			return err
		}
		fmt.Println ("file is now open")
		infologger = log.New(infofile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
		if err != nil {
			fmt.Println ("cannot get a logger .")
			return err
		}
		fmt.Println ("loggers is ok")
		return nil
	} else {
		//fmt.Println ("logger is already istansiate.")
		return nil
	}	
}
