package main

import (
	"fmt"
	"strings"
)

type Translator interface {
	Start() Call
}

type Call interface {
	Say(word string)
	Results() <-chan string
	Done()
}

type hindiTranslator struct{}

func (hindiTranslator) Start() Call {
	return &hindiCall{out: make(chan string, 10)}
}

type hindiCall struct {
	out chan string
}

var dictionary = map[string]string{
	"hello":  "namaste",
	"water":  "paani",
	"thanks": "dhanyavaad",
}

func (c *hindiCall) Say(word string) {
	translated, ok := dictionary[strings.ToLower(word)]
	if !ok {
		translated = "(unknown)"
	}
	c.out <- translated
}

func (c *hindiCall) Results() <-chan string {
	return c.out
}

func (c *hindiCall) Done() {
	close(c.out)
}

func main() {
	var t Translator = hindiTranslator{}

	call := t.Start()

	go func() {
		call.Say("hello")
		call.Say("water")
		call.Say("thanks")
		call.Say("pizza")
		call.Done()
	}()

	for result := range call.Results() {
		fmt.Println("heard back:", result)
	}
}
