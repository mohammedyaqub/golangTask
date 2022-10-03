package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
)

const (
	GetItem = iota
	SetItem
	IncItem
	DecItem
)

type ServerStruct struct {
	orderStructChan chan<- OrderStruct
}
type OrderStruct struct {
	order_Id       int
	customerName   string
	order_Quantity int
	responseChan   chan int
}

// orderManager starts a goroutine that serves as a manager for
// orders. Returns a channel that's used to send orders to the
// manager.
func orderManager(m map[string]int) chan<- OrderStruct {
	//concurrent write to map does not allow so makes sure to protect order quantity inc or dec from overwrite
	//by using channels like input
	orderMap := make(map[string]int)
	for k, v := range m {
		orderMap[k] = v
	}
	//This is used as returning
	orders := make(chan OrderStruct)
	//concuurently running this for an order operations on item
	go func() {
		for order := range orders {
			switch order.order_Id {
			case GetItem:
				if val, ok := orderMap[order.customerName]; ok {
					order.responseChan <- val
				} else {
					order.responseChan <- -1
				}
			case SetItem:
				orderMap[order.customerName] = order.order_Quantity
				order.responseChan <- order.order_Quantity
			case IncItem:
				if _, ok := orderMap[order.customerName]; ok {
					//increment the order_Quantity by finding through the customer name as key in orderMap
					orderMap[order.customerName]++
					order.responseChan <- orderMap[order.customerName]
				} else {
					order.responseChan <- -1
				}
			case DecItem:
				if _, ok := orderMap[order.customerName]; ok {
					orderMap[order.customerName]--
					order.responseChan <- orderMap[order.customerName]
				} else {
					order.responseChan <- -1
				}
			default:
				log.Fatal("wrong order with id", order.order_Id)
			}
		}
	}()
	return orders
}

func (s *ServerStruct) get(w http.ResponseWriter, req *http.Request) {

	name := req.URL.Query().Get("Customername")
	replyChan := make(chan int)
	s.orderStructChan <- OrderStruct{order_Id: GetItem, customerName: name, responseChan: replyChan}
	reply := <-replyChan

	if reply > 0 {
		fmt.Fprintf(w, "%s order with the quantity : %d\n", name, reply)
	} else {
		fmt.Fprintf(w, "%s not found\n", name)
	}
}

func (s *ServerStruct) set(w http.ResponseWriter, req *http.Request) {

	name := req.URL.Query().Get("Customername")
	val := req.URL.Query().Get("OrderQuantity")
	intval, err := strconv.Atoi(val)
	if err != nil {
		fmt.Fprintf(w, "%s\n", err)
	} else {
		inputChan := make(chan int)
		s.orderStructChan <- OrderStruct{order_Id: SetItem, customerName: name, order_Quantity: intval, responseChan: inputChan}
		//no more interest or discard the received value which has been written reponseChan
		_ = <-inputChan

		fmt.Fprintf(w, " %s successfully set the order with quantity %d \n", name, intval)
	}
}
func (s *ServerStruct) dec(w http.ResponseWriter, req *http.Request) {
	//dynamic input taking from URL as json is not recommended
	name := req.URL.Query().Get("Customername")
	inputChan := make(chan int)
	s.orderStructChan <- OrderStruct{order_Id: DecItem, customerName: name, responseChan: inputChan}
	//reponse from order manager which has been wriiten to responseChan
	reply := <-inputChan
	if reply >= 0 {
		fmt.Fprintf(w, "succesfull decrement the item count for customer %s\n", name)
	} else {
		fmt.Fprintf(w, "%s customer not found \n", name)
	}
}

func (s *ServerStruct) inc(w http.ResponseWriter, req *http.Request) {

	name := req.URL.Query().Get("Customername")
	inputChan := make(chan int)
	s.orderStructChan <- OrderStruct{order_Id: IncItem, customerName: name, responseChan: inputChan}
	//reponse from order manager which has been wriiten to responseChan
	reply := <-inputChan
	if reply >= 0 {
		fmt.Fprintf(w, "succesfull increment the item count for customer %s\n", name)
	} else {
		fmt.Fprintf(w, "%s  customer not found \n", name)
	}
}

func main() {
	orderApi := ServerStruct{orderManager(map[string]int{})}

	//go http.HandleFunc("/get", server.get) http handlers automatically run concurrently no need go keyword
	// here name=customer name and val is order quantity
	//need to pass customer name and how much quantity
	//http://localhost:8080/set?Customername=x&OrderQuantity=1
	//http://localhost:8080/inc?Customername=xyz --> inc item OrderQuantity count
	//http://localhost:8080/dec?Customername=xyz --> dec item OrderQuantity count
	//http://localhost:8080/get?Customername=xyz --> get item OrderQuantity count
	http.HandleFunc("/get", orderApi.get)
	//set order by name and quantity
	http.HandleFunc("/set", orderApi.set)
	//increment order quantity by customer name
	http.HandleFunc("/inc", orderApi.inc)
	//decrement order quantity by customer name
	http.HandleFunc("/dec", orderApi.dec)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

/*
channel usage like
func AddOne(ch chan<- int, i int) {
	i++
	ch <- i
}

func MulBy10(ch <-chan int, resch chan<- int) {
	i := <-ch
	i *= 10
	resch <- i
}

func main() {
	ch := make(chan int)
	resch := make(chan int)

	go AddOne(ch, 9)
	go MulBy10(ch, resch)

	result := <-resch
	fmt.Println("Result:", result)
}*/
