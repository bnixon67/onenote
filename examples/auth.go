package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/oauth2"
)

const (
	msBase        = "https://login.microsoftonline.com/common/oauth2/v2.0"
	msAuthURL     = msBase + "/authorize"
	msTokenURL    = msBase + "/token"
	myRedirectURL = "https://login.microsoftonline.com/common/oauth2/nativeclient"
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
	conf   *oauth2.Config
	ctx    context.Context
	client *http.Client
	state  string
}

func (app *appVars) redeemToken(responseURI string) {
	//w http.ResponseWriter, r *http.Request) {

	u, err := url.Parse(responseURI)
	if err != nil {
		log.Fatal(err)
	}

	// get and compare state to prevent Cross-Site Request Forgery
	state := u.Query().Get("state")
	if state != app.state {
		log.Fatalln("state is not the same (CSRF?)")
	}

	// get authorization code
	code := u.Query().Get("code")

	// exchange authorization code for token
	token, err := app.conf.Exchange(app.ctx, code)
	if err != nil {
		log.Println("conf.Exchange", err)
		return
	}

	// write out token to file for reuse in other programs
	writeToken("token.txt", token)

	// update HTTP client with token
	app.client = app.conf.Client(app.ctx, token)

	fmt.Println("Authorization successful, token.txt update")

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
	//msClientSecret, present := os.LookupEnv("MSCLIENTSECRET")
	//if !present {
	//	log.Panic("Must set MSCLIENTSECRET")
	//}

	// setup configuration for OAuth2
	app.conf = &oauth2.Config{
		ClientID: msClientId,
		//ClientSecret: msClientSecret,
		Scopes: []string{"Notes.Read", "offline_access"},
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
	fmt.Printf("Visit the following URI to login:\n\n%v\n\n", url)

	fmt.Println("Enter the response URI:")
	responseURI, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	responseURI = strings.TrimSpace(responseURI)
	fmt.Printf(">%s<\n", responseURI)
	app.redeemToken(responseURI)

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
