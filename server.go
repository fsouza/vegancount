package main

import (
	"encoding/json"
	"fmt"
	"github.com/andreadipersio/goauth-facebook/facebook"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/render"
	r "github.com/dancannon/gorethink"
	"github.com/tsuru/config"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type Config struct {
	FB_ID     string
	FB_SECRET string
}

type VeganRsp struct {
	ID        string
	Location  map[string]string
	Languages []map[string]string
	Email     string
	Name      string `json:"first_name"`
}

type Vegan struct {
	ID        string   `gorethink:"id"`
	Token     string   `gorethink:"token"`
	Location  string   `gorethink:"location"`
	Email     string   `gorethink:"email"`
	Name      string   `gorethink:"name"`
	Languages []string `gorethink:"languages"`
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

	fbHandler := &facebook.GraphHandler{
		Key:    fb_id,
		Secret: fb_secret,

		RedirectURI: "http://www.vegancount.com/oauth/facebook",

		Scope: []string{"email", "user_location", "user_likes"},

		ErrorCallback: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, fmt.Sprintf("OAuth error - %v", err), 500)
		},

		SuccessCallback: func(w http.ResponseWriter, rq *http.Request, token *facebook.Token) {
			http.SetCookie(w, &http.Cookie{
				Name:  "facebook_token",
				Value: token.Token,
			})
			rsp, _ := http.Get(fmt.Sprintf("https://graph.facebook.com/me?access_token=%s", token.Token))
			defer rsp.Body.Close()
			body, _ := ioutil.ReadAll(rsp.Body)
			var veganRsp VeganRsp
			json.Unmarshal(body, &veganRsp)
			languages := make([]string, len(veganRsp.Languages))
			for i, language := range veganRsp.Languages {
				languages[i] = language["name"]
			}
			vegan := Vegan{
				ID:        veganRsp.ID,
				Token:     token.Token,
				Email:     veganRsp.Email,
				Location:  veganRsp.Location["name"],
				Name:      veganRsp.Name,
				Languages: languages,
			}
			r.Table("vegans").Insert(vegan).RunWrite(sess)
			http.SetCookie(w, &http.Cookie{
				Name:    "facebook_id",
				Value:   veganRsp.ID,
				Expires: time.Now().Add(time.Hour * 24),
				MaxAge:  86400,
				Path:    "/",
			})
			http.Redirect(w, rq, "/", http.StatusFound)
		},
	}

	m := martini.Classic()
	m.Use(render.Renderer(render.Options{Layout: "layout"}))
	m.Get("/", func(render render.Render, rq *http.Request) {
		var count int
		rsp, _ := r.Table("vegans").Count().RunRow(sess)
		rsp.Scan(&count)
		_, err := rq.Cookie("facebook_id")
		if err != nil {
			render.HTML(200, "index", map[string]interface{}{"cookie": false, "count": count})
		} else {
			//var vegan Vegan
			//rsp, _ := r.Table("vegans").Get(cookie.Value).RunRow(sess)
			//rsp.Scan(&vegan)
			render.HTML(200, "index", map[string]interface{}{"cookie": true, "count": count})
		}
	})
	m.Get("/oauth/facebook", fbHandler.ServeHTTP)
	m.Run()
}
