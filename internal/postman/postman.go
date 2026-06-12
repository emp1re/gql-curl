package postman

import (
	"bytes"
	"encoding/json"
	"strings"
)

const CollectionSchema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"

type Collection struct {
	Info Info   `json:"info"`
	Item []Item `json:"item"`
}

type Info struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
}

type Item struct {
	Name    string   `json:"name"`
	Item    []Item   `json:"item,omitempty"`
	Request *Request `json:"request,omitempty"`
}

type Request struct {
	Method string   `json:"method"`
	Header []Header `json:"header,omitempty"`
	Body   Body     `json:"body"`
	URL    string   `json:"url"`
}

type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type Body struct {
	Mode    string      `json:"mode"`
	GraphQL GraphQLBody `json:"graphql"`
}

type GraphQLBody struct {
	Query     string `json:"query"`
	Variables string `json:"variables"`
}

func NewCollection(name string, items []Item) Collection {
	return Collection{
		Info: Info{
			Name:   name,
			Schema: CollectionSchema,
		},
		Item: items,
	}
}

func NewFolder(name string, items []Item) Item {
	return Item{
		Name: name,
		Item: items,
	}
}

func NewGraphQLRequestItem(name, endpoint string, headers []Header, query, variablesJSON string) Item {
	return Item{
		Name: name,
		Request: &Request{
			Method: "POST",
			Header: headers,
			URL:    endpoint,
			Body: Body{
				Mode: "graphql",
				GraphQL: GraphQLBody{
					Query:     query,
					Variables: variablesJSON,
				},
			},
		},
	}
}

func Encode(collection Collection) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(collection); err != nil {
		return nil, err
	}

	return []byte(strings.TrimSpace(buf.String()) + "\n"), nil
}
