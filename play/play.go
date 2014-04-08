package play

import (
  "encoding/json"
  "fmt"
  "net/http"
  "github.com/lestrrat/go-xslate"
)

var tx *xslate.Xslate
func init() {
  http.HandleFunc("/api/render", serveApiRender)
  http.HandleFunc("/", serveIndex)
}

func createXslate() *xslate.Xslate {
  if tx != nil {
    return tx
  }
  tx, err := xslate.New(xslate.Args {
    "Loader": xslate.Args {
      "LoadPaths": []string { "./templates" },
    },
  })

  if err != nil {
    panic("Could not instantiate xslate!")
  }

  return tx
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
  http.ServeFile(w, r, "templates/index.html")
}

func serveApiRender(w http.ResponseWriter, r *http.Request) {
  tx := createXslate()

  var vars map[string]interface {}
  if varString := r.FormValue("variables"); varString != "" {
    err := json.Unmarshal([]byte(varString), &vars)
    if err != nil {
      w.WriteHeader(500)
      fmt.Fprintf(w, "Failed to decode variables: %s", err)
      return
    }
  }

  output, err := tx.RenderString(r.FormValue("template"), xslate.Vars(vars))
  if err != nil {
    w.WriteHeader(500)
    fmt.Fprintf(w, err.Error())
    return
  }

  fmt.Fprintf(w, output)
}