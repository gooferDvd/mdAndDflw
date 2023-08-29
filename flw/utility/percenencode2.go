package main

import (
    "encoding/json"
    "net/url"
    "fmt"
)
//{"dangling":["true"]}
func main() {
    // Oggetto JSON da codificare
    data := map[string]interface{}{
      "status": []string{"exited"},
    }

    // Converti l'oggetto JSON in una stringa
    jsonStr, _ := json.Marshal(data)

    // Codifica la stringa JSON in percent encoding
    encodedData := url.Values{}
    encodedData.Set("json", string(jsonStr))

    fmt.Println(encodedData.Encode())
}

