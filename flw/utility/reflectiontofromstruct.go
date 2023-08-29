package main

import (
	"fmt"
	"reflect"
)

type InnerStruct struct {
	A int
	B string
}

type MyStruct struct {
	X      int
	Y      int
	Z      int
	Name   InnerStruct
	Vector []string
}

var activation int = 0

func structToMap(i interface{}) map[string]interface{} {
	k := make(map[string]interface{})
	iVal := reflect.ValueOf(i)             //torna il Value di i
	t := iVal.Type()                       //torna il type perche da questo si possono desumere il nome dei campi
	for i := 0; i < iVal.NumField(); i++ { //nymfield info che e' anche in value
		ft := t.Field(i)       //da ft prendo il campo i esimo ---> questo mi da il nome del campo
		fival := iVal.Field(i) //e il Value del campo --> questo mi da il  il tipo e l'oggetto prossimo della ricorsione
		switch fival.Kind() {
		case reflect.Struct:
			k[ft.Name] = printFields(fival.Interface())
		default:
			k[ft.Name] = fival.Interface()
		}
	}
	return k

}

func mapToStruct(m map[string]interface{}, i interface{}) {
	//deferenzia i, i e' la struttura vuota
	v := reflect.ValueOf(i).Elem()
	//k chiave,val valore della mappa
	for k, val := range m {

		fmt.Println(k, val)
		// dalla struttura vuota preleva il Value del campo k.
		f := v.FieldByName(k)

		if f.IsValid() && f.CanSet() {
			valValue := reflect.ValueOf(val) // costruisci dal val della mappa un Value
			//questo ci serve per vedere se il kind della mappa e della s
			//struttura vuota sono gli stessi
			if valValue.Kind() == reflect.Map && f.Kind() == reflect.Struct {
				// Convert map to struct
				//costruisci una struct vuota pari al campo k , new torna un value che e'
				//un puntatore a una struttura di tipo f.kind().quindi interface() e' una
				//struttura vuota e nella parte valore punta a una struttura vuota .
				newVal := reflect.New(f.Type()).Interface()
				//verifica che val e' una mappa , e richiama la ricorsione
				mapToStruct(val.(map[string]interface{}), newVal)
				//la struct newval ora e' riempita ricorsivanente e
				f.Set(reflect.ValueOf(newVal).Elem())
			} else {
				f.Set(valValue)
			}
		}
	}
}
func main() {
	s := MyStruct{
		X: 1,
		Y: 2,
		Z: 3,
		Name: InnerStruct{
			A: 6,
			B: "DAVIDE",
		},
		Vector: []string{"1", "2", "3"},
	}
	z := structToMap(s)
	fmt.Println(z)
	k := MyStruct{}
	mapToStruct(z, &k)
	fmt.Println(k)
}
