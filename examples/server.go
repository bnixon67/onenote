package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/bnixon67/onenote"
	"golang.org/x/oauth2"
	//	"io/ioutil"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
)

const (
	msBase        = "https://login.microsoftonline.com/common/oauth2/v2.0"
	msAuthURL     = msBase + "/authorize"
	msTokenURL    = msBase + "/token"
	myRedirectURL = "http://localhost:9999/oauth/callback"
)

// appVars contains shared variables to avoid use of globals
// TODO - should I use Context instead?
type appVars struct {
	conf     *oauth2.Config
	ctx      context.Context
	client   *http.Client
	state    string
	authChan chan bool
	token    *oauth2.Token
}

func (app *appVars) login(w http.ResponseWriter, r *http.Request) {
	const tpl = `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}}</title>
	</head>
	<body>
	<a href="{{.Url}}">Microsoft Login</a> 
	</body>
</html>`

	t, err := template.New("login").Parse(tpl)
	if err != nil {
		log.Fatal(err)
	}

	data := struct {
		Title string
		Url   string
	}{}

	data.Title = "OneNote login"

	// generate URL for user consent for permissions (scopes) above
	data.Url = app.conf.AuthCodeURL(app.state, oauth2.AccessTypeOffline)

	err = t.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}
}

func (app *appVars) listNotebooks(w http.ResponseWriter, r *http.Request) {
	const tpl = `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}}</title>
	</head>
	<body>
	{{range .Notebooks}}<div>{{ .}}</div>{{else}} <div><strong>no rows</strong></div>{{end}}
	</body>
</html>`

	t, err := template.New("notesbooks").Parse(tpl)
	if err != nil {
		log.Fatal(err)
	}

	data := struct {
		Title     string
		Notebooks []string
	}{}

	data.Title = "List Notebooks"

	var query url.Values
	var nextLink *url.URL

	// list of notebooks may be returned by multiple queries
	// @odata.nextLink has the link to next set of pages
	for {

		// ----- List Notebooks

		// first query (no nextLink)
		if nextLink == nil {
			query = url.Values{}
		} else {
			// set query value based on nextLink
			query = nextLink.Query()
		}

		// get pages
		notebooksResponse := onenote.ListNotebooks([]byte(app.token.AccessToken), query)

		// get nextLink (if any)
		nextLink, _ = url.Parse(notebooksResponse.ODataNextLink)

		// loop thru each page
		for _, notebook := range notebooksResponse.Value {

			data.Notebooks = append(data.Notebooks, notebook.DisplayName)

		}

		// nextLink is empty, so exit loop
		if notebooksResponse.ODataNextLink == "" {
			break
		}

	}

	err = t.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}
}

func (app *appVars) listPages(w http.ResponseWriter, r *http.Request) {
	log.Printf("*** listPages ***")

	const tpl = `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}}</title>
	</head>
	<body>
	{{range .Pages}}<div>{{ .}}</div>{{else}} <div><strong>no rows</strong></div>{{end}}
	</body>
</html>`

	t, err := template.New("login").Parse(tpl)
	if err != nil {
		log.Fatal(err)
	}

	data := struct {
		Title string
		Pages []string
	}{}

	data.Title = "List Pages"

	var query url.Values
	var nextLink *url.URL

	log.Printf("*** before for listPages ***")

	// list of page may be returned by multiple queries
	// @odata.nextLink has the link to next set of pages
	for {

		log.Printf("*** for listPages ***")

		// ----- List Pages

		// first query (no nextLink)
		if nextLink == nil {
			query = url.Values{}

			// total number of pages
			query.Set("$count", "true")

			// sort by page title
			query.Set("$orderby", "parentSection/displayName,title")
			//query.Set("$orderby", "title")

			// exand parentNotebook to get displayName
			query.Set("$expand", "parentNotebook,parentSection")

			// filter on just one Notebook
			query.Set("$filter",
				"parentNotebook/displayName eq 'UMB Notes'")
		} else {
			// set query value based on nextLink
			query = nextLink.Query()
		}

		// get pages
		pagesResponse := onenote.ListPages([]byte(app.token.AccessToken), query)

		// get nextLink (if any)
		nextLink, _ = url.Parse(pagesResponse.ODataNextLink)

		// loop thru each page
		for _, page := range pagesResponse.Value {

			data.Pages = append(data.Pages, fmt.Sprintf("%s/%s/%s", page.ParentNotebook.DisplayName, page.ParentSection.DisplayName, page.Title))

		}

		// nextLink is empty, so exit loop
		if pagesResponse.ODataNextLink == "" {
			break
		}

	}

	err = t.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}
}

// oauthRedirect handles the redirect from the resource owner
func (app *appVars) oauthRedirect(w http.ResponseWriter, r *http.Request) {

	// get and compare state to prevent Cross-Site Request Forgery
	state := r.FormValue("state")
	if state != app.state {
		log.Fatalln("state is not the same (CSRF?)")
	}

	// get authorization code
	code := r.FormValue("code")

	// exchange authorization code for token
	token, err := app.conf.Exchange(app.ctx, code)
	if err != nil {
		log.Println("conf.Exchange", err)
		// signal that authorization was not successful
		app.authChan <- false
		return
	}

	// update HTTP client with token
	app.client = app.conf.Client(app.ctx, token)

	// TODO
	app.token = token

	const tpl = `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}}</title>
	</head>
	<body>
	<p>Authorization successful
	<p><a href="{{.BaseUrl}}/listNotebooks">List Notebooks</a> 
	<p><a href="{{.BaseUrl}}/listPages">List Pages</a> 
	</body>
</html>`

	t, err := template.New("authorized").Parse(tpl)
	if err != nil {
		log.Fatal(err)
	}

	data := struct {
		Title   string
		BaseUrl string
	}{}

	data.Title = "Authorized"
	data.BaseUrl = "http://localhost:9999"

	err = t.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}

	return
}

// randomToken returns a securely generated token of size n
func randomToken(n int) string {
	// buffer to store n bytes
	b := make([]byte, n)

	// read random bytes based on size of b
	_, err := rand.Read(b)
	if err != nil {
		log.Panic(err)
	}

	// convert buffer to URL friendly string
	return base64.URLEncoding.EncodeToString(b)
}

// main authorizes via OAuth2 with Microsoft
// two environmental variables must be set (MSCLIENTID and MSCLIENTSECRET)
func main() {

	app := &appVars{}

	// get top-level context
	app.ctx = context.Background()

	// get Microsoft client id and secret stored in environment
	// variables to avoid adding to source code repository
	msClientId, present := os.LookupEnv("MSCLIENTID")
	if !present {
		log.Panic("Must set MSCLIENTID")
	}
	msClientSecret, present := os.LookupEnv("MSCLIENTSECRET")
	if !present {
		log.Panic("Must set MSCLIENTSECRET")
	}

	// setup configuration for OAuth2
	app.conf = &oauth2.Config{
		ClientID:     msClientId,
		ClientSecret: msClientSecret,
		Scopes:       []string{"Notes.Read"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  msAuthURL,
			TokenURL: msTokenURL,
		},
		RedirectURL: myRedirectURL,
	}

	// generate a random token to prevent CSRF attacks
	app.state = randomToken(32)

	url := "http://localhost:9999/login"

	// prompt user to visit the URL in a browser
	// once authorization, the remote site will redirect back
	fmt.Printf("Visit the following URL to login:\n\n%v\n\n", url)

	// if possible, open the link for the user
	switch os := runtime.GOOS; os {
	case "darwin":
		cmd := exec.Command("open", url)
		err := cmd.Run()
		if err != nil {
			log.Printf("Command Run failed: %v", err)
		}
	case "windows":
		cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler",
			url)
		err := cmd.Run()
		if err != nil {
			log.Printf("Command Run failed: %v", err)
		}
	}

	// setup local server for redirect
	httpServer := &http.Server{Addr: ":9999"}

	// register handlers
	http.HandleFunc("/login", app.login)
	http.HandleFunc("/listPages", app.listPages)
	http.HandleFunc("/listNotebooks", app.listNotebooks)
	http.HandleFunc("/oauth/callback", app.oauthRedirect)

	// channel for authorization status
	app.authChan = make(chan bool)

	// startup local http server
	log.Fatal(httpServer.ListenAndServe())
}
