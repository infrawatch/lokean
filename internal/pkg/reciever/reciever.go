package reciever

type Reciever interface {
    GetNotifier() chan string
    GetStatus() chan int
    GetDoneChan() chan bool
    Close()
}
