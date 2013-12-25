package main

import (
  "fmt"
  "net/http"
  "github.com/codegangsta/martini"
  "github.com/andreadipersio/goauth-facebook/facebook"
  config "github.com/globocom/config"
)

type Config struct {
  FB_ID string
  FB_SECRET string
}

func main() {
  config.ReadConfigFile("secrets.yml")
  fb_id, _ := config.GetString("fb_id")
  fb_secret, _ := config.GetString("fb_secret")

  fbHandler := &facebook.GraphHandler {
    Key: fb_id,
    Secret: fb_secret,

    RedirectURI: "http://saiko.luxhaven.com/oauth/facebook",

    Scope: []string{"email"},

    ErrorCallback: func(w http.ResponseWriter, r *http.Request, err error) {
      http.Error(w, fmt.Sprintf("OAuth error - %v", err), 500)
    },

    SuccessCallback: func(w http.ResponseWriter,  r *http.Request, token *facebook.Token) {
      http.SetCookie(w, &http.Cookie{
        Name: "facebook_token",
        Value: token.Token,
      })
    },
  }


  m := martini.Classic()
  m.Get("/", func() string {
    return "Hello world!"
  })
  m.Get("/oauth/facebook", fbHandler.ServeHTTP)
  m.Run()
}