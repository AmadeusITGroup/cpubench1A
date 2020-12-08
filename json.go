package main

import (
	"bytes"
	"encoding/json"
	"log"

	"github.com/tidwall/gjson"
)

// BenchJson is a JSON decoding/encoding benchmark
type BenchJson struct {
	input []JsonPerson
	buf   bytes.Buffer
}

// JsonPerson is a record representing an actor
type JsonPerson struct {
	Firstname string
	Lastname  string
	Id        int
	Salary    float64
	Female    bool
}

// NewBenchJson allocates a new benchmark object
func NewBenchJson() *BenchJson {

	// Random data
	persons := []JsonPerson{
		{"Tom", "Cruise", 1, 1000000.0, false},
		{"Nicole", "Kidman", 2, 2000000.0, true},
		{"Bob", "Layton", 3, 1234.0, false},
		{"Gerard", "Depardieu", 4, 100000000.0, false},
		{"Mathilda", "May", 5, 100000.0, true},
		{"Audrey", "Hepburn", 6, 10000.0, true},
		{"Ava", "Gardner", 7, 1000000000.0, true},
		{"Lauren", "Bacall", 8, 123123123.12, true},
		{"John", "Wayne", 9, 4000000.123, false},
		{"Anthony", "Perkins", 10, 54321.0, false},
		{"Kevin", "Bacon", 11, 8764435.432, false},
		{"Edward", "Norton", 12, 3685346.436, false},
		{"Robert", "Wagner", 13, 3214.432, false},
		{"Stephanie", "Powers", 14, 43253245234.12, true},
		{"Charlize", "Theron", 15, 15134363.1234, true},
		{"Kate", "Winslet", 16, 131432.43, true},
	}

	return &BenchJson{input: persons}
}

// Run does execute a number of JSON queries, and then encode a JSON object
func (b *BenchJson) Run() {

	gjson.Get(jsonPopulation, "0.lastupdated")
	gjson.Get(jsonPopulation, "1.#.date")
	gjson.Get(jsonPopulation, "1.#.value")

	gjson.Get(jsonAirlines, `#(Airport.Code=="MCO").Statistics.Carriers.Names`)
	gjson.Get(jsonAirlines, `#(Airport.Code=="SFO").Statistics.Carriers.Names`)

	gjson.Get(jsonAirlines, `#(Airport.Code=="MCO").Statistics.Flights`)
	gjson.Get(jsonAirlines, `#(Airport.Code=="SFO").Statistics.Flights`)

	encoder := json.NewEncoder(&b.buf)
	for i := 0; i < 64; i++ {
		if err := encoder.Encode(b.input); err != nil {
			log.Fatal(err)
		}
	}
	b.buf.Reset()
}
