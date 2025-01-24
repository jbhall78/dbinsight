package main

import (
    "html/template"
    "net/http"
)

type PageData struct {
    Title   string
    Heading string
    Items   []string
}

func handler(w http.ResponseWriter, r *http.Request) {
    data := PageData{
        Title:   "My Page",
        Heading: "Welcome!",
        Items:   []string{"Item 1", "Item 2", "Item 3"},
    }

    tmpl, err := template.ParseFiles("templates/index.html") // Parse the template file
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    err = tmpl.Execute(w, data) // Execute the template with the data
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func main() {
    http.HandleFunc("/", handler)
    http.ListenAndServe(":8080", nil)
}

