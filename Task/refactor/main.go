package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"

	//"strconv"
	"unicode"

	"gopkg.in/go-playground/validator.v9"
)

var (
	GetItem = 1
	SetItem = 2
	IncItem = 3
	DecItem = 4
)

type OrderStruct struct {
	Order_Id       int      `json:"id"`
	CustomerName   string   `json:"name"`
	Order_Quantity int      `json:"quantity"`
	responseChan   chan int //not exported member
}

/* type OrderStruct struct {
	Order_Id       int      `json:"id"`
	CustomerName   string   `json:"name" binding:"required,alphanumunicode"`
	Order_Quantity int      `json:"quantity" binding:"required,numeric"`
	responseChan   chan int //not exported member
} */
type ServerStruct struct {
	orderStructChan chan<- OrderStruct
}
type Users struct {
	//Id       int    `json:"id"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,gte=8,containsany=! @ # $ % ^ & * ( ) - +"`
	Phone    int64  `json:"phone" binding:"required,len=10,numeric"`
	Name     string `json:"name" binding:"required,alphanumunicode"`
}

func connectDatabase() *sql.DB {
	fmt.Println("Getting connected")

	db, err := sql.Open("mysql", "root:<password>@tcp(127.0.0.1:3306)<dbname>")
	//defer db.Close()wrong way of doing must do after checking the err
	if err != nil {

		panic(err.Error())

	}
	//defer db.Close() //correct way of doing
	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}
	fmt.Println("data is reachable now")
	return db
}
func validPassword(s string) bool {
	var (
		hasLen     = false
		hasUpper   = false
		haslower   = false
		hasspecial = false
		hasNum     = false
	)
	//first condition
	if len(s) >= 8 {
		hasLen = true
	}
	
	for _, char := range s {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			haslower = true
		case unicode.IsDigit(char):
			hasNum = true
		case unicode.IsSymbol(char) || unicode.IsPunct(char):
			hasspecial = true
		}
	}
	return hasUpper && haslower && hasNum && hasLen && hasspecial

}

var validate *validator.Validate

//register a new user and add to the database
func register(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("endpoint hit: adding new user to database ")
	db := connectDatabase()
	defer db.Close()

	//	defer db.Close()
	requestBody, _ := ioutil.ReadAll(r.Body)
	var person Users
	json.Unmarshal(requestBody, &person)
	//	err := person.validateUserInput()
	validate = validator.New()
	// validate := validator.New()
	err := validate.Struct(person)

	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			fmt.Println(err)
			return
		}

		fmt.Println("------ List of tag fields with error ---------")

		for _, err := range err.(validator.ValidationErrors) {
			fmt.Println(err.StructField())
			fmt.Println(err.ActualTag())
			fmt.Println(err.Kind())
			fmt.Println(err.Value())
			fmt.Println(err.Param())
			fmt.Println("---------------")
		}
		return
	}
	if validPassword(person.Password) {
		if r.Method == "POST" {

			//id := person.Id
			name := person.Name
			email := person.Email
			phone := person.Phone
			password := person.Password
			q, err := db.Prepare("INSERT INTO login.register(email,password,phone,name ) VALUES(?,?,?,?)")

			if err != nil {
				panic(err.Error())
			}
			fmt.Println(email, password, phone, name)
			q.Exec(email, password, phone, name)

		}
		fmt.Fprintf(w, "successfull registered\n")
	} else {
		fmt.Fprintf(w, "Password criteria is not matched\n")
		fmt.Fprintf(w, "length 8 one digit,one lowercase,one uppercase,one special letter")
		return
	}
	json.NewEncoder(w).Encode(person) //return back response this will print out the added up data to the client
	//db.Close()

}

//validate the user with provided email if exists return success else failure
func loginUser(w http.ResponseWriter, r *http.Request) {
	//	fmt.Println("endpoint hit: checking whether user present in database ")
	db := connectDatabase()
	defer db.Close()
	requestBody, _ := ioutil.ReadAll(r.Body)
	var person Users
	json.Unmarshal(requestBody, &person)
	if r.Method == "POST" {
		data, err := db.Query("select * from login.register")
		if err != nil {
			panic(err.Error())
		}

		res := []Users{}
		for data.Next() {
			for data.Next() {
				var user Users
				data.Scan(&user.Email, &user.Password, &user.Phone, &user.Name)
				//		fmt.Println(user.Email, user.Password, user.Phone, user.Name)
				res = append(res, user)
			}
		}
		//fmt.Println("retrive data from database is ", res)
		for _, item := range res {
			if item.Email == person.Email && item.Password == person.Password {
				fmt.Fprintf(w, "successfull login")
				return
			} //else {
			//fmt.Fprintf(w, "wrong password or email ")
			//	}
		}
		fmt.Fprintln(w, "wrong password or email ")
	}
}

func (s *ServerStruct) GetAllOrders(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint Hit: Total orders")
	fmt.Println("Getting the data from the database")

	db := connectDatabase()

	defer db.Close()
	data, err := db.Query("SELECT * FROM orders.ORDERS")
	if err != nil {
		panic(err.Error())
	}
	//var res []balance
	res := []OrderStruct{}
	for data.Next() {
		for data.Next() {
			var database OrderStruct
			//	for _,item:= range database {mistake here
			data.Scan(&database.Order_Id, &database.CustomerName, &database.Order_Quantity) //, &database.Dueto, &database.Reason)
			fmt.Println(database.Order_Id, database.CustomerName, database.Order_Quantity)  //, database.Dueto, database.Reason)
			res = append(res, database)
		}
	}

	json.NewEncoder(w).Encode(res)
}

// OrderManager starts a goroutine that serves as a manager for
// orders. Returns a channel that's used to send orders to the
// manager.
func OrderManager(m map[string]int) chan<- OrderStruct {
	//concurrent write to map does not allow so makes sure to protect order quantity inc or dec from overwrite
	//by using channels like input
	orderMap := make(map[string]int)
	for k, v := range m {
		orderMap[k] = v
	}
	//This is used as returning
	orders := make(chan OrderStruct)
	//concurrently running this for an order operations on item
	go func() {
		for order := range orders {
			switch order.Order_Id {
			case GetItem:
				if val, ok := orderMap[order.CustomerName]; ok {
					order.responseChan <- val
				} else {
					order.responseChan <- -1
				}
			case SetItem:
				orderMap[order.CustomerName] = order.Order_Quantity
				order.responseChan <- order.Order_Quantity
			case IncItem:
				if _, ok := orderMap[order.CustomerName]; ok {
					//increment the order_Quantity by finding through the customer name as key in orderMap
					orderMap[order.CustomerName]++
					order.responseChan <- orderMap[order.CustomerName]
				} else {
					order.responseChan <- -1
				}
			case DecItem:
				if _, ok := orderMap[order.CustomerName]; ok {
					orderMap[order.CustomerName]--
					order.responseChan <- orderMap[order.CustomerName]
				} else {
					order.responseChan <- -1
				}
			default:
				log.Fatal("wrong order with id", order.Order_Id)
			}
		}
	}()
	return orders
}

func (s *ServerStruct) GetOrderbyId(w http.ResponseWriter, r *http.Request) {
	db := connectDatabase()
	defer db.Close()
	/* vars := mux.Vars(r)
	nId := vars["id"]
	name := vars["name"] */
	name := r.URL.Query().Get("Customername")
	selDB, err := db.Query("SELECT * from orders.ORDERS  WHERE CustomerName = ?", name)
	if err != nil {
		panic(err.Error())
	}
	per := OrderStruct{}
	for selDB.Next() {
		var id int
		var name string
		var c int
		err = selDB.Scan(&id, &name, &c)
		if err != nil {
			panic(err.Error())
		}
		per.Order_Id = id
		per.CustomerName = name
		per.Order_Quantity = c

	}
	fmt.Println(per)
	json.NewEncoder(w).Encode(per)
	replyChan := make(chan int)
	s.orderStructChan <- OrderStruct{Order_Id: GetItem, CustomerName: name, responseChan: replyChan}
	reply := <-replyChan

	if reply > 0 {
		fmt.Printf("%s found in server process only : %d\n", name, reply)
	} else {
		fmt.Fprintf(w, "%s found into database successfully \n", name)
	}
}

func (s *ServerStruct) NewOrder(w http.ResponseWriter, r *http.Request) {
	fmt.Println("endpoint hit: adding up to database ")
	db := connectDatabase()
	defer db.Close()
	requestBody, _ := ioutil.ReadAll(r.Body)
	var person OrderStruct
	json.Unmarshal(requestBody, &person)
	if r.Method == "POST" {

		//id := person.Order_Id
		CustomerName := person.CustomerName
		OrderQuantity := person.Order_Quantity
		insForm, err := db.Prepare("INSERT INTO orders.ORDERS(CustomerName,OrderQuantity ) VALUES(?,?)")

		if err != nil {
			panic(err.Error())
		}
		inputChan := make(chan int)
		s.orderStructChan <- OrderStruct{Order_Id: SetItem, CustomerName: CustomerName, Order_Quantity: OrderQuantity, responseChan: inputChan}
		//no more interest or discard the received value which has been written reponseChan
		_ = <-inputChan
		//fmt.Println(CustomerName, OrderQuantity)
		insForm.Exec(CustomerName, OrderQuantity)

		fmt.Fprintf(w, " %s successfully set the order with quantity %d \n", CustomerName, OrderQuantity)
		json.NewEncoder(w).Encode(person) //return back response this will print out the added up data
	}
}

//takes inputs from the url itself and perform updation to the database
func (s *ServerStruct) DecreamentItemCountby1(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("hit delete endpoint ")
	name := r.URL.Query().Get("Customername")
	db := connectDatabase()
	defer db.Close()
	del, err := db.Prepare("UPDATE ORDERS SET OrderQuantity = OrderQuantity -1  WHERE CustomerName = ?")
	if err != nil {
		panic(err.Error())
	}
	del.Exec(name)
	//fmt.Println("deleted succesfully")
	inputChan := make(chan int)
	s.orderStructChan <- OrderStruct{Order_Id: DecItem, CustomerName: name, responseChan: inputChan}
	//reponse from order manager which has been wriiten to responseChan
	reply := <-inputChan
	if reply >= 0 {
		fmt.Printf("succesfull decrement the item count for customer in server process %s\n", name)
	} else {
		fmt.Fprintf(w, "succesfull decrement the item count for customer and updated database for %s\n", name)
	}
}

func (s *ServerStruct) IncrementItemCountby1(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("Customername")
	db := connectDatabase()
	defer db.Close()
	//vars := mux.Vars(r)
	//fmt.Println("into the inc")
	//	id := vars["Order_Id"]
	/*int_id, err := strconv.Atoi(id)
	  if err != nil {
	      panic(err)
	  }*/

	//name := vars["CustomerName"]
	//fmt.Printf("updated order for  %s  ", name)
	myupdate, err := db.Prepare("UPDATE ORDERS SET OrderQuantity = OrderQuantity +1  WHERE CustomerName = ?")
	if err != nil {
		panic(err.Error())
	}
	//defer myupdate.Close()
	myupdate.Exec(name)
//	fmt.Println("succesfully updated increment order quantity into db")
	inputChan := make(chan int)
	s.orderStructChan <- OrderStruct{Order_Id: IncItem, CustomerName: name, responseChan: inputChan}
	//reponse from order manager which has been wriiten to responseChan
	reply := <-inputChan
	if reply >= 0 {
		fmt.Printf("succesfull increment the item count for customer at process only %s\n", name)
	} else {
		fmt.Fprintf(w, "succesfull increment the item count for customer and updated database for %s\n", name)
	}
}

/* 1.  Not able to get all list Items
    sol:- need a database to store all the orders in table and read the stored data from it using GET.
    addressed by  getallorders
2 . Please make Inc Call as POST Call, that would accept the json array as input. */
//addressed by r.HandleFunc("/neworder", orderApi.NewOrder).Methods("POST")

func main() {
	orderApi := ServerStruct{OrderManager(map[string]int{})}

	//go http.HandleFunc("/get", server.get) http handlers automatically run concurrently no need go keyword
	r := mux.NewRouter()
	//first do register into app
	r.HandleFunc("/register", register).Methods("POST")
	r.HandleFunc("/login", loginUser).Methods("POST")
	r.HandleFunc("/neworder", orderApi.NewOrder).Methods("POST")
	r.HandleFunc("/dec_item/", orderApi.DecreamentItemCountby1).Methods("PUT")
	r.HandleFunc("/inc_item/", orderApi.IncrementItemCountby1).Methods("PUT")
	r.HandleFunc("/getallorders", orderApi.GetAllOrders).Methods("GET")
	r.HandleFunc("/getorderbyname/", orderApi.GetOrderbyId).Methods("GET")
	log.Fatal(http.ListenAndServe(":8080", r))
	//log.Fatal(http.ListenAndServe(":8080", nil))
}
