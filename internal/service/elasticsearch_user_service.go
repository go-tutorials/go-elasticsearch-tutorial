package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"reflect"
	"strings"

	"go-service/internal/model"
)

type ElasticSearchUserService struct {
	elastic *elasticsearch.Client
}

func NewUserService(ela *elasticsearch.Client) *ElasticSearchUserService {
	return &ElasticSearchUserService{elastic: ela}
}

func convertDocToJson(doc interface{}) string {
	jsonString, err := json.Marshal(doc)

	if err != nil {
		fmt.Println("An error is happening when encoded the new user: ", err)
		return ""
	}
	return string(jsonString)
}

func (e *ElasticSearchUserService) All(ctx context.Context) ([]model.User, error) {
	var users []model.User
	var mapResponse map[string]interface{}
	var buf bytes.Buffer

	query := `{
  "query": {
    "match_all": {}
  }
}`

	var queryString = strings.NewReader(query)

	err := json.NewEncoder(&buf).Encode(&queryString)
	if err != nil {
		fmt.Print("error during encoding the query: ", err.Error())
		return users, err
	}

	result, err := e.elastic.Search(
		e.elastic.Search.WithContext(ctx),
		e.elastic.Search.WithIndex("users"),
		e.elastic.Search.WithBody(queryString),
		e.elastic.Search.WithTrackTotalHits(true),
		e.elastic.Search.WithPretty(),
	)
	defer result.Body.Close()

	err = json.NewDecoder(result.Body).Decode(&mapResponse)
	fmt.Println("This is map response: ", mapResponse)
	if err != nil {
		fmt.Println("Error parsing the result to User type:", err.Error())
		return users, err
	}

	var u = &model.User{}
	for _, hit := range mapResponse["hits"].(map[string]interface{})["hits"].([]interface{}) {
		user := hit.(map[string]interface{})

		source := user["_source"]
		u.Id = user["_id"].(string)
		fmt.Println("This is the source:")
		fmt.Println(source)

		bytes, err := json.Marshal(source)
		if err != nil {
			return users, err
		}
		err = json.Unmarshal(bytes, u)
		if err != nil {
			return users, err
		}
		users = append(users, *u)

	}
	return users, nil
}

func (e *ElasticSearchUserService) Load(ctx context.Context, id string) (*model.User, error) {
	var mapResponse map[string]interface{}
	var buf bytes.Buffer

	query := `{
  "query": {
    "match": { "id": "{0}" }
  },
  "size": 1
}`

	query = strings.Replace(query, "{0}", id, 1)
	var queryString = strings.NewReader(query)

	err := json.NewEncoder(&buf).Encode(&queryString)
	if err != nil {
		fmt.Print("Error during encoding the query : ", err.Error())
		return nil, err
	}

	result, err := e.elastic.Search(
		e.elastic.Search.WithContext(ctx),
		e.elastic.Search.WithIndex("users"),
		e.elastic.Search.WithBody(queryString),
		e.elastic.Search.WithTrackTotalHits(true),
		e.elastic.Search.WithPretty(),
	)
	defer result.Body.Close()

	err = json.NewDecoder(result.Body).Decode(&mapResponse)
	if err != nil {
		return nil, err
	}
	fmt.Println("This is map response: ", mapResponse)
	if err != nil {
		fmt.Println("Error parsing the result to User type:", err.Error())
	}

	var u = &model.User{}
	for _, hit := range mapResponse["hits"].(map[string]interface{})["hits"].([]interface{}) {
		user := hit.(map[string]interface{})

		source := user["_source"]
		fmt.Println("This is the source:")
		fmt.Println(source)

		bytes, err:= json.Marshal(source)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(bytes, u)
		if err != nil {
			return nil, err
		}
		return u, nil
	}
	return nil, nil
}

func (e *ElasticSearchUserService) Insert(ctx context.Context, user *model.User) (int64, error) {
	if user == nil {
		fmt.Print("Can not add null user")
		return 0, nil
	}

	userJsonString := convertDocToJson(user)
	request := esapi.IndexRequest{
		Index:      "users",
		DocumentID: user.Id,
		Body:       strings.NewReader(userJsonString),
		Refresh:    "true",
	}
	response, err := request.Do(ctx, e.elastic)

	if err != nil {
		return 0, err
	}

	defer response.Body.Close()

	var result map[string]interface{}

	err = json.NewDecoder(response.Body).Decode(&result)

	if err != nil {
		return -1, err
	}

	fmt.Println("IndexRequest to insert Status: ", response.Status())
	fmt.Println("Result: ", result["result"])

	fmt.Printf("the new user %v has been added successfully", user.Username)
	return 1, nil
}

func (e *ElasticSearchUserService) Update(ctx context.Context, user *model.User) (int64, error) {
	query := map[string]interface{}{
		"doc": user,
	}
	request := esapi.UpdateRequest{
		Index:      "users",
		DocumentID: user.Id,
		Body:       esutil.NewJSONReader(query),
		Refresh:    "true",
	}
	response, err := request.Do(ctx, e.elastic)

	if err != nil {
		return -1, err
	}

	defer response.Body.Close()

	var result map[string]interface{}

	err = json.NewDecoder(response.Body).Decode(&result)

	if err != nil {
		return -1, err
	}

	fmt.Println("IndexRequest to update Status: ", response.Status())
	fmt.Println("Result: ", result)

	fmt.Printf("the user %v has been updated successfully", user.Username)
	return 1, nil
}

func (e *ElasticSearchUserService) Patch(ctx context.Context, user map[string]interface{}) (int64, error) {
	var userid = reflect.ValueOf(user["id"])
	delete(user, "id")
	request := esapi.UpdateRequest{
		Index:      "users",
		DocumentID: userid.String(),
		Body:       esutil.NewJSONReader(map[string]interface{}{"doc": user}),
		Refresh:    "true",
	}
	response, err := request.Do(ctx, e.elastic)

	if err != nil {
		return -1, err
	}

	defer response.Body.Close()

	var result map[string]interface{}

	err = json.NewDecoder(response.Body).Decode(&result)

	if err != nil {
		return -1, err
	}

	fmt.Println("IndexRequest to update Status: ", response.Status())
	fmt.Println("Result: ", result["result"])

	fmt.Printf("the user %v has been updated successfully.", userid.String())
	return 1, nil
}

func (e *ElasticSearchUserService) Delete(ctx context.Context, id string) (int64, error) {
	request := esapi.DeleteRequest{
		Index:      "users",
		DocumentID: id,
	}
	response, err := request.Do(ctx, e.elastic)

	if err != nil {
		return -1, err
	}

	defer response.Body.Close()

	var result map[string]interface{}

	err = json.NewDecoder(response.Body).Decode(&result)

	if err != nil {
		return -1, err
	}

	fmt.Println("IndexRequest to update Status: ", response.Status())
	fmt.Println("Result: ", result["result"])

	fmt.Printf("delete user: %s successfully", id)
	return 1, nil
}
