# Docker Manager

## Links:
- [Go RESTful web service](https://golang.org/doc/tutorial/web-service-gin)
- [Swag](https://github.com/swaggo/swag)
- [Eureka client](https://github.com/xuanbo/eureka-client)

## Commands

Generate Json for DOC with Swagger:
> swag init


<br>

---


Il Docker manager espone degli endpoint a chiamate rest, che permettono di effettuare delle operazioni sui container docker.
Una operazione su un container e' permessa se il container ha una immagine permessa e si trova su una rete permessa.
Rete e container permesse sono dichiarate in una variabile di ambiente.
Per alcune di queste api ha senso, considerare container che sono in uno stato di exit o container che sono in uno stato di running; per questo
e' stato introdotto un parametro state, per indicare se l'endpoint aggira solo su container running, o su container che sono in running o no (state=all)
Le api , contrassegnate con un * supporteranno questa modalita.

Le apid del docker-manager sono le seguenti :
   - /list list all permitted containers method GET *
   - /restart restart a permitted container method POST *
   - /log/    give log of a permitted container method GET *
   - /searchbyname search a permitted container by name ,method GET * 
   - /stopit stop a permitted container,method POST 
   - /listbyimage search all permitted container with a permitted  image and return a list of container with that image,method GET
   - /removexitbyimage remove a permitted container in exit state if his image is permitted ,method POST 
   - /Stopallcontainerbyimage stop all permitted container with an image 
   - /exited list all permitted container in exit state
   - /create create a permitted container , with no volume, or with volume of a type. volume type.
   

   operation on volume:
   - /volumeprune delete all volume that aren't used.
   - /list list all volume
   - /volumecreate create a volume.

   il tipo di volumi utilizzati per queste api sono volumi Volume (ossia quelli creati con, docker volume create. ) non aggiscono su volumi di tipo Bind,  ossia quelli specificati ad esempio in un compose che mappano una directory del filesystem con un mountpoint del container..

Di seguito un esempio di api utilizzate per gli endpoint

- http  http://keycloak.test/api/dm/list?state=all Authorization:"Bearer $TOKEN"
- http  "http://keycloak.test/api/dm/log?name=pippo&rows=8&state=all" Authorization:"Bearer $TOKEN" 
- http  "http://keycloak.test/api/dm/searchbyname?name=pluto&state=all" Authorization:"Bearer $TOKEN" 
- http --form POST "http://keycloak.test/api/dm/stopit?name=pippo"  Authorization:"Bearer $TOKEN" 
- http --form POST "http://keycloak.test/api/dm/restart?name=pippo&state=all"  Authorization:"Bearer $TOKEN"
- http  "http://keycloak.test/api/dm/listbyimage?imgname=nginx&state=all" Authorization:"Bearer $TOKEN"
- http DELETE http://keycloak.test/api/dm/removexitbyimage?imgname=nginx Authorization:"Bearer $TOKEN" 
- http --form POST  http://keycloak.test/api/dm/stopallcontainerbyimage?imgname=nginx  Authorization:"Bearer $TOKEN"
- http  "http://keycloak.test/api/dm/exited"  Authorization:"Bearer $TOKEN"
- http --form POST "http://keycloak.test/api/dm/create?name=ciccio&network=bridge&imgname=nginx"  Authorization:"Bearer $TOKEN"
- http  "http://keycloak.test/api/dm/volumelist"  Authorization:"Bearer $TOKEN
- http --form --json POST http://keycloak.test/api/dm/create Authorization:"Bearer $TOKEN"  <<< '{"Name":"davide1","Image":"nginx","Network":"bridge","VolumeType":"Bind","MountPo":"/mnt","Volume":"bindvolume"}'
- http --form --json POST http://keycloak.test/api/dm/create Authorization:"Bearer $TOKEN"  <<< '{"Name":"davide1","Image":"nginx","Network":"bridge",- "VolumeType":"Bind","MountPo":"/mnt","/Volume":"Volume"}
- http --form --json POST http://keycloak.test/api/dm/create Authorization:"Bearer $TOKEN"  <<< '{"Name":"davide1","Image":"nginx","Network":"bridge","VolumeType":"None","MountPo":"","/Volume":""}
- http --form POST  http://keycloak.test/api/dm/volumecreate  Authorization:"Bearer $TOKEN"  <<< '{"Name":"VOLUME1"}'
- http --form --json POST http://keycloak.test/api/dm/volumeprunes Authorization:"Bearer $TOKEN"



## Compile code.


l'argomento in ciascuna api e' sempre il NOME e non puo' essere l'id del container o della immagine

Il codice e' diviso in pacchetti:
- httpdm : contentente gli hadler delle chiamate http
- utils per funzioni di suporto 
- keyckoak : contenente le funzioni relative alla autenticazione oauth2 
- eurekaclient : contenente le funzioni per il client eureka
- logdm : contentente le funzioni per il log su file

Per procedere alla compilazione dell'eseguibile go si deve tenere conto delle dipendenze 


main --usa---> httpdm --usa--> keycloak --usa--> logdm
               eurekaclient    utils --usa---> keycloak
               logdm                           logdm
                               logdm

in ogni paccheto si deve creare il file go.sub e go.module, e deve essere specificato il path relativo dei pacchetti che usa.

Si parte dai pacchetti che non usano altri pacchetti del nostro progetto.
Questi sono logdm,e eurekaclient.

Pacchettizzate le funzioni relative a keycloak e a Eureka.
Per questi 3 pacchetti si deve dare il seguente comando nella directory dove e' contenuto in sorgente:


(da sotto src/eureka)
 go mod init example.com/eurekaclient
 go mod tidy

 (da sotto src/logdm)
 go mod init example.com/logdm
 go mod tidy

 A salire si va nei pacchetti che usano questi due:
keycloak,utils .
In ciascuna directory di keycloak e utils si da il comando:

keycloak:

go mod init example.com/keycloak
go mod edit -replace example.com/logdm=./logdm
go mod tidy

utils:

go mod init example.com/utils
go mod edit -replace example.com/logdm=../logdm
go mod edit -replace example.com/keycloak=../keycloak
go mod tidy

A salire ancora c'e' httpdm
httpdm:

go mod init example.com/httpdm
go mod edit -replace example.com/logdm=../logdm
go mod edit -replace example.com/utils=../utils
go mod edit -replace example.com/keycloak=../keycloak
go mod tidy

main:
go mod init example.com/main
go mod edit -replace example.com/logdm=./logdm
go mod edit -replace example.com/eurekaclient=./eurekaclient
go mod edit -replace example.com/logdm=./logdm
go mod tidy
