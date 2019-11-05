package onenote

import (
	"encoding/json"
	//	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type OData struct {
	ODataContext  string `json:"@odata.context"`
	ODataCount    int    `json:"@odata.count"`
	ODataNextLink string `json:"@odata.nextLink"`
}

type NotebookResponse struct {
	OData
	Value []Notebook
}

type Identity struct {
	Id          string `json:"id,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

// using pointer to Identity so Unmarshal creates a nil on empty
// (see https://stackoverflow.com/questions/33447334/golang-json-marshal-how-to-omit-empty-nested-struct)
type IdentitySet struct {
	Application *Identity `json:"application,omitempty"`
	Device      *Identity `json:"device,omitempty"`
	User        *Identity `json:"user,omitempty"`
}

type NotebookLinks struct {
	OneNoteClientUrl ExternalLink `json:"oneNoteClientUrl"`
	OneNoteWebUrl    ExternalLink `json:"oneNoteWebUrl"`
}

type ExternalLink struct {
	Href string `json:"href"`
}

type Notebook struct {
	Id                   string        `json:"id"`
	Self                 string        `json:"self"`
	CreatedDateTime      string        `json:"createdDateTime"`
	DisplayName          string        `json:"displayName"`
	CreatedBy            IdentitySet   `json:"createdBy"`
	LastModifiedBy       IdentitySet   `json:"lastModifiedBy"`
	LastModifiedDateTime string        `json:"lastModifiedDateTime"`
	IsDefault            bool          `json:"isDefault"`
	UserRole             string        `json:"userRole"`
	IsShared             bool          `json:"isShared"`
	SectionsUrl          string        `json:"sectionsUrl"`
	SectionGroupsUrl     string        `json:"sectionsGroupsUrl"`
	Links                NotebookLinks `json:"links"`
}

type PageLinks struct {
	OneNoteClientUrl ExternalLink `json:"oneNoteClientUrl"`
	OneNoteWebUrl    ExternalLink `json:"oneNoteWebUrl"`
}

type Section struct {
	Id                   string `json:"id"`
	Self                 string `json:"self"`
	CreatedDateTime      string `json:"createdDateTime"`
	DisplayName          string `json:"displayName"`
	LastModifiedDateTime string `json:"lastModifiedDateTime"`
	IsDefault            bool   `json:"isDefault"`
	PagesUrl             string `json:"pagesUrl"`

	CreatedBy      IdentitySet   `json:"createdBy"`
	LastModifiedBy IdentitySet   `json:"lastModifiedBy"`
	Links          NotebookLinks `json:"links"`
}

type Page struct {
	Id                   string    `json:"id"`
	Self                 string    `json:"self"`
	CreatedDateTime      string    `json:"createdDateTime"`
	Title                string    `json:"title"`
	CreatedByAppId       string    `json:"createdByAppId"`
	Links                PageLinks `json:"links"`
	ContentUrl           string    `json:"contentUrl"`
	LastModifiedDateTime string    `json:"lastModifiedDateTime"`
	Content              string    `json:"content"`
	Level                int32     `json:"level"`
	Order                int32     `json:"order"`
	ParentNotebook       Notebook  `json:"parentNotebook"`
	ParentSection        Section   `json:"parentSection"`
}

type PageResponse struct {
	OData
	Value        []Page `json:"value"`
	ODataContext string `json:"@odata.context"`
}


// get is a helper function to form and execute the HTTP request
// and return the HTTP response
func get(token []byte, urlString string, query url.Values) []byte {

	// default HTTP client
	client := &http.Client{}

	// parse the URL string
	url, err := url.Parse(urlString)
	if err != nil {
		log.Fatal(err)
	}

	// add the query parameters to the URL
	url.RawQuery = query.Encode()

	//fmt.Printf("*****\n%v\n*****\n", url.String())

	// create the HTTP request with proper headers
	req, err := http.NewRequest("GET", url.String(), nil)
	req.Header.Add("Host", "graph,microsoft.com")
	req.Header.Add("Authorization", "Bearer "+string(token))

	// execute HTTP request
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// read HTTP response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// return HTTP response body
	return body
}

func unmarshal(body []byte, v interface{}) {
	err := json.Unmarshal(body, v)
	if err != nil {
		log.Fatal("Cannot unmarshal", err)
	}
}

const msBaseURL = "https://graph.microsoft.com/v1.0"

// ListNotebooks retrives a list of Notebook objects
func ListNotebooks(token []byte, query url.Values) NotebookResponse {
	var body []byte

	body = get(token, msBaseURL+"/me/onenote/notebooks", query)

	var response NotebookResponse
	unmarshal(body, &response)

	return response
}

// ListPages retrives a list of Page objects
func ListPages(token []byte, query url.Values) PageResponse {
	var body []byte

	body = get(token, msBaseURL+"/me/onenote/pages", query)

	var response PageResponse
	unmarshal(body, &response)

	return response
}

func GetPage(token []byte, id string, query url.Values) Page {
	var body []byte

	body = get(token, msBaseURL+"/me/onenote/pages/"+id, query)

	var response Page
	unmarshal(body, &response)

	return response
}

func GetPageContent(token []byte, id string, query url.Values) string {
	var body []byte

	body = get(token, msBaseURL+"/me/onenote/pages/"+id+"/content", query)

	return string(body)
}
