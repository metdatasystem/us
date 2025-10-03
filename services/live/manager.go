package main

type Manager interface {
	Load() error
	Run()
	Subscribe(*client)
	Unsubscribe(*client)
}
