package play

import (
  "appengine"
  "appengine/datastore"
  "bytes"
  "crypto/sha1"
  "encoding/json"
  "fmt"
  "net/http"
  "github.com/lestrrat/go-xslate"
)

type SavedTemplate struct {
  Template string
  Variables string
}

var TX *xslate.Xslate
func init() {
  http.HandleFunc("/api/render", serveApiRender)
  http.HandleFunc("/api/save", serveApiSave)
  http.HandleFunc("/p/", serveLoad)
  http.HandleFunc("/", serveIndex)
}

func createXslate() *xslate.Xslate {
  if TX != nil {
    return TX
  }
  TX, err := xslate.New(xslate.Args {
    "Loader": xslate.Args {
      "LoadPaths": []string { "./templates" },
    },
  })

  if err != nil {
    panic("Could not instantiate xslate!")
  }

  return TX
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
  tx := createXslate()
  err := tx.RenderInto(w, "index.html.tx", xslate.Vars {
    "Template": `[%- # This is where your template go -%]
Hello World [% name %]!
[%- # Variables can be specified in the box below, in JSON format -%]`,
    "Variables": `{
"name": "Bob"
}`,
  })
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
}

func serveLoad(w http.ResponseWriter, r *http.Request) {
  path := r.URL.Path
  // path starts with /p/, we need everything after that
  id := path[3:]

  c := appengine.NewContext(r)
  var st SavedTemplate
  err := datastore.Get(
    c,
    datastore.NewKey(c, "SavedTemplate", id, 0, nil),
    &st,
  )
  if err != nil {
    http.Error(w, err.Error(), http.StatusNotFound)
    return
  }

  tx := createXslate()
  err = tx.RenderInto(w, "index.html.tx", xslate.Vars {
    "Template": st.Template,
    "Variables": st.Variables,
  })
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
}

func createSavedTemplate(r *http.Request) (*SavedTemplate, error) {
  template := r.FormValue("template")
  variables := make(map[string]interface {})

  varString := r.FormValue("variables")
  if varString != "" {
    err := json.Unmarshal([]byte(varString), &variables)
    if err != nil {
        return nil, err
    }
  }

  // Normalize variables into a string
  jsonStr, err := json.Marshal(&variables)
  if err != nil {
    return nil, err
  }
  buf := &bytes.Buffer {}
  json.Compact(buf, jsonStr)

  varString = buf.String()

  st := &SavedTemplate { template, varString }

  return st, nil
}

// StringID creates a new string based on the contents of the template
// that is, if two people try to save identical templates,
// they both refer to the same ID
func (st *SavedTemplate) StringID() string {
  h := sha1.New()
  fmt.Fprintf(h, "%s", st.Template)
  fmt.Fprintf(h, "%s", st.Variables)
  return fmt.Sprintf("%x", h.Sum(nil))
}

func serveApiSave(w http.ResponseWriter, r *http.Request) {
  st, err := createSavedTemplate(r)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  c := appengine.NewContext(r)
  key := datastore.NewKey(c, "SavedTemplate", st.StringID(), 0, nil)
  _, err = datastore.Put(
    c,
    key,
    st,
  )
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  fmt.Fprintf(w, "%s", key.StringID())

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

  fmt.Fprint(w, output)
}
