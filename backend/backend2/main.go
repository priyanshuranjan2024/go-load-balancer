package main

import "go-load-balancer/backend"

func main() {
    backend.Start("9002")
}