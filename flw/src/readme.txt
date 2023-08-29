Pacchettizzate le funzioni relative a keycloak e a Eureka.

Per testare che la pacchettizzazione e' effettiva ho utilizzato il docker-manager, e scritto un dockerfile nuovo per creare l'immagine.Ho messo questa immagine poi nel compose del docker-manager.

Ad ogni modo , a parte il main che e' stato usato come test, la pacchettizzazione rimane valida per ogni main.

Sono presenti, sul repo, 3 directory:
- src che contiene le directory del codice dei pacchetti (in sottodirectory )e del main .
- pkg che contiene i pacchetti usati .
- bin che contiene gli eseguibili.


Le directory:
-./src/keycloak contiene il package keycloak .
-./src/eureka contine il package eurekaclient

Prima di generare l'immagine bisogna generare i file go.mod e go.sum per i pacchetti e per il main.
I passi da effettuare sono i seguenti:

(da sotto src/keycloak )
 go mod init example.com/keycloak
 go mod tidy


(da sotto src/eureka)
 go mod init example.com/eurekaclient
 go mod tidy

(da sotto src)
 go mod init example.com/main
 go mod edit -replace example.com/keycloak=./keycloak
 go mod edit -replace example.com/eurekaclient=./eurekaclient
 go mod tidy


I pacchetti nel main saranno importati con example.com, i comandi go mod edit .... servono appunto a creare il legame tra il nome del pacchetto e la posizione nel filesystem . Fatto questo tutta la struttura /src /pkg /bin viene copiata nel container dal dockerfile, viene generato un eseguibile per il main.go che sara' sotto bin e questo e' l'entrypoint del container.

La workdir del container e' /app. Sotto /app vengono copiate tutte le directory ./src ./pkg  ./bin (per questo bisogna dare il comando docker build una directory sopra ./src ) e vengono copiati anche tutti i go.sum e go.mod.

Il dockerfile e' stato cambiato in modo da produrre un eseguibile sotto /app/bin del container.

FROM golang:1.17.7-alpine
WORKDIR /app
COPY . . 
ENV GOPATH=/app
RUN cd src && go install
EXPOSE 8080
CMD [ "bin/main" ]











