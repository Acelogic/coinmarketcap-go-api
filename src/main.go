package main

import(
	"net/http"
	"github.com/gorilla/mux"
	"github.com/jamespearly/loggly"
	"time"
	"fmt"
	
)

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	// Initalize Loggly Client
	client := loggly.New("CoinServer")
	
	// Get remote address
	ip := r.RemoteAddr

	//If response is get
	if r.Method == "GET" {	
		w.Header().Set("Content-Type", "application/json")
		//Set status to 200
		w.WriteHeader(200)
		// Write the current time
		t := time.Now().Format(time.UnixDate)
		w.Write([]byte(`{"status": "ok", "time": "` + t + `"}`))

		// Log the response
		err := client.EchoSend("info", "Accessed From: " + ip + " | Method Type: " + r.Method + " | EndPoint: /status" + " | Status Code: 200")
		fmt.Println("err", err)
	} else {
		w.WriteHeader(405)
		w.Write([]byte(`{"status": "error", "message": "Method not allowed"}`))
		// log the response
		err := client.EchoSend("error", "Accessed From: " + ip + " | Method Type: " + r.Method + " | EndPoint: /status" + " | Status Code: 405")
		fmt.Println("err", err)
		
	}
}

func main() {
	portHttp := ":8080"
	fmt.Println("Server is running on port" + portHttp)
	r := mux.NewRouter()
	r.HandleFunc("/status", StatusHandler)
	err := http.ListenAndServe(portHttp, r)
	fmt.Println("err", err)


	

}