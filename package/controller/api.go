package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/yusuf/bookwiseAPI/model"
)

// SearchForBook : this method searched for requested books using OpenLibrary API and
// storing the response data in a proper json format for use.
func (ct *Catalogue) SearchForBook(wr http.ResponseWriter, rq *http.Request) {
	wr.Header().Set("Content-Type", "application/json")

	var docs model.Docs

	// parse the form input
	if err := rq.ParseForm(); err != nil {
		ct.App.ErrorLogger.Fatalf("cannot parse post info :  %v \n", err)
	}

	// remove redundant space from the input values
	title := strings.ToLower(rq.PostForm.Get("title"))
	searchBook := strings.Replace(strings.TrimSpace(title), " ", "+", -1)

	// using the http client to get result/data from the API url
	resp, err := ct.Client.Get(fmt.Sprintf("https://openlibrary.org/search.json?q=%v", searchBook))
	if err != nil {
		ct.App.ErrorLogger.Fatalf("url error : %v \n", err)
	}

	// close/stop reading the response body from the request URL
	// after this function is done executing
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			ct.App.ErrorLogger.Fatalln(err)
		}
	}(resp.Body)

	// Read from the response body of the API request
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		ct.App.ErrorLogger.Println(err)
	}

	apiData := string(data)

	// write the data read from the body to json file (api.json)
	err = os.WriteFile("./package/json/api.json", []byte(apiData), 0o666)
	if err != nil {
		ct.App.ErrorLogger.Println(err)
	}

	// Open the JSON file and check for any error that might occur
	bookData, err := os.Open("./package/json/api.json")
	if err != nil {
		ct.App.ErrorLogger.Println(err)
	}

	// close/stop reading data from the json file after this function is done executing
	defer func(bookData *os.File) {
		err := bookData.Close()
		if err != nil {
			ct.App.ErrorLogger.Println(err)
			return
		}
	}(bookData)

	// Read all the data stored in the JSON file as bytes of data
	byteData, err := io.ReadAll(bookData)
	if err != nil {
		ct.App.ErrorLogger.Println(err)
	}

	// Encode the data into struct model format similar to the JSON format
	// Note : the data is the details of the searched book
	err = json.Unmarshal(byteData, &docs)
	if err != nil {
		ct.App.ErrorLogger.Println(err)
	}

	book := docs.Docs[0]

	// Check if Book is in the Library if not add the book to the library collections
	count, bookID, err := ct.CatDB.CheckLibrary(book.Title, book)
	if err != nil {
		ct.App.ErrorLogger.Fatalln("error while checking for book in the library")
	}
	scs := ct.App.Session.Start(wr, rq)
	scs.Set("book_id", bookID)
	scs.Set("book_title", book.Title)

	// conditions : check if the searched book is available in the Main Library/ Store
	// , not available or if an error pop up in the server
	if count >= 1 {
		msg := map[string]interface{}{
			"status_code": http.StatusOK,
			"message":     fmt.Sprintf("%v : Book Found in Library", book.Title),
			"data":        book,
		}
		jsonData, err := json.MarshalIndent(msg, " ", "   ")
		if err != nil {
			return
		}
		_, err = wr.Write(jsonData)
		if err != nil {
			return
		}
	} else if count == 0 {

		msg := map[string]interface{}{
			"status_code": http.StatusOK,
			"message":     fmt.Sprintf(" %v : Book not found in Library!", book.Title),
			"data":        "Adding New Book to Library .... Search again",
		}
		jsonData, err := json.MarshalIndent(msg, " ", "   ")
		if err != nil {
			return
		}
		_, err = wr.Write(jsonData)
		if err != nil {
			return
		}
	} else {
		msg := map[string]interface{}{
			"status_code": http.StatusInternalServerError,
			"message":     "Error While Searching For Book",
		}
		jsonData, err := json.MarshalIndent(msg, " ", "   ")
		if err != nil {
			return
		}
		_, err = wr.Write(jsonData)
		if err != nil {
			return
		}
	}
}
