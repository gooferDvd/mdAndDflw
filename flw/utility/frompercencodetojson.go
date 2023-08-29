package main

import (
    "encoding/json"
    "net/url"
    "fmt"
)

func main() {
    // Stringa codificata in percent encoding
    //encodedData := "json=%7B%22name%22%3A+%22John+Doe%22%2C+%22age%22%3A+30%7D"
    encodedData := "json=%7B%22dangling%22%3A%5B%22true%22%5D%7D"

    // Decodifica la stringa percent-encoded
    decodedData, _ := url.QueryUnescape(encodedData)

    // Estrai la stringa JSON dalla stringa decodificata
    jsonStr := decodedData[5:]

    // Decodifica la stringa JSON in un oggetto Go
    var data map[string]interface{}
    json.Unmarshal([]byte(jsonStr), &data)

    fmt.Println(data)
}

