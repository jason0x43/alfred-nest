package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jason0x43/go-alfred"
)

var OauthApiHost = "https://api.home.nest.com/oauth2/access_token"
var OauthTitle = "Alfred Nest"

var listener net.Listener

type closeableListener struct {
	net.Listener
}

func StartAuthServer() (err error) {
	listener, err = net.Listen("tcp", ":"+CallbackPort)
	if err != nil {
		return
	}

	http.HandleFunc(CallbackPath, oauthHandler)
	return http.Serve(closeableListener{listener}, nil)
}

func (l closeableListener) Accept() (c net.Conn, err error) {
	c, err = l.Listener.Accept()
	if err != nil {
		if opErr, ok := err.(*net.OpError); ok && opErr.Op == "accept" {
			// accept error -- shutdown silently
			log.Printf("shutting down nicely")
			os.Exit(0)
		}
	}
	return
}

func oauthHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received OAuth request")

	params, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		log.Fatal("error parsing query:", err)
	}

	oauth_params := url.Values{}
	oauth_params.Set("code", params.Get("code"))
	oauth_params.Set("client_id", ClientId)
	oauth_params.Set("client_secret", ClientSecret)
	oauth_params.Set("grant_type", "authorization_code")

	log.Println("POSTing to " + OauthApiHost)

	req, err := http.NewRequest("POST", OauthApiHost, strings.NewReader(oauth_params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal("error POSTing to Nest:", err)
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Println("error reading response:", err)
		writeResponse("<h1>Authorization failed</h1><p>"+
			err.Error()+"</p>", "fail", w, r)
	} else if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		log.Printf("bad response code (%d): %s", resp.StatusCode, content)
		writeResponse("<h1>Authorization failed</h1><p>"+
			string(content)+"</p>", "fail", w, r)
	} else {
		var message struct {
			AccessToken string `json:"access_token"`
			ExpiresIn   int64  `json:"expires_in"`
		}

		if err := json.Unmarshal(content, &message); err != nil {
			writeResponse("<h1>Authorization failed</h1><p>"+
				err.Error()+"</p>", "fail", w, r)
		} else {
			log.Printf("Unmarshaled '%s' into %#v\n", string(content), message)

			// save the access token to the workflow config file
			config.AccessToken = message.AccessToken
			config.AccessExpiry = time.Now().Add(time.Duration(message.ExpiresIn) * time.Second)
			err := alfred.SaveJson(configFile, &config)

			if err != nil {
				writeResponse(`<h1>Authorization failed</h1>
					<p>The authorization process itself passed, but there was an
					error saving the token:</p>
					<pre>`+err.Error()+`</pre>`,
					"fail", w, r)
			} else {
				writeResponse("<h1>Authorization was successful!</h1>"+
					"<p>You may now close this window/tab.</p>", "success", w, r)
			}
		}
	}

	log.Printf("Shutting down...")
	listener.Close()
}

func writeResponse(content, class string, w http.ResponseWriter, r *http.Request) {
	log.Printf("Writing response...")
	fmt.Fprintf(w, "<!DOCTYPE html>\n"+
		"<html><head>"+
		"<title>"+OauthTitle+"</title>"+
		"<style>"+
		"body{font-family:sans-serif}"+
		"h1{font-size:20px}"+
		"body>div{"+
		"width:400px;margin:50px auto;text-align:center;"+
		"border:solid 1px transparent;"+
		"}"+
		"body.fail>div{background: #fdd}"+
		"body.success>div{background: #cfc}"+
		"</style>"+
		"</head><body class=\"%s\"><div>%s</div></body></html>",
		class, content)
}
