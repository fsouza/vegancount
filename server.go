package main

import (
  "fmt"
  "log"
  "net/http"
  "io/ioutil"
  "encoding/json"
  "github.com/codegangsta/martini"
  r "github.com/dancannon/gorethink"
  "github.com/andreadipersio/goauth-facebook/facebook"
  config "github.com/globocom/config"
)

type Config struct {
  FB_ID string
  FB_SECRET string
}

type VeganRsp struct {
  ID string
  Location map[string]string
  Email string
}

type Vegan struct {
  ID string `gorethink:"id"`
  Token string `gorethink:"token"`
  Location string `gorethink:"location"`
  Email string `gorethink:"email"`
}

func main() {
  sess, err := r.Connect(map[string]interface{}{
      "address":  "localhost:28015",
      "database": "vegancount",
    })

    if err != nil {
        log.Fatalln(err.Error())
    }

  config.ReadConfigFile("secrets.yml")
  fb_id, _ := config.GetString("fb_id")
  fb_secret, _ := config.GetString("fb_secret")

  fbHandler := &facebook.GraphHandler {
    Key: fb_id,
    Secret: fb_secret,

    RedirectURI: "http://saiko.luxhaven.com/oauth/facebook",

    Scope: []string{"email", "user_location"},

    ErrorCallback: func(w http.ResponseWriter, r *http.Request, err error) {
      http.Error(w, fmt.Sprintf("OAuth error - %v", err), 500)
    },

    SuccessCallback: func(w http.ResponseWriter,  rq *http.Request, token *facebook.Token) {
      http.SetCookie(w, &http.Cookie{
        Name: "facebook_token",
        Value: token.Token,
      })
      rsp, _ := http.Get(fmt.Sprintf("https://graph.facebook.com/me?access_token=%s", token.Token))
      defer rsp.Body.Close()
      body, _ := ioutil.ReadAll(rsp.Body)
      var veganRsp VeganRsp
      json.Unmarshal(body, &veganRsp)
      vegan := Vegan{ID: veganRsp.ID, Token: token.Token, Email: veganRsp.Email, Location: veganRsp.Location["name"]}
      r.Table("vegans").Insert(vegan).RunWrite(sess)
      http.Redirect(w, rq, "/", http.StatusFound)
    },
  }


  m := martini.Classic()
  m.Get("/", func() string {
    return "Hello world!"
  })
  m.Get("/oauth/facebook", fbHandler.ServeHTTP)
  m.Run()
}