package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"net/http"
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

// writeToken writes out the oauth2 token to a file
func writeToken(fileName string, token *oauth2.Token) {
	// create token file
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	// write access token string
	_, err = file.WriteString(token.AccessToken)
	if err != nil {
		log.Fatal(err)
	}

	return
}

// appVars contains shared variables to avoid use of globals
// TODO - should I use Context instead?
type appVars struct {
	conf     *oauth2.Config
	ctx      context.Context
	client   *http.Client
	state    string
	authChan chan bool
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
	fmt.Printf("%v", token)

	// write out token to file for reuse in other programs
	writeToken("token.txt", token)

	// update HTTP client with token
	app.client = app.conf.Client(app.ctx, token)

	fmt.Fprintf(w, "Authorization successful, token.txt update")

	// signal that authorization was successful
	app.authChan <- true

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
// token is written to token.txt file in the current directory
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
	//	msClientSecret, present := os.LookupEnv("MSCLIENTSECRET")
	//	if !present {
	//		log.Panic("Must set MSCLIENTSECRET")
	//	}

	// setup configuration for OAuth2
	app.conf = &oauth2.Config{
		ClientID: msClientId,
		//		ClientSecret: msClientSecret,
		Scopes: []string{"Notes.Read"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  msAuthURL,
			TokenURL: msTokenURL,
		},
		RedirectURL: myRedirectURL,
	}

	// generate a random token to prevent CSRF attacks
	app.state = randomToken(32)

	// generate URL for user consent for permissions (scopes) above
	url := app.conf.AuthCodeURL(app.state, oauth2.AccessTypeOffline)

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

	// register redirect handler
	http.HandleFunc("/oauth/callback", app.oauthRedirect)

	// channel for authorization status
	app.authChan = make(chan bool)

	// startup local http server in a go task
	log.Println("Waiting for redirect")
	go func() {
		log.Fatal(httpServer.ListenAndServe())
	}()

	// get auth result via channel
	authorized := <-app.authChan
	if !authorized {
		log.Panic("Authorization failed\n")
	}

	// try something
	const tryURL = "https://graph.microsoft.com/v1.0/me/onenote/notebooks?$select=displayName"
	fmt.Println("\nGET", tryURL)
	resp, err := app.client.Get(tryURL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n%v\n", string(body))

}
