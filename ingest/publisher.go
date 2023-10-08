package main

type Publisher interface {
	Publish(message []byte) error
}
