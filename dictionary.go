package main

type Dictionary interface {
	LookUp(name string) string
}
