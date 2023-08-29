package main

import "fmt"
import "encoding/json"

type Binds struct {
	Binds [] string `json:”Binds”`
}
type NetworkingConfig struct {
	EndpointsConfig map[string]interface{} `json:”EndpointsConfig”` 
}
type ToJsonContainer struct {
	Image string `json:”Image”`
	HostConfig Binds `json:”HostConfig”`
	NetworkingConfig NetworkingConfig `json:”NetworkingConfig”` 
}
/*
struct sono dizionari, slice sono liste
Esiste un diziononario che ha come chiavi :
- Image guardare ---> `json:”Image”` per il nome della chiave
- ha una chiave hostconfig , che ha come  valore  un dizionario con chiave binds ,e valore una lista di stringhe
- ha una chiave NetworkingConfig che ha come valore un dizionario con chiave endpointsConfig e come valore un dizionario

disegnamola:
{
	"Image": "myimage",
	"HostConfig":{
		"bind":["/mnt:/mnt"]
	},
	"NetworkingConfig":{
		"EndpointsConfig":{
			"mapkey1":"valueMapKey1",
			"mapkey2":"valueMapKey2",
		}
	}
}

*/
func main()  {
	netw := "bridge"
	var toJSON ToJsonContainer
	toJSON.Image = "myImage"
	var bind Binds
	var bindSlice[]string
	bindSlice = append(bindSlice,"/mnt:/mnt")
	bind.Binds = bindSlice
	toJSON.HostConfig = bind
	NetworkingConfig := NetworkingConfig {
		EndpointsConfig: map[string]interface{}{
			netw:map[string]interface{}{},
		},
	}
	toJSON.NetworkingConfig = NetworkingConfig
	jsonData, err := json.Marshal(toJSON)
    if err != nil {
        panic(err)
    }

    // stampiamo il JSON
    fmt.Println(string(jsonData))

   
}