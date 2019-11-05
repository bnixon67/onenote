package main

import "fmt"

const MAX = 20

func main() {
    sem := make(chan int, MAX)
    for {
        sem <- 1 // will block if there is MAX ints in sem
        go func() {
            fmt.Println("hello again, world")
            <-sem // removes an int from sem, allowing another to proceed
        }()
    }
}
